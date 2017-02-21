package external

import (
	"net"
	"net/http"
	"strconv"
	"strings"

	"bytes"

	"bufio"

	"git.daplie.com/Daplie/go-rvpn-server/rvpn/connection"
	"git.daplie.com/Daplie/go-rvpn-server/rvpn/packer"
)

//LaunchExternalServer -- used to listen for external connections destin for WSS
func LaunchExternalServer(serverBinding string, connectionTable *connection.Table) {
	addr, err := net.ResolveTCPAddr("tcp4", serverBinding)
	if err != nil {
		loginfo.Println("Unabled to resolve ", serverBinding, " in launchExternalServer")
		loginfo.Println(err)
		return
	}

	loginfo.Println("passed ResolveTCPAddr")

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		loginfo.Println("unable to bind ", serverBinding)
		return
	}

	loginfo.Println("listening")

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			loginfo.Println("Bad accept ", err)
			continue
		}

		go handleConnection(conn, connectionTable)
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

func handleConnection(conn net.Conn, connectionTable *connection.Table) {
	defer conn.Close()

	state := NewState()

	var buffer [512]byte

	for {
		cnt, err := conn.Read(buffer[0:])
		if err != nil {
			return
		}
		loginfo.Println("state ", state, " ", state.Protocol)
		loginfo.Println("conn ", conn)
		loginfo.Println("byte read", cnt)
		//loginfo.Println("buffer")
		//loginfo.Println(hex.Dump(buffer[0:cnt]))

		if state.Protocol == 0 {
			//attempt to discover protocol

			// HTTP Identifcation
			if bytes.Contains(buffer[:], []byte{0x0d, 0x0a}) {
				//string protocol
				if bytes.ContainsAny(buffer[:], "HTTP/") {
					loginfo.Println("identifed HTTP")
					state.Protocol = protoHTTP
				}

			} else if bytes.Contains(buffer[:], []byte{0x16, 0x03, 0x00}) {
				loginfo.Println("identifed SSLV3")
				state.Protocol = protoSSLV3

			} else if bytes.Contains(buffer[:], []byte{0x16, 0x03, 0x01}) {
				loginfo.Println("identifed TLSV1")
				state.Protocol = protoTLSV1

			} else if bytes.Contains(buffer[:], []byte{0x16, 0x03, 0x02}) {
				loginfo.Println("identifed TLSV1.1")
				state.Protocol = protoTLSV11

			} else if bytes.Contains(buffer[:], []byte{0x16, 0x03, 0x03}) {
				loginfo.Println("identifed TLSV2")
				state.Protocol = protoTLSV2

			} else {
				loginfo.Println("Protocol not identified", conn)
				return
			}
		}

		if state.Protocol == 0 {
			loginfo.Println("Making sure protocol is set")
			loginfo.Println(state)
			return
		} else if state.Protocol == protoHTTP {
			readBuffer := bytes.NewBuffer(buffer[0:cnt])
			reader := bufio.NewReader(readBuffer)
			r, err := http.ReadRequest(reader)

			loginfo.Println(r)

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

			loginfo.Println("Remote: ", conn.RemoteAddr().String())

			remoteSplit := strings.Split(conn.RemoteAddr().String(), ":")
			rAddr := remoteSplit[0]
			rPort := remoteSplit[1]

			if conn, ok := connectionTable.ConnByDomain(hostname); !ok {
				//matching connection can not be found based on ConnByDomain
				loginfo.Println("unable to match ", hostname, " to an existing connection")
				//http.Error(, "Domain not supported", http.StatusBadRequest)

			} else {

				loginfo.Println("Domain Accepted")
				loginfo.Println(conn, rAddr, rPort)
				p := packer.NewPacker()
				p.Header.SetAddress(rAddr)
				p.Header.Port, err = strconv.Atoi(rPort)
				p.Header.Port = 8080
				p.Header.Service = "http"
				p.Data.AppendBytes(buffer[0:cnt])
				buf := p.PackV1()

				sendTrack := connection.NewSendTrack(buf.Bytes(), hostname)
				conn.SendCh() <- sendTrack
			}
		}

	}
}
