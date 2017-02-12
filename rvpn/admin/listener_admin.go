package admin

import (
	"html/template"
	"net/http"
)

//LaunchAdminListener - starts up http listeners and handles various URI paths
func LaunchAdminListener(serverBinding *string) (err error) {
	loginfo.Println("starting launchAdminListener", *serverBinding)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch url := r.URL.Path; url {
		case "/":
			//handleConnectionWebSocket(connectionTable, w, r, false)
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
		Addr:    *serverBinding,
		Handler: mux,
	}

	err = s.ListenAndServe()
	if err != nil {
		loginfo.Println("ListenAndServe: ", err)
	}
	return
}
