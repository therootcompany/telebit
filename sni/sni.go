package sni

// TODO this was probably copied from somewhere that deserves attribution

import (
	"errors"
)

// GetHostname uses SNI to determine the intended target of a new TLS connection.
func GetHostname(b []byte) (string, error) {
	rest := b[5:]
	current := 0
	handshakeType := rest[0]
	current++
	if handshakeType != 0x1 {
		return "", errors.New("Not a ClientHello")
	}

	// Skip over another length
	current += 3
	// Skip over protocolversion
	current += 2
	// Skip over random number
	current += 4 + 28
	// Skip over session ID
	sessionIDLength := int(rest[current])
	current++
	current += sessionIDLength

	cipherSuiteLength := (int(rest[current]) << 8) + int(rest[current+1])
	current += 2
	current += cipherSuiteLength

	compressionMethodLength := int(rest[current])
	current++
	current += compressionMethodLength

	if current > len(rest) {
		return "", errors.New("no extensions")
	}

	current += 2

	hostname := ""
	for current < len(rest) && hostname == "" {
		extensionType := (int(rest[current]) << 8) + int(rest[current+1])
		current += 2

		extensionDataLength := (int(rest[current]) << 8) + int(rest[current+1])
		current += 2

		if extensionType == 0 {

			// Skip over number of names as we're assuming there's just one
			current += 2

			nameType := rest[current]
			current++
			if nameType != 0 {
				return "", errors.New("Not a hostname")
			}
			nameLen := (int(rest[current]) << 8) + int(rest[current+1])
			current += 2
			hostname = string(rest[current : current+nameLen])
		}

		current += extensionDataLength
	}
	if hostname == "" {
		return "", errors.New("No hostname")
	}
	return hostname, nil

}
