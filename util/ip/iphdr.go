package ip

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
)

/* Standard well-defined IP protocols.  */

const (
	IPPROTO_IP   = 0
	IPPROTO_ICMP = 1

	IPPROTO_TCP  = 6
	IPPROTO_UDP  = 17

	IPPROTO_RAW  = 255
)

const MAX_IPHEADER   = 20
const MAX_IP6HEADER  = 40

type IP4Header struct {
	Version uint8
	HeadLen uint8

	Tos     uint8
	TotLen  uint16
	Id      uint16
	FragOff uint16

	TTL      uint8
	Protocal uint8

	Check    uint16
	SAddr    IP4
	DAddr    IP4
}

type IP6Header struct {
	Version  uint8
	Priority uint8

	FlowLabel uint32
	PlayLoad  uint16

	NextHdr  uint8
	HopLimit uint8

	SAddr    IP6
	DAddr    IP6
}

type IPType int

const (
	_ IPType = iota
	Ping
	IPv4
	IPv6
	IPCtrl
)

func IPHeaderType(buff byte) IPType {
	switch (buff >> 4) {
	case 0:return IPCtrl
	case 1:return Ping
	case 4:return IPv4
	case 6:return IPv6
	default:
		return IPCtrl
	}
}

func IP6HeaderDecoder(buff []byte) *IP6Header {
	if len(buff) < 40 {
		return nil
	}
	iphdr := new(IP6Header)
	return iphdr.Decoder(buff)
}

func IP6HeaderCoder(iphdr *IP6Header) []byte {
	buff := make([]byte, 40)
	iphdr.Coder(buff)
	return buff
}

func (iphdr *IP6Header)Decoder(buff []byte) *IP6Header {
	iphdr.Version  = buff[0] >> 4
	iphdr.Priority = buff[0] & 0x0f

	iphdr.SAddr = make([]byte, 16)
	iphdr.DAddr = make([]byte, 16)

	if NativelyLittle() {
		iphdr.FlowLabel = uint32(buff[1]) << 16 | uint32(buff[2]) << 8 | uint32(buff[3])
		iphdr.PlayLoad  = binary.BigEndian.Uint16(buff[4:])
	} else {
		iphdr.FlowLabel = uint32(buff[3]) << 16 | uint32(buff[2]) << 8 | uint32(buff[1])
		iphdr.PlayLoad  = binary.LittleEndian.Uint16(buff[4:])
	}

	iphdr.NextHdr   = buff[6]
	iphdr.HopLimit  = buff[7]

	copy(iphdr.SAddr, buff[8:24])
	copy(iphdr.DAddr, buff[24:40])

	return iphdr
}

func (iphdr *IP6Header)Coder(buff []byte)  {
	buff[0] = iphdr.Version << 4 | iphdr.Priority & 0x0f

	if NativelyLittle() {
		buff[1] = byte(iphdr.FlowLabel >> 16)
		buff[2] = byte(iphdr.FlowLabel >> 8)
		buff[3] = byte(iphdr.FlowLabel )

		binary.BigEndian.PutUint16(buff[4:], iphdr.PlayLoad)
	} else {
		buff[3] = byte(iphdr.FlowLabel >> 16)
		buff[2] = byte(iphdr.FlowLabel >> 8)
		buff[1] = byte(iphdr.FlowLabel )

		binary.LittleEndian.PutUint16(buff[4:], iphdr.PlayLoad)
	}

	buff[6] = iphdr.NextHdr
	buff[7] = iphdr.HopLimit

	copy(buff[8:24], iphdr.SAddr)
	copy(buff[24:40], iphdr.DAddr)
}

func (iphdr *IP6Header)String() string {
	output, _ :=json.Marshal(iphdr)
	return string(output)
}

func IP4HeaderDecoder(buff []byte) *IP4Header {
	if len(buff) < MAX_IPHEADER {
		return nil
	}
	iphdr := new(IP4Header)
	return iphdr.Decoder(buff)
}

func (iphdr *IP4Header)MakeCheckSum()  {
	iphdr.Check = 0
	body := IP4HeaderCoder(iphdr)
	sumb := make([]uint32, 5)
	for i:=0; i<5; i++ {
		sumb[i] = binary.LittleEndian.Uint32(body[i*4:])
	}
	iphdr.Check = CheckSum(sumb)
}

func (iphdr *IP4Header)CheckSum() bool {
	checkold := iphdr.Check
	iphdr.MakeCheckSum()
	checknew := iphdr.Check
	if checkold != checknew {
		iphdr.Check = checkold
		fmt.Printf("old checksum : %x, new checksum: %x", checkold, checknew)
		return false
	}
	return true
}

func (iphdr *IP4Header)Decoder(buff []byte) *IP4Header {
	iphdr.Version = buff[0] >> 4
	iphdr.HeadLen = buff[0] & 0x0f
	iphdr.Tos = buff[1]
	iphdr.TTL = buff[8]
	iphdr.Protocal = buff[9]
	if NativelyLittle() {
		iphdr.TotLen = binary.BigEndian.Uint16(buff[2:])
		iphdr.Id = binary.BigEndian.Uint16(buff[4:])
		iphdr.FragOff = binary.BigEndian.Uint16(buff[6:])

		iphdr.Check = binary.BigEndian.Uint16(buff[10:])
		iphdr.SAddr = IP4(binary.BigEndian.Uint32(buff[12:]))
		iphdr.DAddr = IP4(binary.BigEndian.Uint32(buff[16:]))
	} else {
		iphdr.TotLen = binary.LittleEndian.Uint16(buff[2:])
		iphdr.Id = binary.LittleEndian.Uint16(buff[4:])
		iphdr.FragOff = binary.LittleEndian.Uint16(buff[6:])

		iphdr.Check = binary.LittleEndian.Uint16(buff[10:])
		iphdr.SAddr = IP4(binary.LittleEndian.Uint32(buff[12:]))
		iphdr.DAddr = IP4(binary.LittleEndian.Uint32(buff[16:]))
	}
	return iphdr
}

func (iphdr *IP4Header)Coder(buff []byte)  {
	buff[0] = iphdr.Version << 4 + iphdr.HeadLen & 0x0f
	buff[1] = iphdr.Tos

	buff[8] = iphdr.TTL
	buff[9] = iphdr.Protocal

	if NativelyLittle() {
		binary.BigEndian.PutUint16(buff[2:], iphdr.TotLen)
		binary.BigEndian.PutUint16(buff[4:], iphdr.Id)
		binary.BigEndian.PutUint16(buff[6:], iphdr.FragOff)

		binary.BigEndian.PutUint16(buff[10:], iphdr.Check)
		binary.BigEndian.PutUint32(buff[12:], uint32(iphdr.SAddr))
		binary.BigEndian.PutUint32(buff[16:], uint32(iphdr.DAddr))
	} else {
		binary.LittleEndian.PutUint16(buff[2:], iphdr.TotLen)
		binary.LittleEndian.PutUint16(buff[4:], iphdr.Id)
		binary.LittleEndian.PutUint16(buff[6:], iphdr.FragOff)

		binary.LittleEndian.PutUint16(buff[10:], iphdr.Check)
		binary.LittleEndian.PutUint32(buff[12:], uint32(iphdr.SAddr))
		binary.LittleEndian.PutUint32(buff[16:], uint32(iphdr.DAddr))
	}
}

func (iphdr *IP4Header)String() string {
	output, _ :=json.Marshal(iphdr)
	return string(output)
}

func IP4HeaderCoder(iphdr *IP4Header) []byte {
	buff := make([]byte, 20)
	iphdr.Coder(buff)
	return buff
}

func CheckSum(body []uint32) uint16 {
	var sum uint32
	var t1, t2 uint16

	for _, v := range body {
		sum += v
		if ( sum < v) {
			sum++
		}
	}
	t1 = uint16(sum)
	t2 = uint16(sum >> 16)

	t1 += t2
	if t1 < t2 {
		t1++
	}

	return (^t1 >> 8) | ((^t1 & 0xff) << 8)
}

func (iphdr *IP4Header)DecrementTTL() error {
	iphdr.TTL = iphdr.TTL - 1
	if iphdr.TTL == 0 {
		return fmt.Errorf("Discarding IP fragment %s -> %s due to zero TTL\n",
			iphdr.SAddr, iphdr.DAddr)
	}
	if iphdr.Check >= 0xfeff {
		iphdr.Check += 0x1
	}
	iphdr.Check += 0x100

	/*
	if iphdr.CheckSum() == false {
		return fmt.Errorf("check sum fail")
	}*/
	return nil
}

