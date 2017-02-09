package main

import (
	"html/template"
	"net/http"
)

//launchAdminListener - starts up http listeners and handles various URI paths
func launchAdminListener() {
	loginfo.Println("starting launchAdminListener", *argServerBinding)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch url := r.URL.Path; url {
		case "/":
			handleConnectionWebSocket(connectionTable, w, r, false)
			//w.Header().Set("Content-Type", "text/html; charset=utf-8")
			//template.Must(template.ParseFiles("html/client.html")).Execute(w, r.Host)

		case "/admin":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			template.Must(template.ParseFiles("html/admin.html")).Execute(w, r.Host)

		default:
			http.Error(w, "Not Found", 404)

		}

	})
	s := &http.Server{
		Addr:    *argServerAdminBinding,
		Handler: mux,
	}

	err := s.ListenAndServe()
	if err != nil {
		logfatal.Println("ListenAndServe: ", err)
		panic(err)
	}
}
