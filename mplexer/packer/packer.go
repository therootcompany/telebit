package packer

import (
	"fmt"
)

func Encode(src, dst Addr, domain string, payload []byte) ([]byte, []byte, error) {
	n := len(payload)
	header := []byte(fmt.Sprintf(
		"%s,%s,%d,%d,%s,%d,%s,\n",
		src.family, src.addr, src.port,
		n, dst.scheme, dst.port, domain,
	))
	raw := []byte{255 - 1, byte(len(header))}
	header = append(raw, header...)
	return header, payload, nil
}
