package mplexy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	telebit "git.coolaj86.com/coolaj86/go-telebitd"
	"git.coolaj86.com/coolaj86/go-telebitd/packer"
	"git.coolaj86.com/coolaj86/go-telebitd/relay/api"
	"git.coolaj86.com/coolaj86/go-telebitd/sni"
	"git.coolaj86.com/coolaj86/go-telebitd/tunnel"
)

type contextKey string

//CtxConnectionTrack
const (
	ctxServerStatus             contextKey = "serverstatus"
	ctxConfig                   contextKey = "tlsConfig"
	ctxListenerRegistration     contextKey = "listenerRegistration"
	ctxConnectionTrack          contextKey = "connectionTrack"
	ctxWssHostName              contextKey = "wsshostname"
	ctxAdminHostName            contextKey = "adminHostName"
	ctxCancelCheck              contextKey = "cancelcheck"
	ctxLoadbalanceDefaultMethod contextKey = "lbdefaultmethod"
	//ctxConnectionTable          contextKey = "connectionTable"
)

// TODO isn't this restriction in the TLS lib?
// or are we just pre-checking for remote hosts?
type tlsScheme int

const (
	encryptNone tlsScheme = iota
	encryptSSLV2
	encryptSSLV3
	encryptTLS10
	encryptTLS11
	encryptTLS12
	encryptTLS13
)

// multiListenAndServe -- used to lisen for any https traffic on 443 (8443)
// - setup generic TCP listener, unencrypted TCP, with a Deadtime out
// - leaverage the wedgeConn to peek into the buffer.
// - if TLS, consume connection with TLS certbundle, pass to request identifier
// - else, just pass to the request identififer
func (mx *MPlexy) multiListenAndServe(ctx context.Context, listenerRegistration *ListenerRegistration) {
	Loginfo.Println(":" + string(listenerRegistration.port))
	cancelCheck := ctx.Value(ctxCancelCheck).(int)

	listenAddr, err := net.ResolveTCPAddr("tcp", ":"+strconv.Itoa(listenerRegistration.port))

	if nil != err {
		Loginfo.Println(err)
		return
	}

	ln, err := net.ListenTCP("tcp", listenAddr)
	if err != nil {
		Loginfo.Println("unable to bind", err)
		listenerRegistration.status = listenerFault
		listenerRegistration.err = err
		listenerRegistration.commCh <- listenerRegistration
		return
	}

	listenerRegistration.status = listenerAdded
	listenerRegistration.commCh <- listenerRegistration

	for {
		select {
		case <-ctx.Done():
			Loginfo.Println("Cancel signal hit")
			return
		default:
			ln.SetDeadline(time.Now().Add(time.Duration(cancelCheck) * time.Second))

			conn, err := ln.Accept()

			if nil != err {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
					continue
				}
				Loginfo.Println(err)
				return
			}

			fmt.Println("New connection from %v on %v", conn.LocalAddr(), conn.RemoteAddr())

			// TODO maybe put these into something like mx.newConnCh and have an mx.Accept()?
			wedgeConn := tunnel.NewWedgeConn(conn)
			go mx.accept(ctx, wedgeConn)
		}
	}
}

//accept -
// - accept a wedgeConnection along with all the other required attritvues
// - peek into the buffer, determine TLS or unencrypted
// - if TSL, then terminate with a TLS endpoint, pass to acceptEcryptedStream
// - if clearText, pass to acceptPlainStream
func (mx *MPlexy) accept(ctx context.Context, wConn *tunnel.WedgeConn) {
	peekCnt := 10

	encryptMode := encryptNone

	Loginfo.Println("new conn", wConn, wConn.LocalAddr().String(), wConn.RemoteAddr().String())
	peek, err := wConn.Peek(peekCnt)
	if err != nil {
		Loginfo.Println("error while peeking")
		wConn.Close()
		return
	}

	//take a look for a TLS header.
	if bytes.Contains(peek[0:0], []byte{0x80}) && bytes.Contains(peek[2:4], []byte{0x01, 0x03}) {
		encryptMode = encryptSSLV2

	} else if bytes.Contains(peek[0:3], []byte{0x16, 0x03, 0x00}) {
		encryptMode = encryptSSLV3

	} else if bytes.Contains(peek[0:3], []byte{0x16, 0x03, 0x01}) {
		encryptMode = encryptTLS10
		Loginfo.Println("TLS10")

	} else if bytes.Contains(peek[0:3], []byte{0x16, 0x03, 0x02}) {
		encryptMode = encryptTLS11

	} else if bytes.Contains(peek[0:3], []byte{0x16, 0x03, 0x03}) {
		encryptMode = encryptTLS12

	} else if bytes.Contains(peek[0:3], []byte{0x16, 0x03, 0x04}) {
		encryptMode = encryptTLS13

	}

	if encryptMode == encryptSSLV2 {
		Loginfo.Println("<= SSLv2 is not accepted")
		wConn.Close()
		return

	}

	if encryptMode == encryptNone {
		Loginfo.Println("Handle Unencrypted")
		mx.acceptPlainStream(ctx, wConn, false)
		return
	}

	Loginfo.Println("Handle Encryption")
	mx.acceptEncryptedStream(ctx, wConn)
}

func (mx *MPlexy) acceptEncryptedStream(ctx context.Context, wConn *tunnel.WedgeConn) {
	// Peek at SNI (ServerName) from TLS Hello header

	peek, err := wConn.PeekAll()
	if err != nil {
		Loginfo.Println("Bad socket: read error from", wConn.RemoteAddr(), err)
		Loginfo.Println(hex.Dump(peek[0:]))
		wConn.Close()
		return
	}

	sniHostName, err := sni.GetHostname(peek)
	if err != nil {
		Loginfo.Println("Bad socket: no SNI from", wConn.RemoteAddr(), err)
		Loginfo.Println(err)
		wConn.Close()
		return
	}

	Loginfo.Println("SNI:", sniHostName)

	if sniHostName == mx.wssHostName || sniHostName == mx.adminHostName {
		// The TLS should be terminated and handled internally
		tlsConfig := ctx.Value(ctxConfig).(*tls.Config)
		conn := tls.Client(wConn, tlsConfig)
		tlsWedgeConn := tunnel.NewWedgeConn(conn)
		mx.acceptPlainStream(ctx, tlsWedgeConn, true)
		return
	}

	//oneConn := &oneConnListener{wConn}

	// TLS remains intact and shall be routed downstream, wholesale
	Loginfo.Println("processing non terminating traffic", mx.wssHostName, sniHostName)
	go mx.routeToTarget(ctx, wConn, sniHostName, "https")
}

//acceptPlainStream --
// - we have an unencrypted stream connection with the ability to peek
// - attempt to identify HTTP
// - handle http
// 	- attempt to identify as WSS session
// 	- attempt to identify as ADMIN/API session
// 	- else handle as raw http
// - handle other?
func (mx *MPlexy) acceptPlainStream(ctx context.Context, wConn *tunnel.WedgeConn, encrypted bool) {
	Loginfo.Println("Plain Conn", wConn.LocalAddr().String(), wConn.RemoteAddr().String())

	// TODO couldn't reading everything be dangerous? Or is it limited to a single packet?
	peek, err := wConn.PeekAll()
	if err != nil {
		Loginfo.Println("error while peeking", err)
		Loginfo.Println(hex.Dump(peek[0:]))
		wConn.Close()
		return
	}

	// TODO handle by TCP port as well
	// (which needs a short read timeout since servers expect clients to say hello)

	// HTTP Identifcation // CRLF
	if !bytes.Contains(peek[:], []byte{0x0d, 0x0a}) {
		wConn.Close()
		return
	}

	//string protocol
	if !bytes.ContainsAny(peek[:], "HTTP/") {
		wConn.Close()
		return
	}

	Loginfo.Println("identified HTTP")

	r, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(peek)))
	if err != nil {
		Loginfo.Println("identified as HTTP, failed request parsing", err)
		wConn.Close()
		return
	}

	if strings.Contains(r.Host, telebit.InvalidAdminDomain) {
		Loginfo.Println("admin")
		// TODO mx.Admin.CheckRemoteIP(conn) here
		// handle admin path
		mx.AcceptAdminClient(wConn)
		return

	}

	// TODO add newtypes
	// TODO check if this is a websocket
	_, err = mx.AuthorizeTarget(r)
	if err == nil {
		Loginfo.Println("Valid WSS dected...sending to handler")
		mx.AcceptTargetServer(wConn)
		return
	}

	// TODO sniHostName is the key to the route, which could also be a port or hostname
	//traffic not terminating on the rvpn do not decrypt
	Loginfo.Println("processing non terminating traffic", mx.wssHostName, r.Host)
	Loginfo.Println(hex.Dump(peek))
	if !encrypted {
		// TODO request and cache http resources as a feature??
		go mx.routeToTarget(ctx, wConn, r.Host, "http")
		return
	}

	// This is not presently possible
	Loginfo.Println("impossible condition: local decryption of routable client", mx.wssHostName, r.Host)
	go mx.routeToTarget(ctx, wConn, r.Host, "https")
}

//routeToTarget -
// - get a wConn and start processing requests
func (mx *MPlexy) routeToTarget(ctx context.Context, extConn *tunnel.WedgeConn, hostname, service string) {
	// TODO is this the right place to do this?
	defer extConn.Close()

	//connectionTracking := ctx.Value(ctxConnectionTrack).(*Tracking)
	serverStatus := ctx.Value(ctxServerStatus).(*api.Status)

	defer func() {
		serverStatus.ExtConnectionUnregister(extConn)
		extConn.Close()
	}()

	//find the connection by domain name
	conn, ok := serverStatus.ConnectionTable.ConnByDomain(hostname)
	if !ok {
		//matching connection can not be found based on ConnByDomain
		Loginfo.Println("unable to match ", hostname, " to an existing connection")
		//http.Error(, "Domain not supported", http.StatusBadRequest)
		return
	}

	track := api.NewTrack(extConn, hostname)
	serverStatus.ExtConnectionRegister(track)

	remoteStr := extConn.RemoteAddr().String()
	Loginfo.Println("Domain Accepted", hostname, remoteStr)

	var header *packer.Header
	if rAddr, rPort, err := net.SplitHostPort(remoteStr); err != nil {
		Loginfo.Println("unable to decode hostport", remoteStr, err)
	} else if port, err := strconv.Atoi(rPort); err != nil {
		Loginfo.Printf("unable to parse port string %q: %v\n", rPort, err)
	} else if header, err = packer.NewHeader(rAddr, port, service); err != nil {
		Loginfo.Println("unable to create packer header", err)
	}

	if header == nil {
		return
	}

	for {
		buffer, err := extConn.PeekAll()
		if err != nil {
			Loginfo.Println("unable to peekAll", err)
			return
		}

		Loginfo.Println("Before Packer", hex.Dump(buffer))

		p := packer.NewPacker(header)
		p.Data.AppendBytes(buffer)
		buf := p.PackV1()

		//Loginfo.Println(hex.Dump(buf.Bytes()))

		//Bundle up the send request and dispatch
		sendTrack := api.NewSendTrack(buf.Bytes(), hostname)
		serverStatus.SendExtRequest(conn, sendTrack)

		cnt := len(buffer)
		if _, err = extConn.Discard(cnt); err != nil {
			Loginfo.Println("unable to discard", cnt, err)
			return
		}

	}
}
