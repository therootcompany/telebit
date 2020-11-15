// Package dns01 implements a DNS provider for solving the DNS-01 challenge through a HTTP server.
package dns01

// Adapted from https://github.com/go-acme/lego/blob/master/providers/dns/httpreq/httpreq.go

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/coolaj86/certmagic"
	"github.com/go-acme/lego/v3/challenge"
	"github.com/go-acme/lego/v3/challenge/dns01"
	"github.com/go-acme/lego/v3/platform/config/env"
	"github.com/mholt/acmez/acme"
)

// Environment variables names.
const (
	envNamespace = "API_"

	EnvEndpoint = envNamespace + "ENDPOINT"
	EnvToken    = envNamespace + "TOKEN"

	EnvPropagationTimeout = envNamespace + "PROPAGATION_TIMEOUT"
	EnvPollingInterval    = envNamespace + "POLLING_INTERVAL"
	EnvHTTPTimeout        = envNamespace + "HTTP_TIMEOUT"
)

type dnsChallenge struct {
	Domain        string `json:"domain"`
	Hostname      string `json:"hostname"`
	Token         string `json:"token"`
	KeyAuth       string `json:"key_authorization"`
	KeyAuthDigest string `json:"key_authorization_digest"`
}

// Tokener returns a fresh, valid token
//type Tokener func() string

// Config is used to configure the creation of the DNSProvider.
type Config struct {
	Endpoint           *url.URL
	Tokener            func() string
	PropagationTimeout time.Duration
	PollingInterval    time.Duration
	HTTPClient         *http.Client
}

// NewDefaultConfig returns a default configuration for the DNSProvider.
func NewDefaultConfig() *Config {
	return &Config{
		PropagationTimeout: env.GetOrDefaultSecond(EnvPropagationTimeout, dns01.DefaultPropagationTimeout),
		PollingInterval:    env.GetOrDefaultSecond(EnvPollingInterval, dns01.DefaultPollingInterval),
		HTTPClient: &http.Client{
			Timeout: env.GetOrDefaultSecond(EnvHTTPTimeout, 15*time.Second),
		},
	}
}

// DNSProvider implements the challenge.Provider interface.
type DNSProvider struct {
	config *Config
}

// NewDNSProvider returns a DNSProvider instance.
func NewDNSProvider() (*DNSProvider, error) {
	values, err := env.Get(EnvEndpoint)
	if err != nil {
		return nil, fmt.Errorf("dns01 api: %w", err)
	}

	endpoint, err := url.Parse(values[EnvEndpoint])
	if err != nil {
		return nil, fmt.Errorf("dns01 api: %w", err)
	}

	config := NewDefaultConfig()
	//config.Token = env.GetOrFile(EnvToken)
	config.Endpoint = endpoint
	return NewDNSProviderConfig(config)
}

// NewDNSProviderConfig return a DNSProvider.
func NewDNSProviderConfig(config *Config) (*DNSProvider, error) {
	if config == nil {
		return nil, errors.New("api: the configuration of the DNS provider is nil")
	}

	if config.Endpoint == nil {
		return nil, errors.New("api: the endpoint is missing")
	}

	return &DNSProvider{config: config}, nil
}

// Timeout returns the timeout and interval to use when checking for DNS propagation.
// Adjusting here to cope with spikes in propagation times.
func (d *DNSProvider) Timeout() (timeout, interval time.Duration) {
	return d.config.PropagationTimeout, d.config.PollingInterval
}

// Present creates a TXT record to fulfill the dns-01 challenge.
func (d *DNSProvider) Present(domain, token, keyAuth string) error {
	msg := getDNSChallenge(domain, token, keyAuth)

	err := d.doRequest(http.MethodPost, fmt.Sprintf("/%s", msg.Domain), msg)
	if err != nil {
		return fmt.Errorf("api: %w", err)
	}
	return nil
}

// CleanUp removes the TXT record matching the specified parameters.
func (d *DNSProvider) CleanUp(domain, token, keyAuth string) error {
	msg := getDNSChallenge(domain, token, keyAuth)

	err := d.doRequest(
		http.MethodDelete,
		fmt.Sprintf("/%s/%s/%s", msg.Domain, msg.Token, msg.KeyAuth),
		nil,
	)
	if err != nil {
		return fmt.Errorf("api: %w", err)
	}
	return nil
}

func getDNSChallenge(domain, token, keyAuth string) *dnsChallenge {
	hostname, digest := dns01.GetRecord(domain, keyAuth)
	return &dnsChallenge{
		Domain:        domain,
		Hostname:      hostname,
		Token:         token,
		KeyAuth:       keyAuth,
		KeyAuthDigest: digest,
	}
}

func (d *DNSProvider) doRequest(method, uri string, msg interface{}) error {
	reqBody := &bytes.Buffer{}
	if nil != msg {
		err := json.NewEncoder(reqBody).Encode(msg)
		if err != nil {
			return err
		}
	}

	newURI := path.Join(d.config.Endpoint.EscapedPath(), uri)
	endpoint, err := d.config.Endpoint.Parse(newURI)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(method, endpoint.String(), reqBody)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	apiToken := d.config.Tokener()
	if len(apiToken) > 0 {
		req.Header.Set("Authorization", "Bearer "+apiToken)
	}

	resp, err := d.config.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("%d: failed to read response body: %w", resp.StatusCode, err)
		}

		return fmt.Errorf("%d: request failed: %v", resp.StatusCode, string(body))
	}

	return nil
}

// NewSolver creates a new Solver
func NewSolver(provider challenge.Provider) *Solver {
	return &Solver{
		provider:   provider,
		dnsChecker: certmagic.DNS01Solver{},
	}
}

// Solver wraps a Lego DNS Provider for CertMagic
type Solver struct {
	provider challenge.Provider
	//option   legoDns01.ChallengeOption
	dnsChecker certmagic.DNS01Solver
}

// Present creates a DNS-01 Challenge Token
func (s *Solver) Present(ctx context.Context, ch acme.Challenge) error {
	log.Println("Present DNS-01 challenge solution for", ch.Identifier.Value)
	return s.provider.Present(ch.Identifier.Value, ch.Token, ch.KeyAuthorization)
}

// CleanUp deletes a DNS-01 Challenge Token
func (s *Solver) CleanUp(ctx context.Context, ch acme.Challenge) error {
	log.Println("CleanUp DNS-01 challenge solution for", ch.Identifier.Value)
	c := make(chan error)
	go func() {
		c <- s.provider.CleanUp(ch.Identifier.Value, ch.Token, ch.KeyAuthorization)
	}()
	select {
	case err := <-c:
		return err
	case <-ctx.Done():
		return errors.New("cancelled")
	}
}

// Wait blocks until the TXT record created in Present() appears in
// authoritative lookups, i.e. until it has propagated, or until
// timeout, whichever is first.
func (s *Solver) Wait(ctx context.Context, ch acme.Challenge) error {
	log.Println("Wait on DNS-01 challenge self-verification for", ch.Identifier.Value)
	return s.dnsChecker.Wait(ctx, ch)
}
