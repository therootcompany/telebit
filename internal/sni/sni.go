package sni

// TODO this was probably copied from somewhere that deserves attribution

import (
	"errors"
)

// ErrNotClientHello happens when the TLS packet is not a ClientHello
var ErrNotClientHello = errors.New("Not a ClientHello")

// ErrMalformedHello is a failure to parse the ClientHello
var ErrMalformedHello = errors.New("malformed TLS ClientHello")

// ErrNoExtensions means that SNI is missing from the ClientHello
var ErrNoExtensions = errors.New("no TLS extensions")

// GetHostname uses SNI to determine the intended target of a new TLS connection.
func GetHostname(b []byte) (hostname string, err error) {
	// Since this is a hot piece of code (runs frequently)
	// we protect against out-of-bounds reads with recover
	// rather than adding additional out-of-bounds checks
	// in addition to the ones that Go already provides
	defer func() {
		if r := recover(); nil != r {
			err = ErrMalformedHello
		}
	}()
	rest := b[5:]
	n := len(rest)
	current := 0
	handshakeType := rest[0]
	current++
	if handshakeType != 0x1 {
		return "", ErrNotClientHello
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

	// TODO shouldn't this be current >= n ??
	if current > n {
		return "", ErrNoExtensions
	}

	current += 2

	for current < n {
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
