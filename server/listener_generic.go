package server

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	telebit "git.coolaj86.com/coolaj86/go-telebitd"
	"git.coolaj86.com/coolaj86/go-telebitd/packer"
	"git.coolaj86.com/coolaj86/go-telebitd/sni"
)

type contextKey string

//CtxConnectionTrack
const (
	ctxServerStatus contextKey = "serverstatus"

	//ctxConnectionTable          contextKey = "connectionTable"

	ctxConfig                   contextKey = "tlsConfig"
	ctxListenerRegistration     contextKey = "listenerRegistration"
	ctxConnectionTrack          contextKey = "connectionTrack"
	ctxWssHostName              contextKey = "wsshostname"
	ctxAdminHostName            contextKey = "adminHostName"
	ctxCancelCheck              contextKey = "cancelcheck"
	ctxLoadbalanceDefaultMethod contextKey = "lbdefaultmethod"
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
	loginfo.Println(":" + string(listenerRegistration.port))
	cancelCheck := ctx.Value(ctxCancelCheck).(int)

	listenAddr, err := net.ResolveTCPAddr("tcp", ":"+strconv.Itoa(listenerRegistration.port))

	if nil != err {
		loginfo.Println(err)
		return
	}

	ln, err := net.ListenTCP("tcp", listenAddr)
	if err != nil {
		loginfo.Println("unable to bind", err)
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
			loginfo.Println("Cancel signal hit")
			return
		default:
			ln.SetDeadline(time.Now().Add(time.Duration(cancelCheck) * time.Second))

			conn, err := ln.Accept()

			if nil != err {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
					continue
				}
				log.Println(err)
				return
			}

			fmt.Println("New connection from %v on %v", conn.LocalAddr(), conn.RemoteAddr())

			// TODO maybe put these into something like mx.newConnCh and have an mx.Accept()?
			wedgeConn := NewWedgeConn(conn)
			go mx.accept(ctx, wedgeConn)
		}
	}
}

//accept -
// - accept a wedgeConnection along with all the other required attritvues
// - peek into the buffer, determine TLS or unencrypted
// - if TSL, then terminate with a TLS endpoint, pass to handleStream
// - if clearText, pass to handleStream
func (mx *MPlexy) accept(ctx context.Context, wConn *WedgeConn) {
	// TODO shouldn't this responsibility fall elsewhere?
	// (otherwise I think we're keeping this function in memory while something else fails to end)
	// (i.e. something, somewhere is missing a `go doStuff()`
	defer wConn.Close()
	peekCnt := 10

	encryptMode := encryptNone

	loginfo.Println("conn", wConn, wConn.LocalAddr().String(), wConn.RemoteAddr().String())
	peek, err := wConn.Peek(peekCnt)

	if err != nil {
		loginfo.Println("error while peeking")
		return
	}

	//take a look for a TLS header.
	if bytes.Contains(peek[0:0], []byte{0x80}) && bytes.Contains(peek[2:4], []byte{0x01, 0x03}) {
		encryptMode = encryptSSLV2

	} else if bytes.Contains(peek[0:3], []byte{0x16, 0x03, 0x00}) {
		encryptMode = encryptSSLV3

	} else if bytes.Contains(peek[0:3], []byte{0x16, 0x03, 0x01}) {
		encryptMode = encryptTLS10
		loginfo.Println("TLS10")

	} else if bytes.Contains(peek[0:3], []byte{0x16, 0x03, 0x02}) {
		encryptMode = encryptTLS11

	} else if bytes.Contains(peek[0:3], []byte{0x16, 0x03, 0x03}) {
		encryptMode = encryptTLS12

	} else if bytes.Contains(peek[0:3], []byte{0x16, 0x03, 0x04}) {
		encryptMode = encryptTLS13

	}

	oneConn := &oneConnListener{wConn}
	tlsConfig := ctx.Value(ctxConfig).(*tls.Config)

	if encryptMode == encryptSSLV2 {
		loginfo.Println("<= SSLv2 is not accepted")
		return

	}

	if encryptMode == encryptNone {
		loginfo.Println("Handle Unencrypted")
		mx.handleStream(ctx, wConn)
		return
	}

	loginfo.Println("Handle Encryption")

	// check SNI heading
	// if matched, then looks like a WSS connection
	// else external don't pull off TLS.

	peek, err = wConn.PeekAll()
	if err != nil {
		loginfo.Println("error while peeking")
		loginfo.Println(hex.Dump(peek[0:]))
		return
	}

	wssHostName := ctx.Value(ctxWssHostName).(string)
	adminHostName := ctx.Value(ctxAdminHostName).(string)

	sniHostName, err := sni.GetHostname(peek)
	if err != nil {
		loginfo.Println(err)
		return
	}

	loginfo.Println("sni:", sniHostName)

	// This is where a target device connects to receive traffic
	if sniHostName == wssHostName {
		//handle WSS Path
		tlsListener := tls.NewListener(oneConn, tlsConfig)

		conn, err := tlsListener.Accept()
		if err != nil {
			loginfo.Println(err)
			return
		}

		tlsWedgeConn := NewWedgeConn(conn)
		mx.handleStream(ctx, tlsWedgeConn)
		return
	}

	// This is where an admin of the relay manages it
	if sniHostName == adminHostName {
		// TODO mx.Admin.CheckRemoteIP(conn) here

		// handle admin path
		tlsListener := tls.NewListener(oneConn, tlsConfig)

		conn, err := tlsListener.Accept()
		if err != nil {
			loginfo.Println(err)
			return
		}

		tlsWedgeConn := NewWedgeConn(conn)
		mx.handleStream(ctx, tlsWedgeConn)
		return
	}

	//traffic not terminating on the rvpn do not decrypt
	loginfo.Println("processing non terminating traffic", wssHostName, sniHostName)
	handleExternalHTTPRequest(ctx, wConn, sniHostName, "https")
}

//handleStream --
// - we have an unencrypted stream connection with the ability to peek
// - attempt to identify HTTP
// - handle http
// 	- attempt to identify as WSS session
// 	- attempt to identify as ADMIN/API session
// 	- else handle as raw http
// - handle other?
func (mx *MPlexy) handleStream(ctx context.Context, wConn *WedgeConn) {
	loginfo.Println("handle Stream")
	loginfo.Println("conn", wConn.LocalAddr().String(), wConn.RemoteAddr().String())

	// TODO couldn't this be dangerous? Or is it limited to a single packet?
	peek, err := wConn.PeekAll()
	if err != nil {
		loginfo.Println("error while peeking", err)
		loginfo.Println(hex.Dump(peek[0:]))
		return
	}

	// TODO handle by TCP port as well
	// (which needs a short read timeout since servers expect clients to say hello)

	// HTTP Identifcation // CRLF
	if !bytes.Contains(peek[:], []byte{0x0d, 0x0a}) {
		return
	}

	//string protocol
	if !bytes.ContainsAny(peek[:], "HTTP/") {
		return
	}

	loginfo.Println("identified HTTP")

	r, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(peek)))
	if err != nil {
		loginfo.Println("identified as HTTP, failed request parsing", err)
		return
	}

	// TODO add newtypes
	// TODO check if this is a websocket
	_, err = mx.authorize(r)

	if err == nil {
		loginfo.Println("Valid WSS dected...sending to handler")
		oneConn := &oneConnListener{wConn}
		mx.handleWssClient(ctx, oneConn)

		//do we have a invalid domain indicating Admin?
		//if yes, prep the oneConn and send it to the handler
		return
	}

	if strings.Contains(r.Host, telebit.InvalidAdminDomain) {
		loginfo.Println("admin")
		oneConn := &oneConnListener{wConn}
		handleAdminClient(ctx, oneConn)
		return

	}
	loginfo.Println("unsupported")
	loginfo.Println(hex.Dump(peek))
	return
}

//handleExternalHTTPRequest -
// - get a wConn and start processing requests
func handleExternalHTTPRequest(ctx context.Context, extConn *WedgeConn, hostname, service string) {
	//connectionTracking := ctx.Value(ctxConnectionTrack).(*Tracking)
	serverStatus := ctx.Value(ctxServerStatus).(*Status)

	defer func() {
		serverStatus.ExtConnectionUnregister(extConn)
		extConn.Close()
	}()

	//find the connection by domain name
	conn, ok := serverStatus.ConnectionTable.ConnByDomain(hostname)
	if !ok {
		//matching connection can not be found based on ConnByDomain
		loginfo.Println("unable to match ", hostname, " to an existing connection")
		//http.Error(, "Domain not supported", http.StatusBadRequest)
		return
	}

	track := NewTrack(extConn, hostname)
	serverStatus.ExtConnectionRegister(track)

	remoteStr := extConn.RemoteAddr().String()
	loginfo.Println("Domain Accepted", hostname, remoteStr)

	var header *packer.Header
	if rAddr, rPort, err := net.SplitHostPort(remoteStr); err != nil {
		loginfo.Println("unable to decode hostport", remoteStr, err)
	} else if port, err := strconv.Atoi(rPort); err != nil {
		loginfo.Printf("unable to parse port string %q: %v\n", rPort, err)
	} else if header, err = packer.NewHeader(rAddr, port, service); err != nil {
		loginfo.Println("unable to create packer header", err)
	}

	if header == nil {
		return
	}

	for {
		buffer, err := extConn.PeekAll()
		if err != nil {
			loginfo.Println("unable to peekAll", err)
			return
		}

		loginfo.Println("Before Packer", hex.Dump(buffer))

		p := packer.NewPacker(header)
		p.Data.AppendBytes(buffer)
		buf := p.PackV1()

		//loginfo.Println(hex.Dump(buf.Bytes()))

		//Bundle up the send request and dispatch
		sendTrack := NewSendTrack(buf.Bytes(), hostname)
		serverStatus.SendExtRequest(conn, sendTrack)

		cnt := len(buffer)
		if _, err = extConn.Discard(cnt); err != nil {
			loginfo.Println("unable to discard", cnt, err)
			return
		}

	}
}

//handleWssClient -
// - expecting an existing oneConnListener with a qualified wss client connected.
// - auth will happen again since we were just peeking at the token.
func (mx *MPlexy) handleWssClient(ctx context.Context, oneConn *oneConnListener) {
	serverStatus := ctx.Value(ctxServerStatus).(*Status)

	//connectionTable := ctx.Value(ctxConnectionTable).(*Table)

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		loginfo.Println("HandleFunc /")
		switch url := r.URL.Path; url {
		case "/":
			loginfo.Println("websocket opening ", r.RemoteAddr, " ", r.Host)

			authz, err := mx.authorize(r)
			var upgrader = websocket.Upgrader{
				ReadBufferSize:  65535,
				WriteBufferSize: 65535,
			}

			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				loginfo.Println("WebSocket upgrade failed", err)
				return
			}

			loginfo.Println("before connection table")

			serverName := authz.Domains[0]

			newRegistration := NewRegistration(conn, r.RemoteAddr, authz.Domains, serverStatus.ConnectionTracking, serverName)
			serverStatus.WSSConnectionRegister(newRegistration)

			if ok := <-newRegistration.CommCh(); !ok {
				loginfo.Println("connection registration failed ", newRegistration)
				return
			}

			loginfo.Println("connection registration accepted ", newRegistration)
		}
	})

	s := &http.Server{
		Addr:    ":80",
		Handler: router,
	}

	err := s.Serve(oneConn)
	if err != nil {
		loginfo.Println("Serve error: ", err)
	}

	select {
	case <-ctx.Done():
		loginfo.Println("Cancel signal hit")
		return
	}
}
