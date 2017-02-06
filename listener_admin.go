package main

import (
	"html/template"
	"net/http"
)

//launchAdminListener - starts up http listeners and handles various URI paths
func launchAdminListener() {
	loginfo.Println("starting Admin Listener")

	http.HandleFunc("/admin", handlerServeAdminContent)

	err := http.ListenAndServeTLS(*argServerAdminBinding, "certs/fullchain.pem", "certs/privkey.pem", nil)
	if err != nil {
		logfatal.Println("ListenAndServe: ", err)
		panic(err)
	}
}

func handlerServeAdminContent(w http.ResponseWriter, r *http.Request) {
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
}
