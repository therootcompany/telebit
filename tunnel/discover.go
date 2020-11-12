package tunnel

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// Endpoints represents the endpoints listed in the API service directory.
// Ideally the Relay URL is discoverable and will provide the authn endpoint,
// which will provide the tunnel endpoint. However, for the sake of testing,
// these things may happen out-of-order.
type Endpoints struct {
	ToS            string   `json:"terms_of_service"`
	APIHost        string   `json:"api_host"`
	Tunnel         Endpoint `json:"tunnel"`
	Authenticate   Endpoint `json:"authn"`
	ChallengeProxy Endpoint `json:"acme_challenge_proxy"`
	DNS01Proxy     Endpoint `json:"acme_dns_01_proxy"`
	HTTP01Proxy    Endpoint `json:"acme_http_01_proxy"`
	/*
		{
			"terms_of_service": ":hostname/tos/",
			"api_host": ":hostname/api",
			"pair_request": {
				"method": "POST",
				"pathname": "api/telebit.app/pair_request"
			}
		}
	*/
}

// Endpoint represents a URL Request
type Endpoint struct {
	URL      string `json:"url,omitempty"`
	Method   string `json:"method,omitempty"`
	Scheme   string `json:"scheme,omitempty"`
	Host     string `json:"host,omitempty"`
	Pathname string `json:"pathname"`
}

// Discover checks the .well-known directory for service endpoints
func Discover(relay string) (*Endpoints, error) {
	directives := &Endpoints{}
	relayURL, err := url.Parse(relay)
	if nil != err {
		return nil, err
	}
	if "ws" != relayURL.Scheme[:2] {

		if '/' != relayURL.Path[len(relayURL.Path)-1] {
			relayURL.Path += "/"
		}
		resp, err := http.Get(relayURL.String() + ".well-known/telebit.app/index.json")
		if nil != err {
			return nil, err
		}
		b, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if nil != err {
			return nil, err
		}
		body := strings.Replace(string(b), ":hostname", relayURL.Host, -1)
		err = json.Unmarshal([]byte(body), directives)
		if nil != err {
			return nil, err
		}

		directives.Tunnel.URL = endpointToURLString(directives.APIHost, directives.Tunnel)

	} else {

		directives.Tunnel.URL = relayURL.String()
		directives.APIHost = relayURL.Host

	}

	directives.Authenticate.URL = endpointToURLString(directives.APIHost, directives.Authenticate)
	if len(directives.ChallengeProxy.Pathname) > 0 {
		directives.ChallengeProxy.URL = endpointToURLString(directives.APIHost, directives.ChallengeProxy)
	}
	if len(directives.DNS01Proxy.Pathname) > 0 {
		directives.DNS01Proxy.URL = endpointToURLString(directives.APIHost, directives.DNS01Proxy)
	}
	if len(directives.HTTP01Proxy.Pathname) > 0 {
		directives.HTTP01Proxy.URL = endpointToURLString(directives.APIHost, directives.HTTP01Proxy)
	}

	return directives, nil
}

func endpointToURLString(apiHost string, endpoint Endpoint) string {
	pathname := endpoint.Pathname
	/*
		if "" == pathname {
			return ""
		}
	*/

	host := endpoint.Host
	if "" == host {
		host = apiHost
	}

	scheme := endpoint.Scheme
	if "" == scheme {
		scheme = "https:"
	}

	if "" == pathname {
		return fmt.Sprintf("%s//%s", scheme, host)
	}
	return fmt.Sprintf("%s//%s/%s", scheme, host, pathname)
}
