package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"time"

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

	authenticated := false
	for {
		if conn, _, err := dialer.Dial(serverURL.String(), nil); err == nil {
			authenticated = true
			handler.HandleConn(ctx, conn)
		} else if !authenticated {
			return fmt.Errorf("First connection to server failed - check auth: %v", err)
		}
		loginfo.Println("disconnected from remote server")

		// Sleep for a few seconds before trying again, but only if the context is still active
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(5 * time.Second):
		}
		loginfo.Println("attempting reconnect to remote server")
	}
}
