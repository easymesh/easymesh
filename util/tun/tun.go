package tun

type TunApi interface {
	Write (p []byte ) error
	Read  (p []byte) (n int, err error)
	Close() error
}

const (
	encapOverhead = 28 // 20 bytes IP hdr + 8 bytes UDP hdr
)