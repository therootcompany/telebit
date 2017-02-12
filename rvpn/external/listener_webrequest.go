package external

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
)

//launchWebRequestListener - starts up extern http listeners, gets request and prep's to hand it off inside.
func LaunchWebRequestExternalListener(serverBinding *string) {

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
