package ip

import (
	"fmt"
	"net"
	"strings"
)

func InterfaceByName(ifname string) (*net.Interface, error) {
	iface, err := net.InterfaceByName(ifname)
	if err != nil {
		return nil, fmt.Errorf("error looking up interface %s: %s",
			ifname, err.Error())
	}
	if iface.MTU == 0 {
		return nil, fmt.Errorf("failed to determine MTU for %s interface", ifname)
	}
	return iface, nil
}

func InterfaceByAddr(addr string) (*net.Interface, error) {
	ipaddr := net.ParseIP(addr)
	if ipaddr == nil {
		return nil, fmt.Errorf("address is invalid")
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, v := range ifaces {
		addrs, err := InterfaceAddsGet(&v)
		if err != nil {
			continue
		}
		for _, v2 := range addrs{
			if v2.Equal(ipaddr) {
				if v.MTU == 0 {
					return nil, fmt.Errorf("failed to determine MTU for %s interface", v.Name)
				}
				return &v, nil
			}
		}
	}

	return nil, fmt.Errorf("failed to find interface by address", addr)
}

func InterfaceAddsGet(iface *net.Interface) ([]net.IP, error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, nil
	}
	ips := make([]net.IP, 0)
	for _, v:= range addrs {
		ipone, _, err:= net.ParseCIDR(v.String())
		if err != nil {
			continue
		}
		if len(ipone) > 0 {
			ips = append(ips, ipone)
		}
	}
	return ips, nil
}

func IsIPString(ip string) bool {
	_, err := ParseIP4(ip)
	if err != nil {
		return false
	}
	return true
}

func IsIPv4(ip net.IP) bool {
	return strings.Index(ip.String(), ".") != -1
}
