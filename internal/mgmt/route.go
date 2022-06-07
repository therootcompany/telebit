package mgmt

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.rootprojects.org/root/telebit/internal/dbg"
	"git.rootprojects.org/root/telebit/internal/mgmt/authstore"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

// MgmtClaims includes a Slug, for backwards compatibility
type MgmtClaims struct {
	Slug string `json:"slug"`
	jwt.StandardClaims
}

var presenters = make(chan *Challenge)
var cleanups = make(chan *Challenge)

func RouteStatic(r chi.Router) chi.Router {
	r.Route("/", func(r chi.Router) {

		r.Use(middleware.Logger)
		r.Use(middleware.Timeout(15 * time.Second))
		r.Use(middleware.Recoverer)

		r.Get("/.well-known/acme-challenge/{token}", getACMEChallenges)
	})

	return r
}

func getACMEChallenges(w http.ResponseWriter, r *http.Request) {
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

func RouteAll(r chi.Router) {

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

	r.Route("/", func(r chi.Router) {
		r.Use(middleware.Logger)
		r.Use(middleware.Timeout(15 * time.Second))
		r.Use(middleware.Recoverer)

		r.Get("/.well-known/acme-challenge/{token}", getACMEChallenges)

		r.Route("/api", func(r chi.Router) {
			r.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")

					var tokenString string
					if auth := strings.Split(r.Header.Get("Authorization"), " "); len(auth) > 1 {
						// TODO handle Basic auth tokens as well
						tokenString = auth[1]
					}

					//var err2 error = nil
					tok, err := jwt.ParseWithClaims(
						tokenString,
						&MgmtClaims{},
						func(token *jwt.Token) (interface{}, error) {
							if dbg.Debug {
								fmt.Println("parsed jwt", token)
							}
							kid, ok := token.Header["kid"].(string)
							if !ok {
								return nil, fmt.Errorf("missing jwt header 'kid' (key id)")
							}
							auth, err := store.Get(kid)
							if nil != err {
								return nil, fmt.Errorf("invalid jwt header 'kid' (key id)")
							}

							claims := token.Claims.(*MgmtClaims)
							jti := claims.Id
							if "" == jti {
								return nil, fmt.Errorf("missing jwt payload 'jti' (jwt id / nonce)")
							}
							iat := claims.IssuedAt
							if 0 == iat {
								return nil, fmt.Errorf("missing jwt payload 'iat' (issued at)")
							}
							exp := claims.ExpiresAt
							if 0 == exp {
								return nil, fmt.Errorf("missing jwt payload 'exp' (expires at)")
							}

							if "" != claims.Slug {
								return nil, fmt.Errorf("extra jwt payload 'slug' (unknown)")
							}
							claims.Slug = auth.Slug
							if "" != claims.Subject && auth.Slug != claims.Subject {
								return nil, fmt.Errorf("invalid jwt payload 'sub' (mismatch)")
							}
							claims.Subject = claims.Slug
							claims.Issuer = DeviceDomain
							claims.Audience = fmt.Sprintf("wss://%s/ws", RelayDomain)

							/*
								// a little misdirection there
								mac := hmac.New(sha256.New, auth.MachinePPID)
								_ = mac.Write([]byte(auth.SharedKey))
								_ = mac.Write([]byte(fmt.Sprintf("%d", exp)))
								return []byte(auth.SharedKey), nil
							*/

							//fmt.Println("ppid:", auth.MachinePPID)

							return []byte(auth.MachinePPID), nil
						},
					)

					ctx := r.Context()
					if nil != tok {
						if tok.Valid {
							ctx = context.WithValue(ctx, MWKey("token"), tok)
							ctx = context.WithValue(ctx, MWKey("claims"), tok.Claims)
							ctx = context.WithValue(ctx, MWKey("valid"), true)
							fmt.Println("[auth] Token is fully valid:")
							fmt.Println(ctx.Value(MWKey("claims")))
						} else {
							fmt.Println("[auth] Token was parsed, but NOT valid:", tok)
						}
					}
					if nil != err {
						fmt.Println("[auth] Token is NOT valid due to error:", err)
						ctx = context.WithValue(ctx, MWKey("error"), err)
					}

					next.ServeHTTP(w, r.WithContext(ctx))
				})
			})

			handleACMEChallengeRoutes(r)
			handleDeviceRoutes(r)

			r.Get("/inspect", func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				claims, ok := ctx.Value(MWKey("claims")).(*MgmtClaims)
				if !ok {
					msg := `{"error":"failure to ping: 3", "code":"E_TOKEN"}`
					fmt.Println("touch no claims", claims)
					http.Error(w, msg+"\n", http.StatusBadRequest)
					return
				}

				w.Write([]byte(fmt.Sprintf(
					`{ "sub": "%s", "aud": "%s", "domains": [ "%s.%s" ], "ports": [] }`+"\n",
					claims.Subject,
					claims.Audience,
					claims.Slug,
					DeviceDomain,
				)))
			})

			r.Route("/register-device", func(r chi.Router) {
				// r.Use() // must NOT have slug '*'

				r.Post("/{otp}", func(w http.ResponseWriter, r *http.Request) {
					sharedKey := chi.URLParam(r, "otp")
					original, err := store.Get(sharedKey)
					if nil != err {
						msg := `{"error":"not found", "code":"E_NOT_FOUND"}`
						log.Printf("/api/register-device/\n")
						log.Println(err)
						http.Error(w, msg, http.StatusNotFound)
						return
					}
					if len(original.MachinePPID) > 0 {
						msg := `{"error":"the presented key has already been used", "code":"E_EXIST"}`
						log.Printf("/api/register-device/\n")
						log.Println(err)
						http.Error(w, msg, http.StatusUnprocessableEntity)
						return
					}

					auth := &authstore.Authorization{}
					decoder := json.NewDecoder(r.Body)
					err = decoder.Decode(&auth)
					// MachinePPID and PublicKey are required. ID must NOT be set. Slug is ignored.
					epoch := time.Time{}
					auth.SharedKey = sharedKey
					if nil != err || "" != auth.ID || "" == auth.MachinePPID ||
						"" == auth.PublicKey || "" == auth.SharedKey ||
						epoch != auth.CreatedAt || epoch != auth.UpdatedAt || epoch != auth.DeletedAt {
						msg, _ := json.Marshal(&struct {
							Error string `json:"error"`
						}{
							Error: "expected JSON in the format {\"machine_ppid\":\"\",\"public_key\":\"\"}",
						})
						http.Error(w, string(msg), http.StatusUnprocessableEntity)
						return
					}

					// TODO hash the PPID and check against the Public Key?
					pub := authstore.ToPublicKeyString(auth.MachinePPID)
					if pub != auth.PublicKey {
						msg, _ := json.Marshal(&struct {
							Error string `json:"error"`
						}{
							Error: "expected `public_key` to be the first 24 bytes of the hash of the `machine_ppid`",
						})
						http.Error(w, string(msg), http.StatusUnprocessableEntity)
						return
					}
					original.PublicKey = auth.PublicKey
					original.MachinePPID = auth.MachinePPID
					err = store.Set(original)
					if nil != err {
						msg := `{"error":"not really sure what happened, but it didn't go well (check the logs)", "code":"E_SERVER"}`
						log.Printf("/api/register-device/\n")
						log.Println(err)
						http.Error(w, msg, http.StatusInternalServerError)
						return
					}

					result, _ := json.Marshal(auth)
					w.Write(result)
				})
			})

			r.Post("/ping", func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				claims, ok := ctx.Value(MWKey("claims")).(*MgmtClaims)
				if !ok {
					msg := `{"error":"failure to ping: 1", "code":"E_TOKEN"}`
					http.Error(w, msg+"\n", http.StatusBadRequest)
					return
				}

				fmt.Println("ping pong??", claims)
				err := store.Touch(claims.Slug)
				if nil != err {
					fmt.Fprintf(os.Stderr, "touch err %s\n", err)
					var msg string
					if err == authstore.ErrNotFound {
						// if the token is valid, touch should ALWAYS work
						msg = `{"error":"failure to ping: inconsistent database data", "code":"E_SERVER"}`
					} else {
						msg = `{"error":"failure to ping: 2", "code":"E_SERVER"}`
					}
					http.Error(w, msg+"\n", http.StatusBadRequest)
					return
				}

				w.Write([]byte(`{ "success": true }` + "\n"))
			})

			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("Hello\n"))
			})
		})
	})
}
