package packer

import (
	"fmt"
)

// Encode creates an MPLEXY V1 header for the given addresses and payload
func Encode(payload []byte, id, tun Addr) ([]byte, []byte, error) {
	n := len(payload)
	domain := tun.addr
	header := []byte(fmt.Sprintf(
		"%s,%s,%d,%d,%s,%d,%s,\n",
		id.family, id.addr, id.port,
		n, tun.scheme, tun.port, domain,
	))
	raw := []byte{255 - 1, byte(len(header))}
	header = append(raw, header...)
	return header, payload, nil
}
