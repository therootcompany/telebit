//go:generate go run -mod=vendor git.rootprojects.org/root/go-gitver

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-acme/lego/v3/challenge"
	"github.com/go-acme/lego/v3/providers/dns/duckdns"
	"github.com/go-acme/lego/v3/providers/dns/godaddy"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	_ "github.com/joho/godotenv/autoload"
)

var (
	// GitRev refers to the abbreviated commit hash
	GitRev = "0000000"
	// GitVersion refers to the most recent tag, plus any commits made since then
	GitVersion = "v0.0.0-pre0+0000000"
	// GitTimestamp refers to the timestamp of the most recent commit
	GitTimestamp = "0000-00-00T00:00:00+0000"
)

type MWKey string

func main() {
	var err error
	var provider challenge.Provider = nil // TODO is this concurrency-safe?
	var presenters = make(chan *Challenge)
	var cleanups = make(chan *Challenge)

	addr := flag.String("address", "", "IPv4 or IPv6 bind address")
	port := flag.String("port", "3000", "port to listen to")
	secret := flag.String("secret", "", "a >= 16-character random string for JWT key signing") // SECRET
	flag.Parse()

	if "" != os.Getenv("GODADDY_API_KEY") {
		id := os.Getenv("GODADDY_API_KEY")
		apiSecret := os.Getenv("GODADDY_API_SECRET")
		if provider, err = newGoDaddyDNSProvider(id, apiSecret); nil != err {
			panic(err)
		}
	} else if "" != os.Getenv("DUCKDNS_TOKEN") {
		if provider, err = newDuckDNSProvider(os.Getenv("DUCKDNS_TOKEN")); nil != err {
			panic(err)
		}
	} else {
		panic("Must provide either DUCKDNS or GODADDY credentials")
	}

	if "" == *secret {
		*secret = os.Getenv("SECRET")
	}
	if "" == *secret {
		fmt.Fprintf(os.Stderr, "Usage: signjwt <secret>")
		os.Exit(1)
		return
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Timeout(15 * time.Second))
	r.Use(middleware.Recoverer)

	r.Route("/api/dns", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var tokenString string
				if auth := strings.Split(r.Header.Get("Authorization"), " "); len(auth) > 1 {
					// TODO handle Basic auth tokens as well
					tokenString = auth[1]
				}
				if "" == tokenString {
					tokenString = r.URL.Query().Get("access_token")
				}

				// TODO check expiration and such
				tok, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
					return []byte(*secret), nil
				})
				if nil != err {
					fmt.Println("validation error:", tokenString, err)
					http.Error(w, "{\"error\":\"could not verify token\"}", http.StatusBadRequest)
					return
				}

				ctx := context.WithValue(r.Context(), MWKey("token"), tok)

				next.ServeHTTP(w, r.WithContext(ctx))
			})
		})

		r.Post("/{domain}", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			ch := Challenge{}

			// TODO prevent slow loris
			decoder := json.NewDecoder(r.Body)
			err := decoder.Decode(&ch)
			if nil != err || "" == ch.Token || "" == ch.KeyAuth {
				msg := `{"error":"expected json in the format {\"token\":\"xxx\",\"key_authorization\":\"yyy\"}"}`
				http.Error(w, msg, http.StatusUnprocessableEntity)
				return
			}

			domain := chi.URLParam(r, "domain")
			//domain := chi.URLParam(r, "*")
			ch.Domain = domain

			// TODO some additional error checking before the handoff
			//ch.error = make(chan error, 1)
			ch.error = make(chan error)
			presenters <- &ch
			err = <-ch.error
			if nil != err || "" == ch.Token || "" == ch.KeyAuth {
				msg := `{"error":"expected json in the format {\"token\":\"xxx\",\"key_authorization\":\"yyy\"}"}`
				http.Error(w, msg, http.StatusUnprocessableEntity)
				return
			}

			w.Write([]byte("{\"success\":true}\n"))
		})

		// TODO ugly Delete, but whatever
		r.Delete("/{domain}/{token}/{keyAuth}", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			ch := Challenge{
				Domain:  chi.URLParam(r, "domain"),
				Token:   chi.URLParam(r, "token"),
				KeyAuth: chi.URLParam(r, "keyAuth"),
				error:   make(chan error),
				//error:   make(chan error, 1),
			}

			cleanups <- &ch
			err = <-ch.error
			if nil != err || "" == ch.Token || "" == ch.KeyAuth {
				msg := `{"error":"expected json in the format {\"token\":\"xxx\",\"key_authorization\":\"yyy\"}"}`
				http.Error(w, msg, http.StatusUnprocessableEntity)
				return
			}

			w.Write([]byte("{\"success\":true}\n"))
		})
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("welcome\n"))
	})

	go func() {
		for {
			// TODO make parallel?
			// TODO make cancellable?
			ch := <-presenters
			err := provider.Present(ch.Domain, ch.Token, ch.KeyAuth)
			ch.error <- err
		}
	}()

	go func() {
		for {
			// TODO make parallel?
			// TODO make cancellable?
			ch := <-cleanups
			ch.error <- provider.CleanUp(ch.Domain, ch.Token, ch.KeyAuth)
		}
	}()

	bind := *addr + ":" + *port
	fmt.Println("Listening on", bind)
	fmt.Fprintf(os.Stderr, "failed:", http.ListenAndServe(bind, r))
}

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

// newDuckDNSProvider is for the sake of demoing the tunnel
func newDuckDNSProvider(token string) (*duckdns.DNSProvider, error) {
	config := duckdns.NewDefaultConfig()
	config.Token = token
	return duckdns.NewDNSProviderConfig(config)
}

// newGoDaddyDNSProvider is for the sake of demoing the tunnel
func newGoDaddyDNSProvider(id, secret string) (*godaddy.DNSProvider, error) {
	config := godaddy.NewDefaultConfig()
	config.APIKey = id
	config.APISecret = secret
	return godaddy.NewDNSProviderConfig(config)
}
