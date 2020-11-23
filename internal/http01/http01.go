package http01

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

	"github.com/mholt/acmez/acme"
)

// Config is used to configure the creation of the HTTP-01 Solver.
type Config struct {
	Endpoint   *url.URL
	Tokener    func() string
	HTTPClient *http.Client
}

// Solver implements the challenge.Provider interface.
type Solver struct {
	config *Config
}

// Challenge is an ACME http-01 challenge
type Challenge struct {
	Type             string     `json:"type"`
	Token            string     `json:"token"`
	KeyAuthorization string     `json:"key_authorization"`
	Identifier       Identifier `json:"identifier"`
}

// Identifier is restricted to DNS Domain Names for now
type Identifier struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// NewSolver return a new HTTP-01 Solver.
func NewSolver(config *Config) (*Solver, error) {
	if config == nil {
		return nil, errors.New("api: the configuration of the DNS provider is nil")
	}

	if config.Endpoint == nil {
		return nil, errors.New("api: the endpoint is missing")
	}

	if nil == config.HTTPClient {
		config.HTTPClient = &http.Client{
			Timeout: 5 * time.Second,
		}
	}

	return &Solver{config: config}, nil
}

// Present creates a HTTP-01 Challenge Token
func (s *Solver) Present(ctx context.Context, ch acme.Challenge) error {
	log.Println("Present HTTP-01 challenge solution for", ch.Identifier.Value)
	msg := &Challenge{
		Type:             "http-01",
		Token:            ch.Token,
		KeyAuthorization: ch.KeyAuthorization,
		Identifier: Identifier{
			Type:  ch.Identifier.Type,
			Value: ch.Identifier.Value,
		},
	}

	err := s.doRequest(http.MethodPost, fmt.Sprintf("/%s", msg.Identifier.Value), msg)
	if err != nil {
		return fmt.Errorf("api: %w", err)
	}
	return nil
}

// CleanUp deletes an HTTP-01 Challenge Token
func (s *Solver) CleanUp(ctx context.Context, ch acme.Challenge) error {
	log.Println("CleanUp HTTP-01 challenge solution for", ch.Identifier.Value)
	msg := &Challenge{
		Type:             "http-01",
		Token:            ch.Token,
		KeyAuthorization: ch.KeyAuthorization,
		Identifier: Identifier{
			Type:  ch.Identifier.Type,
			Value: ch.Identifier.Value,
		},
	}

	err := s.doRequest(
		http.MethodDelete,
		fmt.Sprintf("/%s/%s/%s/%s", msg.Identifier.Value, msg.Token, msg.KeyAuthorization, msg.Type),
		nil,
	)
	if err != nil {
		return fmt.Errorf("api: %w", err)
	}
	return nil
}

func (s *Solver) doRequest(method, uri string, msg interface{}) error {
	data, _ := json.MarshalIndent(msg, "", "  ")
	reqBody := bytes.NewBuffer(data)

	newURI := path.Join(s.config.Endpoint.EscapedPath(), uri)
	endpoint, err := s.config.Endpoint.Parse(newURI)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(method, endpoint.String(), reqBody)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	apiToken := s.config.Tokener()
	if len(apiToken) > 0 {
		req.Header.Set("Authorization", "Bearer "+apiToken)
	}

	//fmt.Printf("curl -X %s %s \\\n    -H 'Authorization: Bearer %s' \\\n    -d '%s'\n\n", method, endpoint.String(), apiToken, string(data))
	resp, err := s.config.HTTPClient.Do(req)
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
