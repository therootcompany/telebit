package http01fs

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/mholt/acmez/acme"
)

const (
	challengeDir = ".well-known/acme-challenge"
	tmpBase      = "acme-tmp"
)

var Provider Solver

// Solver implements the challenge.Provider interface.
type Solver struct {
	Path string
}

// Present creates a HTTP-01 Challenge Token
func (s *Solver) Present(ctx context.Context, ch acme.Challenge) error {
	log.Println("Present HTTP-01 challenge solution for", ch.Identifier.Value)

	myBase := s.Path
	if 0 == len(tmpBase) {
		myBase = "acme-tmp"
	}

	challengeBase := filepath.Join(myBase, ch.Identifier.Value, ".well-known/acme-challenge")
	_ = os.MkdirAll(challengeBase, 0700)
	tokenPath := filepath.Join(challengeBase, ch.Token)
	return ioutil.WriteFile(tokenPath, []byte(ch.KeyAuthorization), 0600)
}

// CleanUp deletes an HTTP-01 Challenge Token
func (s *Solver) CleanUp(ctx context.Context, ch acme.Challenge) error {
	log.Println("CleanUp HTTP-01 challenge solution for", ch.Identifier.Value)

	myBase := s.Path
	if 0 == len(tmpBase) {
		myBase = "acme-tmp"
	}

	// always try to remove, as there's no harm
	tokenPath := filepath.Join(myBase, ch.Identifier.Value, challengeDir, ch.Token)
	_ = os.Remove(tokenPath)

	return nil
}
