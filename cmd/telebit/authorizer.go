package main

import (
	"fmt"
	"net/http"
	"strings"

	telebit "git.coolaj86.com/coolaj86/go-telebitd/mplexer"
)

func NewAuthorizer(authURL string) telebit.Authorizer {
	return func(r *http.Request) (*telebit.Grants, error) {
		// do we have a valid wss_client?

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

		grants, err := telebit.Inspect(authURL, tokenString)

		if nil != err {
			fmt.Println("return an error, do not go on")
			return nil, err
		}
		if "" != r.URL.Query().Get("access_token") {
			r.URL.Query().Set("access_token", "[redacted:"+grants.Subject+"]")
		}

		return grants, err
	}
}
