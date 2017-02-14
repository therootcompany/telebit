package external

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"strconv"

	"strings"

	"git.daplie.com/Daplie/go-rvpn-server/rvpn/connection"
	"git.daplie.com/Daplie/go-rvpn-server/rvpn/packer"
)

//LaunchWebRequestExternalListener - starts up extern http listeners, gets request and prep's to hand it off inside.
func LaunchWebRequestExternalListener(serverBinding *string, connectionTable *connection.Table) {
	loginfo.Println("starting WebRequestExternal Listener ", *serverBinding)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch url := r.URL.Path; url {
		default:
			loginfo.Println("handlerWebRequestExternal")

			dump, err := httputil.DumpRequest(r, true)
			if err != nil {
				loginfo.Println(err)
			} else {
				loginfo.Printf("%q", dump)
			}

			hostname := r.Host

			if strings.Contains(hostname, ":") {
				arr := strings.Split(hostname, ":")
				hostname = arr[0]
			}

			remoteSplit := strings.Split(r.RemoteAddr, ":")
			rAddr := remoteSplit[0]
			rPort := remoteSplit[1]

			if conn, ok := connectionTable.ConnByDomain(hostname); !ok {
				//matching connection can not be found based on ConnByDomain
				loginfo.Println("unable to match ", hostname, " to an existing connection")
				http.Error(w, "Domain not supported", http.StatusBadRequest)

			} else {
				loginfo.Println(conn, rAddr, rPort)
				p := packer.NewPacker()
				p.Header.SetAddress("127.0.0.2")
				p.Header.Port, err = strconv.Atoi(rPort)
				p.Data.AppendBytes(dump)
				buf := p.PackV1()

				conn.SendCh() <- buf.Bytes()
			}
		}
	})
	s := &http.Server{
		Addr:      *serverBinding,
		Handler:   mux,
		ConnState: connState,
	}

	err := s.ListenAndServe()
	if err != nil {
		loginfo.Println("ListenAndServe: ", err)
		panic(err)
	}
}

func connState(conn net.Conn, state http.ConnState) {
	loginfo.Println("connState")
	fmt.Println(conn, conn.LocalAddr(), conn.RemoteAddr())
	fmt.Println(state)
}
