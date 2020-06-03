package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"git.coolaj86.com/coolaj86/go-telebitd/mgmt/authstore"
	"github.com/go-chi/chi"
)

func handleDeviceRoutes(r chi.Router) {
	r.Route("/devices", func(r chi.Router) {
		// only the admin can get past this point
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				claims, ok := ctx.Value(MWKey("claims")).(*MgmtClaims)
				if !ok || "*" != claims.Slug {
					msg := `{"error":"missing or invalid authorization token"}`
					http.Error(w, msg+"\n", http.StatusUnprocessableEntity)
					return
				}

				next.ServeHTTP(w, r.WithContext(ctx))
			})
		})

		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			auth := &authstore.Authorization{}

			// Slug is mandatory, ID and MachinePPID must NOT be set
			decoder := json.NewDecoder(r.Body)
			err := decoder.Decode(&auth)
			epoch := time.Time{}
			if nil != err || "" != auth.ID || "" != auth.MachinePPID || "" == auth.Slug ||
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

			if "" == auth.SharedKey {
				rnd := make([]byte, 16)
				if _, err := rand.Read(rnd); nil != err {
					panic(err)
				}
				auth.SharedKey = base64.RawURLEncoding.EncodeToString(rnd)
			}
			if len(auth.SharedKey) < 20 {
				msg := `{"error":"shared_key must be >= 16 bytes"}`
				http.Error(w, string(msg), http.StatusUnprocessableEntity)
				return
			}

			pub := authstore.ToPublicKeyString(auth.SharedKey)
			if "" == auth.PublicKey {
				auth.PublicKey = pub
			}
			if len(auth.PublicKey) > 24 {
				auth.PublicKey = auth.PublicKey[:24]
			}
			if pub != auth.PublicKey {
				msg := `{"error":"public_key must be the first 24 bytes of the base64-encoded hash of the shared_key"}`
				http.Error(w, msg+"\n", http.StatusUnprocessableEntity)
				return
			}

			err = store.Add(auth)
			if nil != err {
				msg := `{"error":"not really sure what happened, but it didn't go well (check the logs)"}`
				if authstore.ErrExists == err {
					msg = fmt.Sprintf(`{ "error": "%s" }`, err.Error())
				}
				log.Printf("/api/devices/\n")
				log.Println(err)
				http.Error(w, msg, http.StatusInternalServerError)
				return
			}

			result, _ := json.Marshal(auth)
			w.Write([]byte(string(result) + "\n"))
		})

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			var things []authstore.Authorization
			var err error
			if "true" == strings.Join(r.URL.Query()["inactive"], " ") {
				things, err = store.Inactive()
			} else {
				things, err = store.Active()
			}
			if nil != err {
				msg := `{"error":"not really sure what happened, but it didn't go well (check the logs)"}`
				log.Printf("/api/devices/\n")
				log.Println(err)
				http.Error(w, msg, http.StatusInternalServerError)
				return
			}

			for i := range things {
				auth := things[i]
				// Redact private data
				if "" != auth.MachinePPID {
					auth.MachinePPID = "[redacted]"
				}
				if "" != auth.SharedKey {
					auth.SharedKey = "[redacted]"
				}
				things[i] = auth
			}

			encoder := json.NewEncoder(w)
			encoder.SetEscapeHTML(true)
			_ = encoder.Encode(things)
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
			w.Write([]byte(string(result) + "\n"))
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

			w.Write([]byte(`{"success":true}` + "\n"))
		})
	})

}
