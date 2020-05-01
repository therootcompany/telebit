package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

// The Config struct holds all of the information needed to establish and handle a connection
// with the RVPN server.
type Config struct {
	Server   string
	Token    string
	Insecure bool
	Services map[string]map[string]int
}

// Run establishes a connection with the RVPN server specified in the config. If the first attempt
// to connect fails it is assumed that something is wrong with the authentication and it will
// return an error. Otherwise it will continuously attempt to reconnect whenever the connection
// is broken.
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

	for name, portList := range config.Services {
		if _, ok := portList["*"]; !ok {
			return fmt.Errorf(`service %s missing port for "*"`, name)
		}
	}
	handler := NewWsHandler(config.Services)

	authenticated := false
	for {
		fmt.Printf("debug serverURL:\n%+v", serverURL)
		if conn, _, err := dialer.Dial(serverURL.String(), nil); err == nil {
			loginfo.Println("connected to remote server")
			authenticated = true
			handler.HandleConn(ctx, conn)
		} else if !authenticated {
			return fmt.Errorf("first connection to server failed - check auth: %s", err.Error())
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
