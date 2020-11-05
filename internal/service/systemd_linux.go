package service

import (
	"errors"
)

// Install ensures a systemd service is active
func Install() error {
	return errors.New("'install' not supported for system services on this platform")
}
