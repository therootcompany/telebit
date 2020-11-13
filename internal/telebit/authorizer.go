package telebit

import (
	"fmt"
	"net/http"
	"strings"

	"git.rootprojects.org/root/telebit"
)

func NewAuthorizer(authURL string) telebit.Authorizer {
	return func(r *http.Request) (*telebit.Grants, error) {
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
		grants, err := telebit.Inspect(authURL, tokenString)

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
