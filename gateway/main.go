package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/astaxie/beego/logs"
	"github.com/easymesh/easymesh/route"
	"github.com/easymesh/easymesh/util"
	"github.com/easymesh/easymesh/util/ip"
	"github.com/easymesh/easymesh/util/tun"
	"github.com/easymesh/easymesh/util/udp"
	"net"
	"os"
	"time"
)

func SendUnreachable(tun tun.TunApi, tun_ip ip.IP4, off_iph *ip.IP4Header, offender []byte) error {
	body, err := ip.ICMPUnreachable(tun_ip, off_iph, offender)
	if err != nil {
		return err
	}
	err = tun.Write(body)
	if err != nil {
		return fmt.Errorf("send ICMP net unreachable to tun fail, %s", err.Error())
	}
	return nil
}

func TunRecvTask(tun tun.TunApi, conn *net.UDPConn)  {
	buff := make([]byte, 8192)
	for  {
		cnt, err := tun.Read(buff)
		if err != nil {
			logs.Error("tun read fail", err.Error())
			continue
		}

		if ip.IPHeaderType(buff[0]) != ip.IPv4 {
			continue
		}

		if cnt < ip.MAX_IPHEADER {
			logs.Error("tun read length too smail", cnt)
			continue
		}

		ip4hdr := ip.IP4HeaderDecoder(buff[:ip.MAX_IPHEADER])

		dstAddr := findRoute(ip4hdr.DAddr)
		if dstAddr == nil {
			err = SendUnreachable(tun, selfOverIP, ip4hdr, buff[:])
			if err != nil {
				logs.Error("send unreachable fail", ip4hdr.String(), err.Error())
			}
			continue
		}

		err = ip4hdr.DecrementTTL()
		if err != nil {
			logs.Warn("ipv4 packet ttl is zero", ip4hdr.String(), err.Error())
			continue
		}
		ip4hdr.Coder(buff[:ip.MAX_IPHEADER])

		err = udp.UdpWrite(conn, dstAddr, buff[:cnt])
		if err != nil {
			logs.Error("udp send fail", dstAddr.String(), err.Error())
		}
	}
}

func UdpRecvTask(conn *net.UDPConn, tun tun.TunApi)  {
	buff := make([]byte, 8192)
	for  {
		cnt, srcAddr, err := conn.ReadFromUDP(buff)
		if err != nil {
			logs.Error("udp socket read fail", err.Error())
			continue
		}

		if cnt < 1 {
			logs.Error("recv bad body: %d, %s", cnt, srcAddr.String())
			continue
		}

		pktType := ip.IPHeaderType(buff[0])
		if pktType == ip.IPv4 {
			if cnt < ip.MAX_IPHEADER {
				logs.Error("udp socket recv length too smail", cnt)
				continue
			}

			ip4hdr := ip.IP4HeaderDecoder(buff[:ip.MAX_IPHEADER])
			err = ip4hdr.DecrementTTL()
			if err != nil {
				logs.Warn("ipv4 packet ttl is zero", ip4hdr.String(), err.Error())
				continue
			}
			ip4hdr.Coder(buff[:ip.MAX_IPHEADER])

			err = tun.Write(buff[:cnt])
			if err != nil {
				logs.Error("udp to tun send fail", err.Error())
			}
		}

		if pktType == ip.IPCtrl {
			if srcAddr.String() != transAddr.String() {
				logs.Error("recv bad ctrl",
					srcAddr.String(), transAddr.String(), buff[:cnt])
			} else {
				SyncRoute(buff[1:cnt])
			}
		}

		if pktType == ip.Ping {
			ProcessPingPong(conn, srcAddr, buff[1:cnt])
		}
	}
}

var tunHandler tun.TunApi

func initTun(ipn ip.IP4Net) error {
	var err error
	tunHandler, err = tun.OpenTun(BIND_INFACE, ipn)
	if err != nil {
		return err
	}
	logs.Info("tun init success")
	return nil
}

var udpHander *net.UDPConn
func initUdp(bindAddr string) error {
	var err error
	udpHander, err = udp.OpenUdp(bindAddr)
	if err != nil {
		return err
	}
	return nil
}

var inface *net.Interface
var localUdpAddr  route.UdpAddr
var selfOverIP ip.IP4

func initIface() error {
	var err error
	var inface *net.Interface

	if ip.IsIPString(BIND_INFACE) {
		inface, err = ip.InterfaceByAddr(BIND_INFACE)
		if err != nil {
			return err
		}
	} else {
		inface, err = ip.InterfaceByName(BIND_INFACE)
		if err != nil {
			return err
		}
	}

	logs.Info("interface %s MTU: %d", inface.Name, inface.MTU)

	addrs, err := ip.InterfaceAddsGet(inface)
	if err != nil {
		return err
	}

	logs.Info("interfase %s address %s", inface.Name, addrs)

	for _, v := range addrs {
		if ip.IsIPv4(v) == false {
			continue
		}

		localAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", v, BIND_PORT))
		if err != nil {
			return err
		}

		logs.Info("local address", localAddr.String())

		localUdpAddr = route.NewUdpAddr(route.UDP_LOCALADD_T, *localAddr)
		return nil
	}

	return fmt.Errorf("interface init fail")
}

var routeCtrl *route.RouteCtrl
var transAddr *net.UDPAddr

func transferRetry(trans *net.UDPAddr) error {
	var buff [4096]byte

	udpconn, err := net.Dial("udp", trans.String())
	if err != nil {
		return err
	}

	for i := 0; i < 3; i++ {
		time.Sleep(5*time.Second)

		r := route.NewRoute(OVER_IP, localUdpAddr, TOKEN)
		_, err := udpconn.Write(udp.UdpCtrl(r.Coder()))
		if err != nil {
			logs.Error("udp write to transfer fail", err.Error())
			continue
		}

		udpconn.SetReadDeadline(time.Now().Add(5 * time.Second))
		cnt, err := udpconn.Read(buff[:])
		if err != nil {
			logs.Error("udp read fail", err.Error())
			continue
		}

		pktType := ip.IPHeaderType(buff[0])
		if pktType != ip.IPCtrl {
			logs.Error("parse fail")
			continue
		}

		err = SyncRoute(buff[1:cnt])
		if err != nil {
			logs.Error(err.Error())
			continue
		}

		return nil
	}
	return fmt.Errorf("transfer address not connect")
}

func init()  {
	routeCtrl = route.NewRouteCtrl(time.Minute, 30 * time.Second)
}

func initRoute() error {
	var err error
	transAddr, err = net.ResolveUDPAddr("udp", TRANS_ADDR)
	if err != nil {
		return err
	}
	logs.Info("%s reslove to %s", TRANS_ADDR, transAddr.String())
	err = transferRetry(transAddr)
	if err != nil {
		return err
	}
	logs.Info("transfer connect %s suucess", transAddr.String())
	return nil
}

func findRoute(ip4 ip.IP4) *net.UDPAddr {
	r := routeCtrl.Route(ip4)
	if r == nil {
		return nil
	}

	local := r.LocalUdpAddr()
	if local != nil && local.Usability() > 0 {
		return &local.Udp
	}

	through := r.ThroughUdpAddr()
	if through != nil && through.Usability() > 0 {
		return &through.Udp
	}

	return transAddr
}

func UpdateRoute()  {
	ticker := time.NewTicker(15*time.Second)
	for  {
		r := route.NewRoute(OVER_IP, localUdpAddr, TOKEN)

		logs.Info("update local route to transfer", r.String(), transAddr.String())

		err := udp.UdpWrite(udpHander, transAddr, udp.UdpCtrl(r.Coder()))
		if err != nil {
			logs.Error("udp send fail", err.Error())
		}

		<-ticker.C
	}
}

func SyncRoute(body []byte) error {
	routelist := route.RouteListDecoder(body)
	if len(routelist) == 0 {
		return fmt.Errorf("sync route from transfer fail")
	}
	routeCtrl.SyncBatch(routelist)
	logs.Info("sync route from transfer", routelist.String())
	return nil
}

const PING_TYPE = 0x111
const PONG_TYPE = 0x222

type TestPing struct {
	Type         int
	SerialNumber uint64
	Timestamp    time.Time
	FromIP       ip.IP4
	ToIP         ip.IP4
}

func ProcessPingPong(conn *net.UDPConn, srcAddr *net.UDPAddr, body []byte)  {
	test := ParsePing(body)

	if test.ToIP != selfOverIP {
		logs.Error("drop unkown ping/pong packet", string(body))
		return
	}

	if test.Type == PING_TYPE {
		output := BuildPing(PONG_TYPE, test.SerialNumber, test.FromIP)
		err := udp.UdpWrite(conn, srcAddr, udp.UdpPing(output))
		if err != nil {
			logs.Error("udp send ping/pong fail", err.Error())
		}
	}

	if test.Type == PONG_TYPE {
		r := routeCtrl.Route(test.FromIP)
		if r == nil {
			logs.Error("drop unkown ping/pong packet", string(body))
			return
		}
		r.Usability(srcAddr)
	}
}

func ParsePing(body []byte) *TestPing {
	ping := new(TestPing)
	err := json.Unmarshal(body, ping)
	if err != nil {
		logs.Error("parse ping fail", err.Error())
		return nil
	}
	return ping
}

func BuildPing(typ int, number uint64, toIP ip.IP4) []byte {
	ping := new(TestPing)
	ping.Type = typ
	ping.SerialNumber = number
	ping.Timestamp = time.Now()
	ping.FromIP = selfOverIP
	ping.ToIP = toIP

	body, err := json.Marshal(ping)
	if err != nil {
		logs.Error("build ping fail", ping, err.Error())
		return nil
	}
	return body
}

func RetryRoute()  {
	ticker := time.NewTicker(5*time.Second)
	for  {
		<-ticker.C

		routelist := routeCtrl.Export()
		for _, v := range routelist {
			if v.IP == selfOverIP {
				continue
			}
			pingBody := BuildPing(PING_TYPE, 0, v.IP)

			localAddr := v.LocalUdpAddr()
			if localAddr != nil {
				err := udp.UdpWrite(udpHander, &localAddr.Udp, udp.UdpPing(pingBody))
				if err != nil {
					logs.Error("udp send ping/pong fail", err.Error())
				}
			}

			throughAdr := v.ThroughUdpAddr()
			if throughAdr != nil {
				err := udp.UdpWrite(udpHander, &throughAdr.Udp, udp.UdpPing(pingBody))
				if err != nil {
					logs.Error("udp send ping/pong fail", err.Error())
				}
			}
		}
	}
}

var (
	help        bool
	debug       bool

	LOG_DIR     string
	TOKEN       string

	BIND_INFACE string
	OVER_IP     string

	TRANS_ADDR  string

	BIND_PORT int
)

func init()  {
	flag.BoolVar(&help, "help", false, "usage")
	flag.BoolVar(&debug, "debug", false, "debug mode")
	flag.StringVar(&LOG_DIR, "log", "./", "log dir")
	flag.StringVar(&TOKEN, "token", "", "access auth")
	flag.StringVar(&BIND_INFACE, "iface", "eth0", "interface or ip")
	flag.StringVar(&OVER_IP, "ip", "172.168.0.1", "virtual ip")
	flag.StringVar(&TRANS_ADDR, "trans", "www.domain.com:8000", "transfer public address")
}

func main()  {
	flag.Parse()
	if help || TOKEN == ""{
		flag.Usage()
		return
	}

	util.LogInit(LOG_DIR, debug,"gateway.log")

	BIND_PORT = udp.UnusedPort()

	err := initIface()
	if err != nil {
		logs.Error(err.Error())
		return
	}

	err = initRoute()
	if err != nil {
		logs.Error(err.Error())
		return
	}

	err = initUdp(fmt.Sprintf(":%d", BIND_PORT))
	if err != nil {
		logs.Error(err.Error())
		return
	}

	selfOverIP, err = ip.ParseIP4(OVER_IP)
	if err != nil {
		logs.Error(err.Error())
		return
	}

	ipnet, err := ip.NewIP4Net(OVER_IP, 16)
	if err != nil {
		logs.Error(err.Error())
		return
	}

	err = initTun(*ipnet)
	if err != nil {
		logs.Error(err.Error())
		return
	}

	for i:= 0 ; i < 10 ; i++ {
		go TunRecvTask(tunHandler, udpHander)
		go UdpRecvTask(udpHander, tunHandler)
	}

	go RetryRoute()
	go UpdateRoute()

	util.WaitSignal(Shutdown)
}

func Shutdown(sig os.Signal)  {
	tunHandler.Close()
	udpHander.Close()
}