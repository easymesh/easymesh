package route

import (
	"encoding/json"
	"github.com/astaxie/beego/logs"
	"github.com/lixiangyun/easymesh/util/ip"
	"net"
	"sync"
	"time"
)

type UDP_TYPE int

const (
	_ UDP_TYPE = iota
	UDP_TRANSFER_T
	UDP_THROUGH_T
	UDP_LOCALADD_T
)

type UdpAddr struct {
	Typ  UDP_TYPE
	Udp  net.UDPAddr

	used int
	timestamp time.Time
}

type Route struct {
	IP  ip.IP4
	Udp []UdpAddr

	timestamp time.Time
}

func NewUdpAddr(typ UDP_TYPE, addr net.UDPAddr) UdpAddr {
	return UdpAddr{Typ: typ, Udp: addr, timestamp: time.Now()}
}

func (u *UdpAddr)Usability() int {
	return u.used
}

func (u *UdpAddr)UsabilitySet(used int)  {
	u.used = used
}

func NewRoute(ipAddr string, udpAddr UdpAddr) *Route {
	tmNow := time.Now()

	ips, err := ip.ParseIP4(ipAddr)
	if err != nil {
		logs.Error(err.Error())
		return nil
	}
	udpAddr.timestamp = tmNow
	return &Route{IP: ips, Udp: []UdpAddr{udpAddr}, timestamp: tmNow}
}

func (r *Route)Usability(dst *net.UDPAddr)  {
	for i, _ := range r.Udp {
		if r.Udp[i].Udp.String() == dst.String() {
			r.Udp[i].timestamp = time.Now()
			r.Udp[i].UsabilitySet(1)

			logs.Info("%s udp addr usability %s", r.IP, dst.String())
			return
		}
	}
	logs.Error("can not find udp addr", dst.String())
}

func (r *Route)SyncAddr(newList []UdpAddr)  {
	udps := make([]UdpAddr, len(newList))
	for i, newUdp := range newList {
		for _, oldUdp := range r.Udp {
			if newUdp.Typ == oldUdp.Typ && newUdp.Udp.String() == oldUdp.Udp.String() {
				newUdp.UsabilitySet(oldUdp.Usability())
				newUdp.timestamp = oldUdp.timestamp
			}
		}
		udps[i] = newUdp
	}
	r.Udp = udps
}

func (r *Route)Clone() *Route {
	tmNow := time.Now()

	cp := &Route{IP: r.IP, timestamp: tmNow}
	cp.Udp = make([]UdpAddr, len(r.Udp))
	copy(cp.Udp, r.Udp)

	for i, _ := range cp.Udp {
		cp.Udp[i].timestamp = tmNow
		cp.Udp[i].used = 0
	}
	return cp
}

func (r *Route)TransferUdpAddr() *UdpAddr {
	for _, v := range r.Udp {
		if v.Typ == UDP_TRANSFER_T {
			return &v
		}
	}
	return nil
}

func (r *Route)ThroughUdpAddr() *UdpAddr {
	for _, v := range r.Udp {
		if v.Typ == UDP_THROUGH_T {
			return &v
		}
	}
	return nil
}

func (r *Route)LocalUdpAddr() *UdpAddr {
	for _, v := range r.Udp {
		if v.Typ == UDP_LOCALADD_T {
			return &v
		}
	}
	return nil
}

func RouteDecoder(body []byte) *Route {
	route := new(Route)
	return route.Decoder(body)
}

func RouteCoder(r *Route) []byte {
	return r.Coder()
}

func (r *Route)String() string {
	return string(r.Coder())
}

func (r *Route)Coder() []byte {
	body, err := json.Marshal(r)
	if err != nil {
		logs.Error(err.Error())
		return nil
	}
	return body
}

func (r *Route)Decoder(body []byte) *Route {
	err := json.Unmarshal(body, r)
	if err != nil {
		logs.Error("json unmarshal fail", string(body), err.Error())
		return nil
	}
	return r
}

type RouteCtrl struct {
	sync.RWMutex
	drop time.Duration
	udp  time.Duration
	list map[ip.IP4]*Route 
}

func NewRouteCtrl(dropTime time.Duration, udpTime time.Duration) *RouteCtrl {
	routes := &RouteCtrl{list: make(map[ip.IP4]*Route, 1024), drop: dropTime, udp: udpTime}
	go func() {
		ticker := time.NewTicker(5*time.Second)
		defer ticker.Stop()

		for  {
			<- ticker.C
			routes.timeoutDrop()
		}
	}()
	return routes
}

func (routes *RouteCtrl)timeoutDrop()  {
	routes.Lock()
	defer routes.Unlock()

	now := time.Now()
	for _, v := range routes.list {
		for i, _ := range v.Udp {
			if v.Udp[i].used > 0 && now.Sub(v.Udp[i].timestamp) > routes.udp {
				v.Udp[i].used = 0

				logs.Error("timeout drop udp addr", v.Udp[i].Udp.String())
			}
		}

		if now.Sub(v.timestamp) > routes.drop {
			delete(routes.list, v.IP)

			logs.Error("timeout drop route", v.String())
		}
	}
}

func (routes *RouteCtrl)Sync(r Route)  {
	routes.Lock()
	defer routes.Unlock()

	routes.updateRoute(r)
}

func (routes *RouteCtrl)updateRoute(r Route)  {
	oldRoute, _ := routes.list[r.IP]
	if oldRoute != nil {
		oldRoute.timestamp = time.Now()
		oldRoute.SyncAddr(r.Udp)
	} else {
		routes.list[r.IP] = r.Clone()
	}
}

func (routes *RouteCtrl)SyncBatch(r []Route)  {
	routes.Lock()
	defer routes.Unlock()

	for _, v := range r {
		routes.updateRoute(v)
	}
}

func (routes *RouteCtrl)Route(ip4 ip.IP4) *Route {
	routes.RLock()
	defer routes.RUnlock()

	r, _ := routes.list[ip4]
	return r
}

func (routes *RouteCtrl)Export() RouteList {
	routes.RLock()
	defer routes.RUnlock()

	routeList := make([]Route, 0)
	for _, v := range routes.list {
		routeList = append(routeList, *v)
	}
	return routeList
}

type RouteList []Route

func RouteListDecoder(body []byte) RouteList {
	var list []Route
	err := json.Unmarshal(body, &list)
	if err != nil {
		logs.Error("json unmarshal fail", string(body), err.Error())
		return nil
	}
	return list
}

func (r RouteList)Coder() []byte {
	body, err := json.Marshal(r)
	if err != nil {
		logs.Error(err.Error())
		return nil
	}
	return body
}

func (r RouteList)String() string {
	return string(r.Coder())
}

