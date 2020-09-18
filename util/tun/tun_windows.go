package tun

import (
	"fmt"
	"github.com/lixiangyun/easymesh/util/ip"
	"golang.org/x/sys/windows"
)

const WIN_TUN_DHCP_LEASE_TIME = 365*24*3600

func (tun *TunWin)Write(p []byte) error {
	hevent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return err
	}

	overlapped := new(windows.Overlapped)
	overlapped.HEvent = hevent
	var l1 uint32
	var l2 uint32

	err = windows.WriteFile(tun.FD, p, &l1, overlapped)
	if err != nil {
		if err == windows.ERROR_IO_PENDING {
			for {
				_, err = windows.WaitForSingleObject(overlapped.HEvent, 100)
				if err != nil {
					return fmt.Errorf("wait for single object fail, %s", err.Error())
				}
				err = windows.GetOverlappedResult(tun.FD, overlapped, &l2, false)
				if err == windows.ERROR_IO_INCOMPLETE {
					continue
				} else {
					break
				}
			}
		} else {
			return fmt.Errorf("write file fail, %s", err.Error())
		}
	}

	if abc(l1, l2) != len(p) {
		return fmt.Errorf("tun send %d out of %d bytes", abc(l1, l2), len(p))
	}

	return nil
}

func abc(a uint32, b uint32) int {
	if a > b {
		return int(a)
	}
	return int(b)
}

func (tun *TunWin)Read(p []byte) (int, error) {
	hevent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return 0, err
	}

	overlapped := new(windows.Overlapped)
	overlapped.HEvent = hevent
	var l1 uint32
	var l2 uint32

	err = windows.ReadFile(tun.FD, p, &l1, overlapped)
	if err != nil {
		if err == windows.ERROR_IO_PENDING {
			for {
				_, err = windows.WaitForSingleObject(overlapped.HEvent, 100)
				if err != nil {
					return 0, fmt.Errorf("wait for single object fail, %s", err.Error())
				}
				err = windows.GetOverlappedResult(tun.FD, overlapped, &l2, false)
				if err == windows.ERROR_IO_INCOMPLETE {
					continue
				} else {
					break
				}
			}
		} else {
			return 0, fmt.Errorf("read file fail, %s", err.Error())
		}
	}

	return abc(l1, l2), err
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