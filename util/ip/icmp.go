package ip

import (
	"encoding/binary"
	"fmt"
)

const MAX_IPOPTLEN   = 40
const MAX_ICMPHEADER = 8

const IPVERSION = 4

const ICMP_DEST_UNREACH = 3
const ICMP_NET_UNREACH  = 0

type ICMPHeader struct {
	Type     uint8
	Code     uint8
	Check    uint16
	Reserved [4]byte
}

type ICMPPacket struct {
	iph   IP4Header
	icmph ICMPHeader
}

func (icmp *ICMPHeader)Coder(body []byte)  {
	body[0] = icmp.Type
	body[1] = icmp.Code
	if NativelyLittle() {
		binary.BigEndian.PutUint16(body[2:], icmp.Check)
	} else {
		binary.LittleEndian.PutUint16(body[2:], icmp.Check)
	}
	copy(body[4:], icmp.Reserved[:])
}

func (icmp *ICMPHeader)MakeCheckSum(data []byte)  {
	body := make([]byte, len(data) + MAX_ICMPHEADER )
	icmp.Coder(body[:MAX_ICMPHEADER])
	copy(body[MAX_ICMPHEADER:], data)
	sumb := make([]uint32, (len(data) + MAX_ICMPHEADER) / 4)
	for i,_ := range sumb {
		sumb[i] = binary.LittleEndian.Uint32(body[i*4:])
	}
	icmp.Check = CheckSum(sumb)
}

func (pkt *ICMPPacket)Coder(data []byte) []byte {
	body := make([]byte, MAX_IPHEADER + MAX_ICMPHEADER + len(data) )
	pkt.iph.Coder(body[:])
	pkt.icmph.Coder(body[MAX_IPHEADER:])
	copy(body[MAX_IPHEADER + MAX_ICMPHEADER:], data)
	return body
}

func ICMPUnreachable(saddr IP4, off_iph *IP4Header, offender []byte) ([]byte, error) {
	var pkt ICMPPacket

	off_iph_len := uint32(off_iph.HeadLen) * 4
	if (off_iph_len >= MAX_IPHEADER + MAX_IPOPTLEN ) {
		return nil,fmt.Errorf("not sending net unreachable: mulformed ip pkt: iph len %d\n", off_iph_len)
	}

	if off_iph.Protocal == IPPROTO_ICMP {
		return nil,fmt.Errorf("To avoid infinite loops, RFC 792 instructs not to send ICMPs about ICMPs")
	}

	if (off_iph.FragOff & 0x1fff) != 0 {
		return nil,fmt.Errorf("ICMP messages are only sent for first fragment")
	}

	pkt.iph.HeadLen = MAX_IPHEADER/4
	pkt.iph.Version = IPVERSION
	pkt.iph.TotLen = uint16(off_iph_len + 8 + MAX_IPHEADER + MAX_ICMPHEADER)
	pkt.iph.TTL = 8
	pkt.iph.Protocal = IPPROTO_ICMP
	pkt.iph.SAddr = saddr
	pkt.iph.DAddr = off_iph.SAddr
	pkt.iph.MakeCheckSum()

	pkt.icmph.Type = ICMP_DEST_UNREACH
	pkt.icmph.Code = ICMP_NET_UNREACH
	pkt.icmph.MakeCheckSum(offender[:off_iph_len + 8])

	return pkt.Coder(offender[:off_iph_len + 8]), nil
}

