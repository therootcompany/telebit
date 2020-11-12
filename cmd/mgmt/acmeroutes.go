package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-acme/lego/v3/challenge"
	"github.com/go-chi/chi"
)

const (
	tmpBase      = "acme-tmp"
	challengeDir = ".well-known/acme-challenge"
)

// Challenge is an ACME http-01 challenge
type Challenge struct {
	Type       string     `json:"type"`
	Token      string     `json:"token"`
	KeyAuth    string     `json:"key_authorization"`
	Identifier Identifier `json:"identifier"`
	// for the old one
	Domain string `json:"domain"`
	error  chan error
}

// Identifier is restricted to DNS Domain Names for now
type Identifier struct {
	Type  string `json:"type"`
	Value string `json:"value"`
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
	handleACMEChallenges := func(r chi.Router) {
		r.Post("/{domain}", createChallenge)

		// TODO ugly Delete, but whatever
		r.Delete("/{domain}/{token}/{keyAuth}", deleteChallenge)
		r.Delete("/{domain}/{token}/{keyAuth}/{challengeType}", deleteChallenge)
	}

	// TODO pick one and stick with it
	r.Route("/acme-relay", handleACMEChallenges)
	r.Route("/acme-solver", handleACMEChallenges)
	r.Route("/dns", handleACMEChallenges)
	r.Route("/http", handleACMEChallenges)
}

func createChallenge(w http.ResponseWriter, r *http.Request) {
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
	ch.Identifier.Value = domain

	if "" == ch.Token || "" == ch.KeyAuth {
		err = errors.New("missing token and/or key auth")
	} else if strings.Contains(ch.Type, "http") {
		challengeBase := filepath.Join(tmpBase, ch.Domain, ".well-known/acme-challenge")
		_ = os.MkdirAll(challengeBase, 0700)
		tokenPath := filepath.Join(challengeBase, ch.Token)
		err = ioutil.WriteFile(tokenPath, []byte(ch.KeyAuth), 0600)
	} else {
		// TODO some additional error checking before the handoff
		//ch.error = make(chan error, 1)
		ch.error = make(chan error)
		presenters <- &ch
		err = <-ch.error
	}

	if nil != err {
		fmt.Println("presenter err", err, ch.Token, ch.KeyAuth)
		msg := `{"error":"ACME dns-01 error", "code":"E_SERVER"}`
		http.Error(w, msg, http.StatusUnprocessableEntity)
		return
	}

	w.Write([]byte("{\"success\":true}\n"))
}

func deleteChallenge(w http.ResponseWriter, r *http.Request) {
	// TODO authenticate

	ch := Challenge{
		Domain:  chi.URLParam(r, "domain"),
		Token:   chi.URLParam(r, "token"),
		KeyAuth: chi.URLParam(r, "keyAuth"),
		error:   make(chan error),
		//error:   make(chan error, 1),
	}

	var err error
	if "" == ch.Token || "" == ch.KeyAuth {
		err = errors.New("missing token and/or key auth")
	} else if strings.Contains(ch.Type, "http") {
		// always try to remove, as there's no harm
		tokenPath := filepath.Join(tmpBase, ch.Domain, challengeDir, ch.Token)
		_ = os.Remove(tokenPath)
	} else {
		cleanups <- &ch
		err = <-ch.error
	}

	if nil != err {
		msg := `{"error":"expected json in the format {\"token\":\"xxx\",\"key_authorization\":\"yyy\"}", "code":"E_BAD_REQUEST"}`
		http.Error(w, msg, http.StatusUnprocessableEntity)
		return
	}

	w.Write([]byte("{\"success\":true}\n"))
}
