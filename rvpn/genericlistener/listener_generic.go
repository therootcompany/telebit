package genericlistener

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"net/http"

	"bufio"

	"git.daplie.com/Daplie/go-rvpn-server/rvpn/packer"
)

type contextKey string

//CtxConnectionTrack
const (
	ctxSecretKey            contextKey = "secretKey"
	ctxConnectionTable      contextKey = "connectionTable"
	ctxConfig               contextKey = "config"
	ctxDeadTime             contextKey = "deadtime"
	ctxListenerRegistration contextKey = "listenerRegistration"
	ctxConnectionTrack      contextKey = "connectionTrack"
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

	listenAddr, err := net.ResolveTCPAddr("tcp", ":"+strconv.Itoa(listenerRegistration.port))
	deadTime := ctx.Value(ctxDeadTime).(int)

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
			ln.SetDeadline(time.Now().Add(time.Duration(deadTime) * time.Second))

			conn, err := ln.Accept()

			loginfo.Println("Deadtime reached")

			if nil != err {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
					continue
				}
				log.Println(err)
				return
			}

			wedgeConn := NewWedgeConn(conn)
			go handleConnection(ctx, wedgeConn)
		}
	}
}

//handleConnection -
// - accept a wedgeConnection along with all the other required attritvues
// - peek into the buffer, determine TLS or unencrypted
// - if TSL, then terminate with a TLS endpoint, pass to handleStream
// - if clearText, pass to handleStream
func handleConnection(ctx context.Context, wConn *WedgeConn) {
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

	} else if encryptMode != encryptNone {
		loginfo.Println("Handle Encryption")
		tlsListener := tls.NewListener(oneConn, config)

		conn, err := tlsListener.Accept()
		if err != nil {
			loginfo.Println(err)
			return
		}

		tlsWedgeConn := NewWedgeConn(conn)
		handleStream(ctx, tlsWedgeConn)
		return
	}

	loginfo.Println("Handle Unencrypted")
	handleStream(ctx, wConn)

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
	loginfo.Println("conn", wConn, wConn.LocalAddr().String(), wConn.RemoteAddr().String())

	peek, err := wConn.PeekAll()
	if err != nil {
		loginfo.Println("error while peeking")
		loginfo.Println(hex.Dump(peek[0:]))
		return
	}

	// HTTP Identifcation
	if bytes.Contains(peek[:], []byte{0x0d, 0x0a}) {
		//string protocol
		if bytes.ContainsAny(peek[:], "HTTP/") {
			loginfo.Println("identifed HTTP")

			r, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(peek)))
			if err != nil {
				loginfo.Println("identifed as HTTP, failed request parsing", err)
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
			} else if strings.Contains(r.Host, "rvpn.daplie.invalid") {
				loginfo.Println("admin")
				oneConn := &oneConnListener{wConn}
				handleAdminClient(ctx, oneConn)
				return

			} else {
				loginfo.Println("default connection")
				handleExternalHTTPRequest(ctx, wConn)
				return
			}
		}
	}
}

//handleExternalHTTPRequest -
// - get a wConn and start processing requests
func handleExternalHTTPRequest(ctx context.Context, extConn net.Conn) {
	connectionTracking := ctx.Value(ctxConnectionTrack).(*Tracking)
	connectionTracking.register <- extConn

	defer func() {
		connectionTracking.unregister <- extConn
		extConn.Close()
	}()

	connectionTable := ctx.Value(ctxConnectionTable).(*Table)

	var buffer [512]byte
	for {
		cnt, err := extConn.Read(buffer[0:])
		if err != nil {
			return
		}

		readBuffer := bytes.NewBuffer(buffer[0:cnt])
		reader := bufio.NewReader(readBuffer)
		r, err := http.ReadRequest(reader)

		if err != nil {
			loginfo.Println("error parsing request")
			return
		}

		hostname := r.Host
		loginfo.Println("Host: ", hostname)

		if strings.Contains(hostname, ":") {
			arr := strings.Split(hostname, ":")
			hostname = arr[0]
		}

		loginfo.Println("Remote: ", extConn.RemoteAddr().String())

		remoteSplit := strings.Split(extConn.RemoteAddr().String(), ":")
		rAddr := remoteSplit[0]
		rPort := remoteSplit[1]

		//find the connection by domain name
		conn, ok := connectionTable.ConnByDomain(hostname)
		if !ok {
			//matching connection can not be found based on ConnByDomain
			loginfo.Println("unable to match ", hostname, " to an existing connection")
			//http.Error(, "Domain not supported", http.StatusBadRequest)
			return
		}

		loginfo.Println("Domain Accepted", conn, rAddr, rPort)
		p := packer.NewPacker()
		p.Header.SetAddress(rAddr)
		p.Header.Port, err = strconv.Atoi(rPort)
		if err != nil {
			loginfo.Println("Unable to set Remote port", err)
			return
		}

		p.Header.Service = "http"
		p.Data.AppendBytes(buffer[0:cnt])
		buf := p.PackV1()

		sendTrack := NewSendTrack(buf.Bytes(), hostname)
		conn.SendCh() <- sendTrack
	}
}

//handleAdminClient -
// - expecting an existing oneConnListener with a qualified wss client connected.
// - auth will happen again since we were just peeking at the token.
func handleAdminClient(ctx context.Context, oneConn *oneConnListener) {
	connectionTable := ctx.Value(ctxConnectionTable).(*Table)

	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		loginfo.Println("HandleFunc /")
		switch url := r.URL.Path; url {
		case "/":
			// check to see if we are using the administrative Host
			if strings.Contains(r.Host, "rvpn.daplie.invalid") {
				http.Redirect(w, r, "/admin", 301)
			}

		default:
			http.Error(w, "Not Found", 404)
		}
	})

	router.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintln(w, "<html>Welcome..press <a href=/api/servers>Servers</a> to access stats</html>")
	})

	router.HandleFunc("/api/servers", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("here")
		serverContainer := NewServerAPIContainer()

		for c := range connectionTable.Connections() {
			serverAPI := NewServerAPI(c)
			serverContainer.Servers = append(serverContainer.Servers, serverAPI)

		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		json.NewEncoder(w).Encode(serverContainer)

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

//handleWssClient -
// - expecting an existing oneConnListener with a qualified wss client connected.
// - auth will happen again since we were just peeking at the token.
func handleWssClient(ctx context.Context, oneConn *oneConnListener) {
	secretKey := ctx.Value(ctxSecretKey).(string)
	connectionTable := ctx.Value(ctxConnectionTable).(*Table)

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
				ReadBufferSize:  1024,
				WriteBufferSize: 1024,
			}

			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				loginfo.Println("WebSocket upgrade failed", err)
				return
			}

			loginfo.Println("before connection table")

			//newConnection := connection.NewConnection(connectionTable, conn, r.RemoteAddr, domains)

			connectionTrack := ctx.Value(ctxConnectionTrack).(*Tracking)
			newRegistration := NewRegistration(conn, r.RemoteAddr, domains, connectionTrack)
			connectionTable.Register() <- newRegistration
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
