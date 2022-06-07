package acmeroutes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"git.rootprojects.org/root/telebit/internal/authutil"
	"git.rootprojects.org/root/telebit/internal/http01fs"

	"github.com/go-acme/lego/v3/challenge"
	"github.com/go-chi/chi"
	"github.com/mholt/acmez/acme"
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

/*
type acmeProvider struct {
	BaseURL  string
	provider challenge.Provider
}
*/

var provider challenge.Provider = nil
var presenters = make(chan *Challenge)
var cleanups = make(chan *Challenge)

// Init initializes some package variables
func Init(p challenge.Provider) {
	provider = p

	go func() {
		for {
			// TODO make parallel?
			// TODO make cancellable?
			ch := <-presenters
			if nil != provider {
				err := provider.Present(ch.Domain, ch.Token, ch.KeyAuth)
				ch.error <- err
			} else {
				ch.error <- fmt.Errorf("missing acme challenge provider for present")
			}
		}
	}()

	go func() {
		for {
			// TODO make parallel?
			// TODO make cancellable?
			ch := <-cleanups
			if nil != provider {
				ch.error <- provider.CleanUp(ch.Domain, ch.Token, ch.KeyAuth)
			} else {
				ch.error <- fmt.Errorf("missing acme challenge provider for cleanup")
			}
		}
	}()
}

/*
func (p *acmeProvider) Present(domain, token, keyAuth string) error {
	return p.provider.Present(domain, token, keyAuth)
}

func (p *acmeProvider) CleanUp(domain, token, keyAuth string) error {
	return p.provider.CleanUp(domain, token, keyAuth)
}
*/

// GetACMEChallenges fetches stored HTTP-01 challenges
func GetACMEChallenges(w http.ResponseWriter, r *http.Request) {
	//token := chi.URLParam(r, "token")
	host := r.Host
	/*
		// TODO TrustProxy option?
		xHost := r.Header.Get("X-Forwarded-Host")
		//log.Printf("[debug] Host: %q\n[debug] X-Host: %q", host, xHost)
		if len(xHost) > 0 {
			host = xHost
		}
	*/

	// disallow FS characters
	if strings.ContainsAny(host, "/:|\\") {
		host = ""
	}
	tokenPath := filepath.Join(tmpBase, host)

	fsrv := http.FileServer(http.Dir(tokenPath))
	fsrv.ServeHTTP(w, r)
}

// HandleACMEChallengeRoutes allows storing ACME challenges for relay
func HandleACMEChallengeRoutes(r chi.Router) {
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

func isSlugAllowed(domain, slug string) bool {
	if "*" == slug {
		return true
	}
	// ex: "abc.devices.example.com" has prefix "abc."
	return strings.HasPrefix(domain, slug+".")
}

func createChallenge(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")

	ctx := r.Context()
	claims, ok := ctx.Value(authutil.MWKey("claims")).(*authutil.Claims)
	if !ok || !isSlugAllowed(domain, claims.Slug) {
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
		http01Provider := &http01fs.Provider
		http01Provider.Present(context.Background(), acme.Challenge{
			Token:            ch.Token,
			KeyAuthorization: ch.KeyAuth,
			Identifier: acme.Identifier{
				Value: ch.Domain,
				Type:  "dns", // TODO is this correct??
			},
		})
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
		Type:    chi.URLParam(r, "challengeType"),
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
		http01Provider := &http01fs.Provider
		http01Provider.CleanUp(context.Background(), acme.Challenge{
			Token:            ch.Token,
			KeyAuthorization: ch.KeyAuth,
			Identifier: acme.Identifier{
				Value: ch.Domain,
				Type:  "dns", // TODO is this correct??
			},
		})
	} else {
		// TODO what if DNS-01 is not enabled?
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
