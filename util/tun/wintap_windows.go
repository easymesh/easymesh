package tun

import (
	"encoding/binary"
	"fmt"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"net"
)

const (
	TAPWIN32_MAX_REG_SIZE = 256
	TUNTAP_COMPONENT_ID = "tap0901"
	ADAPTER_KEY = `SYSTEM\CurrentControlSet\Control\Class\{4D36E972-E325-11CE-BFC1-08002BE10318}`
	NETWORK_CONNECTIONS_KEY = `SYSTEM\CurrentControlSet\Control\Network\{4D36E972-E325-11CE-BFC1-08002BE10318}`
	USERMODEDEVICEDIR = `\\.\Global\`
	SYSDEVICEDIR = `\Device\`
	USERDEVICEDIR = `\DosDevices\Global`
	TAP_WIN_SUFFIX = ".tap"
)

const (
	TAP_WIN_IOCTL_GET_MAC = 1
	TAP_WIN_IOCTL_GET_VERSION = 2
	TAP_WIN_IOCTL_GET_MTU = 3
	TAP_WIN_IOCTL_GET_INFO = 4
	TAP_WIN_IOCTL_CONFIG_POINT_TO_POINT = 5
	TAP_WIN_IOCTL_SET_MEDIA_STATUS = 6
	TAP_WIN_IOCTL_CONFIG_DHCP_MASQ = 7
	TAP_WIN_IOCTL_GET_LOG_LINE = 8
	TAP_WIN_IOCTL_CONFIG_DHCP_SET_OPT = 9
	TAP_WIN_IOCTL_CONFIG_TUN = 10
)

const (
	FILE_ANY_ACCESS = 0
	METHOD_BUFFERED = 0
)

type TunWin struct {
	ID               string
	MTU              uint32
	DevicePath       string
	FD               windows.Handle
	NetworkName      string

	readBody         chan []byte
	writeBody        chan []byte
}

func ctl_code(device_type, function, method, access uint32) uint32 {
	return (device_type << 16) | (access << 14) | (function << 2) | method
}

func tap_control_code(request, method uint32) uint32 {
	return ctl_code(34, request, method, FILE_ANY_ACCESS)
}

func tap_ioctl(cmd uint32) uint32 {
	return tap_control_code(cmd, METHOD_BUFFERED)
}

func matchKey(zones registry.Key, kName string) (string, error) {
	k, err := registry.OpenKey(zones, kName, registry.READ)
	if err != nil {
		return "", err
	}
	defer k.Close()

	cId, _, err := k.GetStringValue("ComponentId")
	if cId == "tap0901" {
		netCfgInstanceId, _, err := k.GetStringValue("NetCfgInstanceId")
		if err != nil {
			return "", err
		}
		return netCfgInstanceId, nil
	}
	cId, _, err = k.GetStringValue("ProductName")
	if cId == "TAP-Windows Adapter V9" {
		netCfgInstanceId, _, err := k.GetStringValue("NetCfgInstanceId")
		if err != nil {
			return "", err
		}
		return netCfgInstanceId, nil
	}
	return "", fmt.Errorf("not tap windows interface")
}

func getTuntapInstanceID() ([]string, error) {
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE, ADAPTER_KEY,
		registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
	if err != nil {
		return nil, err
	}
	defer k.Close()

	names, err := k.ReadSubKeyNames(-1)
	if err != nil {
		return nil, err
	}

	instanceIDs := make([]string, 0)
	for _, name := range names {
		nID, _ := matchKey(k, name)
		if nID != "" {
			instanceIDs = append(instanceIDs, nID)
		}
	}

	if len(instanceIDs) == 0 {
		return nil, fmt.Errorf("Not Found")
	}
	return instanceIDs, nil
}

// OpenTun function open the tap0901 device and set config
// Params: addr -> the localIPAddr
//         network -> remoteNetwork
//         mask -> remoteNetmask
// The function configure a network for later actions
// The tun will process those transmit between local ip
// and remote network
func openTun(addr, network, mask net.IP) (*TunWin, error) {
	nIDs, err := getTuntapInstanceID()
	if err != nil {
		return nil, err
	}

	tun := new(TunWin)
	for _, id := range nIDs {
		tun.ID = id
		tun.DevicePath = fmt.Sprintf(USERMODEDEVICEDIR+"%s"+TAP_WIN_SUFFIX, id)

		name, _ := windows.UTF16PtrFromString(fmt.Sprintf(`\\.\Global\%s.tap`, id))
		access := uint32(windows.GENERIC_READ | windows.GENERIC_WRITE)
		mode := uint32(windows.FILE_SHARE_READ | windows.FILE_SHARE_WRITE)

		tun.FD, err = windows.CreateFile(name, access, mode, nil,
			windows.OPEN_EXISTING,
			windows.FILE_ATTRIBUTE_SYSTEM | windows.FILE_FLAG_OVERLAPPED, 0)
		if err != nil {
			continue
		}
		break
	}

	if err != nil {
		return nil, err
	}

	var returnLen uint32
	configTunParam := append(addr.To4(), network.To4()...)
	configTunParam = append(configTunParam, mask.To4()...)
	err = windows.DeviceIoControl(tun.FD, tap_ioctl(TAP_WIN_IOCTL_CONFIG_TUN),
		&configTunParam[0], uint32(len(configTunParam)),
		&configTunParam[0], uint32(len(configTunParam)), // I think here can be nil
		&returnLen, nil)
	if err != nil {
		return nil, err
	}

	tun.readBody = make(chan []byte, 1024)
	tun.writeBody = make(chan []byte, 1024)

	return tun, nil
}

func (tun *TunWin) GetMTU(refresh bool) (uint32) {
	if !refresh && tun.MTU != 0 {
		return tun.MTU
	}

	var returnLen uint32
	var umtu = make([]byte, 4)
	err := windows.DeviceIoControl(tun.FD, tap_ioctl(TAP_WIN_IOCTL_GET_MTU),
		&umtu[0], uint32(len(umtu)),
		&umtu[0], uint32(len(umtu)),
		&returnLen, nil)
	if err != nil {
		return 0
	}
	tun.MTU = binary.LittleEndian.Uint32(umtu)

	return tun.MTU
}

func (tun *TunWin) Connect() error {
	go tun.ReadEventTask()
	go tun.WriteEventTask()

	var returnLen uint32
	inBuffer := []byte{1, 0, 0, 0}
	err := windows.DeviceIoControl(
		tun.FD, tap_ioctl(TAP_WIN_IOCTL_SET_MEDIA_STATUS),
		&inBuffer[0], uint32(len(inBuffer)),
		&inBuffer[0], uint32(len(inBuffer)),
		&returnLen, nil)
	return err
}

func (tun *TunWin) SetDHCPMasq(dhcpAddr, dhcpMask, serverIP net.IP, leaseTime uint32) error {
	var returnLen uint32

	var leaseTimes [4]byte
	binary.LittleEndian.PutUint32(leaseTimes[:], leaseTime)

	configTunParam := append(dhcpAddr.To4(), dhcpMask.To4()...)
	configTunParam = append(configTunParam, serverIP.To4()...)
	configTunParam = append(configTunParam, net.IP(leaseTimes[:])...)

	err := windows.DeviceIoControl(tun.FD, tap_ioctl(TAP_WIN_IOCTL_CONFIG_DHCP_MASQ),
		&configTunParam[0], uint32(len(configTunParam)),
		&configTunParam[0], uint32(len(configTunParam)), // I think here can be nil
		&returnLen, nil)

	return err
}

func (tun *TunWin) GetNetworkName(refresh bool) string {
	if !refresh && tun.NetworkName != "" {
		return tun.NetworkName
	}
	keyName := `SYSTEM\CurrentControlSet\Control\Network\{4D36E972-E325-11CE-BFC1-08002BE10318}\` +
		tun.ID + `\Connection`
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, keyName, registry.ALL_ACCESS)
	if err != nil {
		return ""
	}
	szname, _, err := k.GetStringValue("Name")
	if err != nil {
		return ""
	}
	k.Close()
	tun.NetworkName = szname
	return szname
}
