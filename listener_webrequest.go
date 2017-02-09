package main

import "net/http"
import "net/http/httputil"

//launchWebRequestListener - starts up extern http listeners, gets request and prep's to hand it off inside.
func launchWebRequestExternalListener() {

	loginfo.Println("starting WebRequestExternal Listener ", *argServerExternalBinding)

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
		Addr:    *argServerExternalBinding,
		Handler: mux,
	}

	err := s.ListenAndServe()
	if err != nil {
		logfatal.Println("ListenAndServe: ", err)
		panic(err)
	}
}
