package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"

	"github.com/gorilla/websocket"
)

type Config struct {
	Server   string
	Token    string
	Services map[string]int
	Insecure bool
}

func Run(ctx context.Context, config *Config) error {
	serverURL, err := url.Parse(config.Server)
	if err != nil {
		return fmt.Errorf("Invalid server URL: %v", err)
	}
	if serverURL.Scheme == "" {
		serverURL.Scheme = "wss"
	}
	serverURL.Path = ""

	query := make(url.Values)
	query.Set("access_token", config.Token)
	serverURL.RawQuery = query.Encode()

	dialer := websocket.Dialer{}
	if config.Insecure {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	handler := NewWsHandler(config.Services)

	conn, _, err := dialer.Dial(serverURL.String(), nil)
	if err != nil {
		return fmt.Errorf("First connection to server failed - check auth: %v", err)
	}

	handler.HandleConn(ctx, conn)
	return nil
}
