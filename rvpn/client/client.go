package client

import (
	"crypto/tls"
	"net/url"

	"fmt"

	"git.daplie.com/Daplie/go-rvpn-server/rvpn/packer"
	"github.com/gorilla/websocket"
)

type Config struct {
	Server   string
	Token    string
	Services map[string]int
	Insecure bool
}

func Run(config *Config) error {
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

	conn, _, err := dialer.Dial(serverURL.String(), nil)
	if err != nil {
		return fmt.Errorf("First connection to server failed - check auth: %v", err)
	}

	localConns := newLocalConns(conn, config.Services)
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("websocket read errored: %v", err)
		}

		p, err := packer.ReadMessage(message)
		if err != nil {
			return fmt.Errorf("packer read failed: %v", err)
		}

		err = localConns.Write(p)
		if err != nil {
			return fmt.Errorf("failed to write data: %v", err)
		}
	}
}
