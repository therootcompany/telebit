package server

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/hex"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	telebit "git.coolaj86.com/coolaj86/go-telebitd"
	"git.coolaj86.com/coolaj86/go-telebitd/packer"
	"git.coolaj86.com/coolaj86/go-telebitd/sni"
)

type contextKey string

//CtxConnectionTrack
const (
	ctxSecretKey    contextKey = "secretKey"
	ctxServerStatus contextKey = "serverstatus"

	//ctxConnectionTable          contextKey = "connectionTable"

	ctxConfig                   contextKey = "config"
	ctxListenerRegistration     contextKey = "listenerRegistration"
	ctxConnectionTrack          contextKey = "connectionTrack"
	ctxWssHostName              contextKey = "wsshostname"
	ctxAdminHostName            contextKey = "adminHostName"
	ctxCancelCheck              contextKey = "cancelcheck"
	ctxLoadbalanceDefaultMethod contextKey = "lbdefaultmethod"
)

const (
	encryptNone int = iota
	encryptSSLV2
	encryptSSLV3
	encryptTLS10
	encryptTLS11
	encryptTLS12
)

//GenericListenAndServe -- used to lisen for any https traffic on 443 (8443)
// - setup generic TCP listener, unencrypted TCP, with a Deadtime out
// - leaverage the wedgeConn to peek into the buffer.
// - if TLS, consume connection with TLS certbundle, pass to request identifier
// - else, just pass to the request identififer
func GenericListenAndServe(ctx context.Context, listenerRegistration *ListenerRegistration) {
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

			wedgeConn := NewWedgeConn(conn)
			go acceptTCPOrTLS(ctx, wedgeConn)
		}
	}
}

//acceptTCPOrTLS -
// - accept a wedgeConnection along with all the other required attritvues
// - peek into the buffer, determine TLS or unencrypted
// - if TSL, then terminate with a TLS endpoint, pass to handleStream
// - if clearText, pass to handleStream
func acceptTCPOrTLS(ctx context.Context, wConn *WedgeConn) {
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
	}

	oneConn := &oneConnListener{wConn}
	config := ctx.Value(ctxConfig).(*tls.Config)

	if encryptMode == encryptSSLV2 {
		loginfo.Println("SSLv2 is not accepted")
		return

	}

	if encryptMode == encryptNone {
		loginfo.Println("Handle Unencrypted")
		handleStream(ctx, wConn)
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

	if sniHostName == wssHostName {
		//handle WSS Path
		tlsListener := tls.NewListener(oneConn, config)

		conn, err := tlsListener.Accept()
		if err != nil {
			loginfo.Println(err)
			return
		}

		tlsWedgeConn := NewWedgeConn(conn)
		handleStream(ctx, tlsWedgeConn)
		return

	} else if sniHostName == adminHostName {
		// handle admin path
		tlsListener := tls.NewListener(oneConn, config)

		conn, err := tlsListener.Accept()
		if err != nil {
			loginfo.Println(err)
			return
		}

		tlsWedgeConn := NewWedgeConn(conn)
		handleStream(ctx, tlsWedgeConn)
		return

	} else {
		//traffic not terminating on the rvpn do not decrypt
		loginfo.Println("processing non terminating traffic", wssHostName, sniHostName)
		handleExternalHTTPRequest(ctx, wConn, sniHostName, "https")
	}

	return
}

//handleStream --
// - we have an unencrypted stream connection with the ability to peek
// - attempt to identify HTTP
// - handle http
// 	- attempt to identify as WSS session
// 	- attempt to identify as ADMIN/API session
// 	- else handle as raw http
// - handle other?
func handleStream(ctx context.Context, wConn *WedgeConn) {
	loginfo.Println("handle Stream")
	loginfo.Println("conn", wConn.LocalAddr().String(), wConn.RemoteAddr().String())

	// TODO couldn't this be dangerous? Or is it limited to a single packet?
	peek, err := wConn.PeekAll()
	if err != nil {
		loginfo.Println("error while peeking", err)
		loginfo.Println(hex.Dump(peek[0:]))
		return
	}

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

	// do we have a valid wss_client?
	secretKey := ctx.Value(ctxSecretKey).(string)
	tokenString := r.URL.Query().Get("access_token")
	result, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})

	if err == nil && result.Valid {
		loginfo.Println("Valid WSS dected...sending to handler")
		oneConn := &oneConnListener{wConn}
		handleWssClient(ctx, oneConn)

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
func handleWssClient(ctx context.Context, oneConn *oneConnListener) {
	secretKey := ctx.Value(ctxSecretKey).(string)
	serverStatus := ctx.Value(ctxServerStatus).(*Status)

	//connectionTable := ctx.Value(ctxConnectionTable).(*Table)

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		loginfo.Println("HandleFunc /")
		switch url := r.URL.Path; url {
		case "/":
			loginfo.Println("websocket opening ", r.RemoteAddr, " ", r.Host)

			tokenString := r.URL.Query().Get("access_token")
			result, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				return []byte(secretKey), nil
			})

			if err != nil || !result.Valid {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("Not Authorized"))
				loginfo.Println("access_token invalid...closing connection")
				return
			}

			claims := result.Claims.(jwt.MapClaims)
			domains, ok := claims["domains"].([]interface{})

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

			serverName := domains[0].(string)

			newRegistration := NewRegistration(conn, r.RemoteAddr, domains, serverStatus.ConnectionTracking, serverName)
			serverStatus.WSSConnectionRegister(newRegistration)

			ok = <-newRegistration.CommCh()
			if !ok {
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
