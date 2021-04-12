package main

import (
	"flag"
	"fmt"
	"github.com/astaxie/beego/logs"
	"github.com/easymesh/easymesh/route"
	"github.com/easymesh/easymesh/util"
	"github.com/easymesh/easymesh/util/ip"
	"github.com/easymesh/easymesh/util/udp"
	"net"
	"os"
	"time"
)

type Transfer struct {
	routeCtl   *route.RouteCtrl
	transAddr  *net.UDPAddr
	udpSocket  *net.UDPConn
	oAddr       ip.IP4
}


func (t *Transfer)TransferIP(conn *net.UDPConn, oAddr ip.IP4, srcAddr *net.UDPAddr, buff []byte)  {
	if len(buff) < ip.MAX_IPHEADER {
		logs.Error("udp socket recv length too smail", len(buff))
		return
	}

	ip4hdr := ip.IP4HeaderDecoder(buff[:ip.MAX_IPHEADER])
	var sendBody []byte
	var err error

	dstAddr := t.findRoute(ip4hdr.DAddr)
	if dstAddr == nil {
		sendBody, err = ip.ICMPUnreachable(oAddr, ip4hdr, buff[:])
		if err != nil {
			logs.Error("ipv4 send unreachable fail", ip4hdr.String(), err.Error())
			return
		}
		dstAddr = srcAddr
	} else {
		err = ip4hdr.DecrementTTL()
		if err != nil {
			logs.Error("ipv4 ttl is zero", ip4hdr.String(), err.Error())
			return
		}
		ip4hdr.Coder(buff[:ip.MAX_IPHEADER])
		sendBody = buff
	}

	err = udp.UdpWrite(conn, dstAddr, sendBody)
	if err != nil {
		logs.Error("udp send fail", err.Error())
	}
}

func (t *Transfer)UdpRecvTask(conn *net.UDPConn, oAddr ip.IP4)  {
	buff := make([]byte, 8192 )
	for  {
		cnt, srcAddr, err := conn.ReadFromUDP(buff)
		if err != nil {
			logs.Error(err.Error())
			continue
		}

		if cnt < 1 {
			logs.Error("recv bad body: %d, %s", cnt, srcAddr.String())
			continue
		}

		pktType := ip.IPHeaderType(buff[0])
		if pktType == ip.IPv4 {
			t.TransferIP(conn, oAddr, srcAddr, buff[:cnt])
			continue
		}

		if pktType == ip.IPCtrl {
			t.syncRoute(conn, srcAddr, buff[1:cnt])
			continue
		}
	}
}

func NewTransfer(port int, pubip string) *Transfer {
	var err error

	trans := new(Transfer)
	trans.routeCtl = route.NewRouteCtrl(time.Minute, time.Minute)

	trans.udpSocket, err = udp.OpenUdp(fmt.Sprintf(":%d", port))
	if err != nil {
		logs.Error(err.Error())
		return nil
	}

	publicIP, err := net.ResolveIPAddr("ip4", pubip)
	if err != nil {
		logs.Error(err.Error())
		return nil
	}
	trans.oAddr = ip.FromIP(publicIP.IP)

	trans.transAddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", pubip, port))
	if err != nil {
		logs.Error(err.Error())
		return nil
	}

	go trans.UdpRecvTask(trans.udpSocket, trans.oAddr)

	return trans
}

func transferOwner(r *route.Route, addr *net.UDPAddr) bool {
	for _, v:= range r.Udp {
		if v.Typ == route.UDP_TRANSFER_T &&
			v.Udp.String() == addr.String() {
			return true
		}
	}
	return false
}

func (t *Transfer)String() string {
	return t.transAddr.String()
}

func (t *Transfer)findRoute(ip4 ip.IP4) *net.UDPAddr {
	var udpAddr *route.UdpAddr

	r := t.routeCtl.Route(ip4)
	if r == nil {
		return nil
	}

	if transferOwner(r, t.transAddr) == true {
		udpAddr = r.ThroughUdpAddr()
	} else {
		udpAddr = r.TransferUdpAddr()
	}

	if udpAddr != nil {
		return &udpAddr.Udp
	}
	return nil
}

func (t *Transfer)syncRoute(conn *net.UDPConn, srcAddr *net.UDPAddr, body []byte)  {
	r := route.RouteDecoder(body)
	if r == nil {
		logs.Error("route decoder fail", string(body))
		return
	}

	through := route.NewUdpAddr(route.UDP_THROUGH_T, *srcAddr)
	transfer := route.NewUdpAddr(route.UDP_TRANSFER_T, *t.transAddr)

	r.Udp = append(r.Udp, through, transfer)
	t.routeCtl.Sync(*r)

	routelist := t.routeCtl.Export()
	output := routelist.Coder()

	logs.Info("[%s] sync route list %s\n", t.String(), string(output))

	err := udp.UdpWrite(conn, srcAddr, udp.UdpCtrl(output))
	if err != nil {
		logs.Error("sync route fail", err.Error())
	}
}

var (
	help   bool
	debug  bool

	BIND_PORT   int
	BIND_NUMS   int

	LOG_DIR     string
	PUB_ADDR    string
)

func init()  {
	flag.BoolVar(&help, "help", false, "usage")
	flag.BoolVar(&debug, "debug", false, "debug mode")
	flag.StringVar(&LOG_DIR, "log", "./", "log dir")
	flag.IntVar(&BIND_PORT, "bind", 8000, "transfer server bind port")
	flag.IntVar(&BIND_NUMS, "nums", 1000, "transfer server instance nums")
	flag.StringVar(&PUB_ADDR, "public", "www.domain.com", "public IP")
}

var transList []*Transfer

func main()  {
	flag.Parse()
	if help {
		flag.Usage()
		return
	}

	util.LogInit(LOG_DIR, debug,"transfer.log")

	for i := BIND_PORT ; i < (BIND_PORT + BIND_NUMS); i++ {
		temp := NewTransfer(i, PUB_ADDR)
		if temp != nil {
			transList = append(transList, temp)
		} else {
			logs.Error("bind port %d fail", i)
		}
	}

	util.WaitSignal(Shutdown)
}

func Shutdown(sig os.Signal)  {
	for _, v := range transList {
		v.udpSocket.Close()
	}
	os.Exit(-1)
}

