package main

import (
	"context"
	"net"

	"git.coolaj86.com/coolaj86/go-telebitd/mplexer"
)

func main() {
	r := &Router{
		secret: os.Getenv("SECRET"),
	}
	m := &mplexer.MultiplexLocal{
		Relay:      os.Getenv("RELAY"),
		SortingHat: r,
	}

	ctx := context.Background()

	// TODO more m.ListenAndServe(mux) style?
	m.ListenAndServe(ctx)
}

type Router struct {
	secret string
}

func (r *Router) Authz() (string, error) {
	return r.secret, nil
}

// this function is very client-specific logic
func (r *Router) LookupTarget(paddr packer.Addr) (net.Conn, error) {
	//if target := LookupPort(paddr.Servername()); nil != target { }
	if target := r.LookupServername(paddr.Port()); nil != target {
		tconn, err := net.Dial(target.Network(), target.Hostname())
		if nil != err {
			return nil, err
		}
		/*
			// TODO for http proxy
			return mplexer.TargetOptions {
				Hostname // default localhost
				Termination // default TLS
				XFWD // default... no?
				Port // default 0
				Conn // should be dialed beforehand
			}, nil
		*/
		return tconn, nil
	}
}

func (r *Router) LookupServername(servername string) mplexer.Addr {
	return &mplexer.NewAddr(
		mplexer.HTTPS,
		mplexer.TCP, // TCP -> termination.None? / Plain?
		"localhost",
		3000,
	)
}
