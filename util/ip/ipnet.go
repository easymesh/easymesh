package ip

import (
	"bytes"
	"fmt"
	"net"
)

// similar to net.IPNet but has uint based representation
type IP4Net struct {
	IP        IP4
	PrefixLen uint
}

func NewIP4Net(addr string, len uint) (*IP4Net, error) {
	ipnet, err := ParseIP4(addr)
	if err != nil {
		return nil, err
	}
	return &IP4Net{IP: ipnet, PrefixLen: len}, nil
}

func (n IP4Net) String() string {
	return fmt.Sprintf("%s/%d", n.IP.String(), n.PrefixLen)
}

func (n IP4Net) StringSep(octetSep, prefixSep string) string {
	return fmt.Sprintf("%s%s%d", n.IP.StringSep(octetSep), prefixSep, n.PrefixLen)
}

func (n IP4Net) Network() IP4Net {
	return IP4Net{
		n.IP & IP4(n.Mask()),
		n.PrefixLen,
	}
}

func (n IP4Net) NetworkToIP() net.IP {
	return n.Network().IP.ToIP()
}

func (n IP4Net) Next() IP4Net {
	return IP4Net{
		n.IP + (1 << (32 - n.PrefixLen)),
		n.PrefixLen,
	}
}

func FromIPNet(n *net.IPNet) IP4Net {
	prefixLen, _ := n.Mask.Size()
	return IP4Net{
		FromIP(n.IP),
		uint(prefixLen),
	}
}

func (n IP4Net) ToIPNet() *net.IPNet {
	return &net.IPNet{
		IP:   n.IP.ToIP(),
		Mask: net.CIDRMask(int(n.PrefixLen), 32),
	}
}

func (n IP4Net) Overlaps(other IP4Net) bool {
	var mask uint32
	if n.PrefixLen < other.PrefixLen {
		mask = n.Mask()
	} else {
		mask = other.Mask()
	}
	return (uint32(n.IP) & mask) == (uint32(other.IP) & mask)
}

func (n IP4Net) Equal(other IP4Net) bool {
	return n.IP == other.IP && n.PrefixLen == other.PrefixLen
}

func (n IP4Net) Mask() uint32 {
	var ones uint32 = 0xFFFFFFFF
	return ones << (32 - n.PrefixLen)
}

func (n IP4Net) MaskToIP() net.IP {
	return net.IP(n.ToIPNet().Mask)
}

func (n IP4Net) Contains(ip IP4) bool {
	return (uint32(n.IP) & n.Mask()) == (uint32(ip) & n.Mask())
}

func (n IP4Net) Empty() bool {
	return n.IP == IP4(0) && n.PrefixLen == uint(0)
}

// MarshalJSON: json.Marshaler impl
func (n IP4Net) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, n)), nil
}

// UnmarshalJSON: json.Unmarshaler impl
func (n *IP4Net) UnmarshalJSON(j []byte) error {
	j = bytes.Trim(j, "\"")
	if _, val, err := net.ParseCIDR(string(j)); err != nil {
		return err
	} else {
		*n = FromIPNet(val)
		return nil
	}
}
