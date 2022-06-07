package authutil

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"git.rootprojects.org/root/telebit/internal/dbg"
)

// Authorizer is called when a new client connects and we need to know something about it
type Authorizer func(*http.Request) (*Grants, error)

var NotAuthorizedContent = []byte("{ \"error\": \"not authorized\" }\n")

// NewAuthorizer will create a new (proxiable) token verifier
func NewAuthorizer(authURL string) Authorizer {
	return func(r *http.Request) (*Grants, error) {
		// do we have a valid wss_client?

		fmt.Printf("[authz] Authorization = %s\n", r.Header.Get("Authorization"))
		var tokenString string
		if auth := strings.Split(r.Header.Get("Authorization"), " "); len(auth) > 1 {
			// TODO handle Basic auth tokens as well
			tokenString = auth[1]
		}
		if "" == tokenString {
			// Browsers do not allow Authorization Headers and must use access_token query string
			tokenString = r.URL.Query().Get("access_token")
		}
		if "" != r.URL.Query().Get("access_token") {
			r.URL.Query().Set("access_token", "[redacted]")
		}

		fmt.Printf("[authz] authURL = %s\n", authURL)
		fmt.Printf("[authz] token = %s\n", tokenString)
		grants, err := Inspect(authURL, tokenString)

		if nil != err {
			fmt.Printf("[authorizer] error inspecting %q: %s\ntoken: %s\n", authURL, err, tokenString)
			return nil, err
		}
		if "" != r.URL.Query().Get("access_token") {
			r.URL.Query().Set("access_token", "[redacted:"+grants.Subject+"]")
		}

		return grants, err
	}
}

// Grants are verified token Claims
type Grants struct {
	Subject  string   `json:"sub"`
	Audience string   `json:"aud"`
	Domains  []string `json:"domains"`
	Ports    []int    `json:"ports"`
}

// Inspect will verify a token and return its details, decoded
func Inspect(authURL, token string) (*Grants, error) {
	inspectURL := strings.TrimSuffix(authURL, "/inspect") + "/inspect"
	if dbg.Debug {
		fmt.Fprintf(os.Stderr, "[debug] telebit.Inspect(\n\tinspectURL = %s,\n\ttoken = %s,\n)\n", inspectURL, token)
	}
	msg, err := Request("GET", inspectURL, token, nil)
	if nil != err {
		return nil, err
	}
	if nil == msg {
		return nil, fmt.Errorf("invalid response")
	}

	grants := &Grants{}
	err = json.NewDecoder(msg).Decode(grants)
	if err != nil {
		return nil, err
	}
	if "" == grants.Subject {
		fmt.Fprintf(os.Stderr, "TODO update mgmt server to show Subject: %q\n", msg)
		grants.Subject = strings.Split(grants.Domains[0], ".")[0]
	}
	return grants, nil
}
