package packer

import (
	"strconv"
)

func Marshal(addr Addr, body []byte) ([]byte, []byte) {
	header := []byte(`IPv4,192.168.1.101,6743,` + strconv.Itoa(len(body)) + `,http,80,ex1.telebit.io`)
	raw := []byte{255 - 1, byte(len(header))}
	header = append(raw, header...)
	return header, body
}
