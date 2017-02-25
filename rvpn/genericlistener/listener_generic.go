package genericlistener

import (
	"net"

	"crypto/tls"

	"git.daplie.com/Daplie/go-rvpn-server/rvpn/connection"
)

//LaunchGenericServer -- used to lisen for any https traffic on 443 (8443)
//used to make sure customer devices can reach 443. wss or client
func LaunchGenericServer(connectionTable *connection.Table, secretKey string, serverBinding string, certbundle tls.Certificate) {

	config := &tls.Config{Certificates: []tls.Certificate{certbundle}}

	listener, err := tls.Listen("tcp", serverBinding, config)
	if err != nil {
		loginfo.Println("unable to bind ", serverBinding)
		return
	}

	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			loginfo.Println("Bad accept ", err)
			continue
		}

		go handleConnection(conn, connectionTable, secretKey)
	}
}

type protocol int

//Family -- ENUM for Address Family
const (
	protoHTTP protocol = iota + 1
	protoHTTPS
	protoSSLV3
	protoTLSV1
	protoTLSV11
	protoTLSV2
)

//State -- state of connection
type State struct {
	Protocol protocol
}

//NewState -- Constructor
func NewState() (p *State) {
	p = new(State)
	return
}

func handleConnection(conn net.Conn, connectionTable *connection.Table, secretKey string) {
	defer conn.Close()

	loginfo.Println("conn", conn)
	loginfo.Println("hank")
	loginfo.Println("here", conn.LocalAddr().String(), conn.RemoteAddr().String())

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

	return
}
