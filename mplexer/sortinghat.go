package mplexer

import (
	"git.coolaj86.com/coolaj86/go-telebitd/mplexer/packer"
)

type SortingHat interface {
	LookupTarget(*packer.Addr) (*packer.Conn, error)
	Authz() (string, error)
}
