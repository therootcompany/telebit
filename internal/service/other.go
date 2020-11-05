// +build !darwin,!linux,!windows

package service

import (
	"errors"
)

// Install ensures a windows service is active
func Install() error {
	return errors.New("not supported for system services on this platform")
}
