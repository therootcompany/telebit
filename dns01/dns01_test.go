package dns01

// Adapted from https://github.com/go-acme/lego/blob/master/providers/dns/httpreq/httpreq.go

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strings"
	"testing"

	"github.com/go-acme/lego/v3/platform/tester"
	"github.com/stretchr/testify/require"
)

var envTest = tester.NewEnvTest(EnvEndpoint, EnvToken)

func TestNewDNSProvider(t *testing.T) {
	testCases := []struct {
		desc     string
		envVars  map[string]string
		expected string
	}{
		{
			desc: "success",
			envVars: map[string]string{
				EnvEndpoint: "http://localhost:8090",
			},
		},
		{
			desc: "invalid URL",
			envVars: map[string]string{
				EnvEndpoint: ":",
			},
			expected: `api: parse ":": missing protocol scheme`,
		},
		{
			desc: "missing endpoint",
			envVars: map[string]string{
				EnvEndpoint: "",
			},
			expected: "api: some credentials information are missing: API_ENDPOINT",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			defer envTest.RestoreEnv()
			envTest.ClearEnv()

			envTest.Apply(test.envVars)

			p, err := NewDNSProvider()

			if len(test.expected) == 0 {
				require.NoError(t, err)
				require.NotNil(t, p)
				require.NotNil(t, p.config)
			} else {
				require.EqualError(t, err, test.expected)
			}
		})
	}
}

func TestNewDNSProviderConfig(t *testing.T) {
	testCases := []struct {
		desc     string
		endpoint *url.URL
		expected string
	}{
		{
			desc:     "success",
			endpoint: mustParse("http://localhost:8090"),
		},
		{
			desc:     "missing endpoint",
			expected: "api: the endpoint is missing",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			config := NewDefaultConfig()
			config.Endpoint = test.endpoint

			p, err := NewDNSProviderConfig(config)

			if len(test.expected) == 0 {
				require.NoError(t, err)
				require.NotNil(t, p)
				require.NotNil(t, p.config)
			} else {
				require.EqualError(t, err, test.expected)
			}
		})
	}
}

func TestNewDNSProvider_Present(t *testing.T) {
	envTest.RestoreEnv()

	testCases := []struct {
		desc          string
		token         string
		pathPrefix    string
		handler       http.HandlerFunc
		expectedError string
	}{
		{
			desc:    "success",
			handler: successHandler,
		},
		{
			desc:       "success with path prefix",
			handler:    successHandler,
			pathPrefix: "/api/acme/",
		},
		{
			desc:          "error",
			handler:       http.NotFound,
			expectedError: "api: 404: request failed: 404 page not found\n",
		},
		{
			desc:    "success raw mode",
			handler: successRawModeHandler,
		},
		{
			desc:          "error raw mode",
			handler:       http.NotFound,
			expectedError: "api: 404: request failed: 404 page not found\n",
		},
		{
			desc:  "bearer auth",
			token: "foobar",
			handler: func(rw http.ResponseWriter, req *http.Request) {
				token := strings.Replace(req.Header.Get("Authorization"), "Bearer ", "", 1)
				if token != "foobar" {
					rw.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, "Please enter your username and password."))
					http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
					return
				}

				fmt.Fprint(rw, "lego")
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			mux := http.NewServeMux()
			hostname := "domain"
			mux.HandleFunc(path.Join("/", test.pathPrefix, "/"+hostname), test.handler)
			server := httptest.NewServer(mux)

			config := NewDefaultConfig()
			config.Endpoint = mustParse(server.URL + test.pathPrefix)
			config.Token = test.token

			p, err := NewDNSProviderConfig(config)
			require.NoError(t, err)

			err = p.Present("domain", "token", "key")
			if test.expectedError == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, test.expectedError)
			}
		})
	}
}

func TestNewDNSProvider_Cleanup(t *testing.T) {
	envTest.RestoreEnv()

	testCases := []struct {
		desc          string
		token         string
		handler       http.HandlerFunc
		expectedError string
	}{
		{
			desc:    "success",
			handler: successHandler,
		},
		{
			desc:          "error",
			handler:       http.NotFound,
			expectedError: "api: 404: request failed: 404 page not found\n",
		},
		{
			desc:    "success raw mode",
			handler: successRawModeHandler,
		},
		{
			desc:          "error raw mode",
			handler:       http.NotFound,
			expectedError: "api: 404: request failed: 404 page not found\n",
		},
		{
			desc:  "basic auth",
			token: "foobar",
			handler: func(rw http.ResponseWriter, req *http.Request) {
				token := strings.Replace(req.Header.Get("Authorization"), "Bearer ", "", 1)
				if token != "foobar" {
					rw.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, "Please enter your username and password."))
					http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
					return
				}
				fmt.Fprint(rw, "lego")
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			hostname := "domain"
			dnsToken := "token"
			dnsKeyAuth := "key"
			mux := http.NewServeMux()
			mux.HandleFunc(
				fmt.Sprintf("/%s/%s/%s", hostname, dnsToken, dnsKeyAuth),
				test.handler,
			)
			server := httptest.NewServer(mux)

			config := NewDefaultConfig()
			config.Endpoint = mustParse(server.URL)
			config.Token = test.token

			p, err := NewDNSProviderConfig(config)
			require.NoError(t, err)

			err = p.CleanUp("domain", "token", "key")
			if test.expectedError == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, test.expectedError)
			}
		})
	}
}

func successHandler(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost && req.Method != http.MethodDelete {
		http.Error(rw, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	msg := &dnsChallenge{}
	err := json.NewDecoder(req.Body).Decode(msg)
	if err != nil {
		if !(req.Method == http.MethodDelete && io.EOF == err) {
			http.Error(rw, err.Error(), http.StatusBadRequest)
		}
		return
	}

	fmt.Fprint(rw, "lego")
}

func successRawModeHandler(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost && req.Method != http.MethodDelete {
		http.Error(rw, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	msg := &dnsChallenge{}
	err := json.NewDecoder(req.Body).Decode(msg)
	if err != nil {
		if !(req.Method == http.MethodDelete && io.EOF == err) {
			http.Error(rw, err.Error(), http.StatusBadRequest)
		}
		return
	}

	fmt.Fprint(rw, "lego")
}

func mustParse(rawURL string) *url.URL {
	uri, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return uri
}
