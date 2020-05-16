package mplexer

import (
	"net"
)

type SortingHat interface {
	LookupTarget(*Addr) (net.Conn, error)
	Authz() (string, error)
}
