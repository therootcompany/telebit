package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-acme/lego/v3/challenge"
	"github.com/go-chi/chi"
)

// A Challenge has the data necessary to create an ACME DNS-01 Key Authorization Digest.
type Challenge struct {
	Domain  string `json:"domain"`
	Token   string `json:"token"`
	KeyAuth string `json:"key_authorization"`
	error   chan error
}

type acmeProvider struct {
	BaseURL  string
	provider challenge.Provider
}

func (p *acmeProvider) Present(domain, token, keyAuth string) error {
	return p.provider.Present(domain, token, keyAuth)
}

func (p *acmeProvider) CleanUp(domain, token, keyAuth string) error {
	return p.provider.CleanUp(domain, token, keyAuth)
}

func handleDNSRoutes(r chi.Router) {
	r.Route("/dns", func(r chi.Router) {
		r.Post("/{domain}", func(w http.ResponseWriter, r *http.Request) {
			domain := chi.URLParam(r, "domain")

			ctx := r.Context()
			claims, ok := ctx.Value(MWKey("claims")).(*MgmtClaims)
			if !ok || !strings.HasPrefix(domain+".", claims.Slug) {
				msg := `{ "error": "invalid domain", "code":"E_BAD_REQUEST"}`
				http.Error(w, msg+"\n", http.StatusUnprocessableEntity)
				return
			}

			ch := Challenge{}

			// TODO prevent slow loris
			decoder := json.NewDecoder(r.Body)
			err := decoder.Decode(&ch)
			if nil != err || "" == ch.Token || "" == ch.KeyAuth {
				msg := `{"error":"expected json in the format {\"token\":\"xxx\",\"key_authorization\":\"yyy\"}", "code":"E_BAD_REQUEST"}`
				http.Error(w, msg, http.StatusUnprocessableEntity)
				return
			}

			//domain := chi.URLParam(r, "*")
			ch.Domain = domain

			// TODO some additional error checking before the handoff
			//ch.error = make(chan error, 1)
			ch.error = make(chan error)
			presenters <- &ch
			err = <-ch.error
			if nil != err || "" == ch.Token || "" == ch.KeyAuth {
				fmt.Println("presenter err", err, ch.Token, ch.KeyAuth)
				msg := `{"error":"ACME dns-01 error", "code":"E_SERVER"}`
				http.Error(w, msg, http.StatusUnprocessableEntity)
				return
			}

			w.Write([]byte("{\"success\":true}\n"))
		})

		// TODO ugly Delete, but whatever
		r.Delete("/{domain}/{token}/{keyAuth}", func(w http.ResponseWriter, r *http.Request) {
			// TODO authenticate

			ch := Challenge{
				Domain:  chi.URLParam(r, "domain"),
				Token:   chi.URLParam(r, "token"),
				KeyAuth: chi.URLParam(r, "keyAuth"),
				error:   make(chan error),
				//error:   make(chan error, 1),
			}

			cleanups <- &ch
			err := <-ch.error
			if nil != err || "" == ch.Token || "" == ch.KeyAuth {
				msg := `{"error":"expected json in the format {\"token\":\"xxx\",\"key_authorization\":\"yyy\"}", "code":"E_BAD_REQUEST"}`
				http.Error(w, msg, http.StatusUnprocessableEntity)
				return
			}

			w.Write([]byte("{\"success\":true}\n"))
		})
	})
}
