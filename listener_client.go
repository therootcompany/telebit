package main

import (
	"html/template"
	"net/http"
)

//launchListener - starts up http listeners and handles various URI paths
func launchClientListener() {
	loginfo.Println("starting Client Listener ", argServerBinding)

	connectionTable = newConnectionTable()
	go connectionTable.run()
	http.HandleFunc("/", handlerServeContent)

	err := http.ListenAndServeTLS(*argServerBinding, "certs/fullchain.pem", "certs/privkey.pem", nil)
	if err != nil {
		logfatal.Println("ListenAndServe: ", err)
		panic(err)
	}
}

func handlerServeContent(w http.ResponseWriter, r *http.Request) {
	switch url := r.URL.Path; url {
	case "/":
		handleConnectionWebSocket(connectionTable, w, r, false)
		//w.Header().Set("Content-Type", "text/html; charset=utf-8")
		//template.Must(template.ParseFiles("html/client.html")).Execute(w, r.Host)

	case "/admin":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		template.Must(template.ParseFiles("html/admin.html")).Execute(w, r.Host)

	case "/ws/client":
		handleConnectionWebSocket(connectionTable, w, r, false)

	case "/ws/admin":
		handleConnectionWebSocket(connectionTable, w, r, true)

	default:
		http.Error(w, "Not Found", 404)

	}
}
