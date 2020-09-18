package udp

import (
	"fmt"
	"math/rand"
	"net"
)

func OpenUdp(bindAddr string) (*net.UDPConn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", bindAddr)
	if err != nil {
		return nil, err
	}
	udpHander, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}
	return udpHander, nil
}

func UdpWrite(conn *net.UDPConn, dstAddr *net.UDPAddr, body []byte ) error {
	cnt, err := conn.WriteToUDP(body, dstAddr)
	if err != nil {
		return fmt.Errorf("udp write fail, %s", err.Error())
	}
	if cnt != len(body) {
		return fmt.Errorf("udp send %d out of %d bytes", cnt, len(body))
	}
	return nil
}

func UdpCtrl(body []byte) []byte {
	output := make([]byte, len(body) + 1)
	output[0] = 0
	copy(output[1:], body)
	return output
}

func UdpPing(body []byte) []byte {
	output := make([]byte, len(body) + 1)
	output[0] = 1 << 4
	copy(output[1:], body)
	return output
}

func UnusedPort() int {
	begin := 10000
	end := 50000
	for  {
		port := begin + (rand.Int() % (end - begin))
		udpconn, err := OpenUdp(fmt.Sprintf("0.0.0.0:%d", port))
		if err != nil {
			continue
		}
		defer udpconn.Close()
		return port
	}
}
