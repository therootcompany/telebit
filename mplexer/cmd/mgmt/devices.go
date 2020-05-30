package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"git.coolaj86.com/coolaj86/go-telebitd/mplexer/mgmt/authstore"
	"github.com/go-chi/chi"
)

func handleDeviceRoutes(r chi.Router) {
	r.Route("/devices", func(r chi.Router) {
		// TODO needs admin auth
		// r.Use() // must have slug '*'

		r.Post("/", func(w http.ResponseWriter, r *http.Request) {

			auth := &authstore.Authorization{}
			decoder := json.NewDecoder(r.Body)
			err := decoder.Decode(&auth)
			// Slug is mandatory, ID and MachinePPID must NOT be set
			epoch := time.Time{}
			if nil != err || "" != auth.ID || "" != auth.MachinePPID ||
				"" == auth.Slug || "" == auth.SharedKey ||
				epoch != auth.CreatedAt || epoch != auth.UpdatedAt || epoch != auth.DeletedAt {
				result, _ := json.Marshal(&authstore.Authorization{})
				msg, _ := json.Marshal(&struct {
					Error string `json:"error"`
				}{
					Error: "expected JSON in the format " + string(result),
				})
				http.Error(w, string(msg), http.StatusUnprocessableEntity)
				return
			}

			err = store.Add(auth)
			if nil != err {
				msg := `{"error":"not really sure what happened, but it didn't go well (check the logs)"}`
				log.Printf("/api/devices/\n", auth.Slug)
				log.Println(err)
				http.Error(w, msg, http.StatusInternalServerError)
				return
			}

			//auth.SharedKey = "[redacted]"
			result, _ := json.Marshal(auth)
			w.Write(result)
		})

		r.Get("/{slug}", func(w http.ResponseWriter, r *http.Request) {
			slug := chi.URLParam(r, "slug")
			// TODO store should be concurrency-safe
			auth, err := store.Get(slug)
			if nil != err {
				msg := `{"error":"not really sure what happened, but it didn't go well (check the logs)"}`
				log.Printf("/api/devices/%s\n", slug)
				log.Println(err)
				http.Error(w, msg, http.StatusInternalServerError)
				return
			}

			// Redact private data
			if "" != auth.MachinePPID {
				auth.MachinePPID = "[redacted]"
			}
			if "" != auth.SharedKey {
				auth.SharedKey = "[redacted]"
			}
			result, _ := json.Marshal(auth)
			w.Write(result)
		})

		r.Delete("/{slug}", func(w http.ResponseWriter, r *http.Request) {
			slug := chi.URLParam(r, "slug")
			auth, err := store.Get(slug)
			if nil == auth {
				msg := `{"error":"not really sure what happened, but it didn't go well (check the logs)"}`
				log.Printf("/api/devices/%s\n", slug)
				log.Println(err)
				http.Error(w, msg, http.StatusInternalServerError)
				return
			}

			w.Write([]byte("{\"success\":true}\n"))
		})
	})

}
