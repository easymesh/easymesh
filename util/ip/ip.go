package ip

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"unsafe"
)

// NativeEndian is the ByteOrder of the current system.
var NativeEndian binary.ByteOrder

func init() {
	// Examine the memory layout of an int16 to determine system
	// endianness.
	var one int16 = 1
	b := (*byte)(unsafe.Pointer(&one))
	if *b == 0 {
		NativeEndian = binary.BigEndian
	} else {
		NativeEndian = binary.LittleEndian
	}
}

func NativelyLittle() bool {
	return NativeEndian == binary.LittleEndian
}

type IP6 []byte

type IP4 uint32

func FromBytesIP4(ip []byte) IP4 {
	return IP4(uint32(ip[3]) |
		(uint32(ip[2]) << 8) |
		(uint32(ip[1]) << 16) |
		(uint32(ip[0]) << 24))
}

func FromIP(ip net.IP) IP4 {
	return FromBytesIP4(ip.To4())
}

func (ip IP6) String() string {
	return net.IP(ip).String()
}

// MarshalJSON: json.Marshaler impl
func (ip IP6) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, ip)), nil
}

// UnmarshalJSON: json.Unmarshaler impl
func (ip IP6) UnmarshalJSON(j []byte) error {
	j = bytes.Trim(j, "\"")
	if val, err := ParseIP6(string(j)); err != nil {
		return err
	} else {
		ip = []byte(val)
		return nil
	}
}

func ParseIP6(s string) (net.IP, error) {
	ip := net.ParseIP(s)
	if ip == nil {
		return nil, errors.New("Invalid IP address format")
	}
	return ip.To16(), nil
}

func ParseIP4(s string) (IP4, error) {
	ip := net.ParseIP(s)
	if ip == nil {
		return IP4(0), errors.New("Invalid IP address format")
	}
	return FromIP(ip), nil
}

func MustParseIP4(s string) IP4 {
	ip, err := ParseIP4(s)
	if err != nil {
		panic(err)
	}
	return ip
}

func (ip IP4) Octets() (a, b, c, d byte) {
	a, b, c, d = byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip)
	return
}

func (ip IP4) ToIP() net.IP {
	return net.IPv4(ip.Octets())
}

func (ip IP4) NetworkOrder() uint32 {
	if NativelyLittle() {
		a, b, c, d := byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip)
		return uint32(a) | (uint32(b) << 8) | (uint32(c) << 16) | (uint32(d) << 24)
	} else {
		return uint32(ip)
	}
}

func (ip IP4) String() string {
	return ip.ToIP().String()
}

func (ip IP4) StringSep(sep string) string {
	a, b, c, d := ip.Octets()
	return fmt.Sprintf("%d%s%d%s%d%s%d", a, sep, b, sep, c, sep, d)
}

// MarshalJSON: json.Marshaler impl
func (ip IP4) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, ip)), nil
}

// UnmarshalJSON: json.Unmarshaler impl
func (ip *IP4) UnmarshalJSON(j []byte) error {
	j = bytes.Trim(j, "\"")
	if val, err := ParseIP4(string(j)); err != nil {
		return err
	} else {
		*ip = val
		return nil
	}
}
