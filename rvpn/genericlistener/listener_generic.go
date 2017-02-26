package genericlistener

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/hex"
	"log"
	"net"
	"time"

	"git.daplie.com/Daplie/go-rvpn-server/rvpn/connection"
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
func GenericListenAndServe(ctx context.Context, connectionTable *connection.Table, secretKey string, serverBinding string, certbundle tls.Certificate, deadTime int) {
	config := &tls.Config{Certificates: []tls.Certificate{certbundle}}

	listenAddr, err := net.ResolveTCPAddr("tcp", serverBinding)
	if nil != err {
		loginfo.Println(err)
		return
	}

	ln, err := net.ListenTCP("tcp", listenAddr)
	if err != nil {
		loginfo.Println("unable to bind", err)
		return
	}

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
			go handleConnection(ctx, wedgeConn, connectionTable, secretKey, config)
		}
	}
}

//handleConnection -
// - accept a wedgeConnection along with all the other required attritvues
// - peek into the buffer, determine TLS or unencrypted

func handleConnection(ctx context.Context, wConn *WedgeConn, connectionTable *connection.Table, secretKey string, config *tls.Config) {
	defer wConn.Close()
	peekCnt := 10

	encryptMode := encryptNone

	loginfo.Println("conn", wConn, wConn.LocalAddr().String(), wConn.RemoteAddr().String())
	peek, err := wConn.Peek(peekCnt)

	if err != nil {
		loginfo.Println("error while peeking")
		return
	}
	loginfo.Println(hex.Dump(peek[0:peekCnt]))
	loginfo.Println(hex.Dump(peek[2:4]))
	loginfo.Println("after peek")

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
		loginfo.Println(conn)
		handleStream(conn)
		return
	}

	loginfo.Println("Handle Unencrypted")
	handleStream(wConn)

	return
}

func handleStream(conn net.Conn) {
	var buf [512]byte
	cnt, err := conn.Read(buf[0:])
	if err != nil {
		loginfo.Println(err)
		return
	}

	loginfo.Println(hex.Dump(buf[0:cnt]))

}

//state := NewState()
//wConn := NewWedgeConnSize(conn, 512)
//var buffer [512]byte

// Peek for data to figure out what connection we have
//peekcnt := 32
//peek, err := wConn.Peek(peekcnt)

//if err != nil {
//	loginfo.Println("error while peeking")
//		return
//	}
//loginfo.Println(hex.Dump(peek[0:peekcnt]))
//loginfo.Println("after peek")

// assume http websocket.

//loginfo.Println("wConn", wConn)

//wedgeListener := &WedgeListener{conn: conn}
//LaunchWssListener(connectionTable, &secretKey, wedgeListener)
