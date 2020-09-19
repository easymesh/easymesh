package tun

import (
	"github.com/astaxie/beego/logs"
	"github.com/easymesh/easymesh/util/ip"
	"golang.org/x/sys/windows"
)

const WIN_TUN_DHCP_LEASE_TIME = 365*24*3600

func (tun *TunWin)WriteEventTask()  {
	hevent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		panic(err.Error())
	}

	defer windows.Close(hevent)

	overlapped := new(windows.Overlapped)
	overlapped.HEvent = hevent
	var l1 uint32
	var l2 uint32

	for  {
		body, flag := <- tun.writeBody
		if flag == false {
			return
		}

		err = windows.WriteFile(tun.FD, body, &l1, overlapped)
		if err != nil {
			if err == windows.ERROR_IO_PENDING {
				for {
					_, err = windows.WaitForSingleObject(overlapped.HEvent, 10)
					if err != nil {
						logs.Error("wait for single object fail, %s", err.Error())
						return
					}
					err = windows.GetOverlappedResult(tun.FD, overlapped, &l2, false)
					if err == windows.ERROR_IO_INCOMPLETE {
						continue
					} else {
						break
					}
				}
			} else {
				logs.Error("windows write tun fail, %s", err.Error())
				return
			}
		}

		if abc(l1, l2) != len(body) {
			logs.Error("tun send %d out of %d bytes", abc(l1, l2), len(body))
		}
	}
}

func (tun *TunWin)Write(p []byte) error {
	tun.writeBody <- CloneBody(p)
	return nil
}

func abc(a uint32, b uint32) int {
	if a > b {
		return int(a)
	}
	return int(b)
}

func CloneBody(body []byte) []byte {
	coBody := make([]byte, len(body))
	copy(coBody, body)
	return coBody
}

func (tun *TunWin)ReadEventTask() {
	hevent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		panic(err.Error())
	}

	defer windows.Close(hevent)

	overlapped := new(windows.Overlapped)
	overlapped.HEvent = hevent
	var l1 uint32
	var l2 uint32

	var body [8192]byte

	for  {
		err = windows.ReadFile(tun.FD, body[:], &l1, overlapped)
		if err != nil {
			if err == windows.ERROR_IO_PENDING {
				for {
					_, err = windows.WaitForSingleObject(overlapped.HEvent, 10)
					if err != nil {
						logs.Error("wait for single object fail, %s", err.Error())
						return
					}
					err = windows.GetOverlappedResult(tun.FD, overlapped, &l2, false)
					if err == windows.ERROR_IO_INCOMPLETE {
						continue
					} else {
						break
					}
				}
			} else {
				logs.Error("windows read tun fail, %s", err.Error())
				return
			}
		}

		tun.readBody <- CloneBody(body[:abc(l1, l2)])
	}
}


func (tun *TunWin)Read(p []byte) (int, error) {
	body := <- tun.readBody
	copy(p, body)
	return len(body), nil
}

func (tun *TunWin)Close() error {
	return windows.CloseHandle(tun.FD)
}

func OpenTun(ifname string, ipnet ip.IP4Net) (TunApi, error) {
	wtun, err := openTun(ipnet.IP.ToIP(), ipnet.NetworkToIP(), ipnet.MaskToIP())
	if err != nil {
		return nil, err
	}

	err = wtun.SetDHCPMasq( ipnet.IP.ToIP(), ipnet.MaskToIP(), []byte{0, 0, 0, 0}, WIN_TUN_DHCP_LEASE_TIME)
	if err != nil {
		return nil, err
	}

	err = wtun.Connect()
	if err != nil {
		return nil, err
	}

	return wtun, nil
}