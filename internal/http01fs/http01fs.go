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

// Provider is the default Challenge Solver Provider
var Provider Solver

// Solver implements the challenge.Provider interface.
type Solver struct {
	Path string
}

// Present creates a HTTP-01 Challenge Token
func (s *Solver) Present(ctx context.Context, ch acme.Challenge) error {
	log.Println("Present HTTP-01 (fs) challenge solution for", ch.Identifier.Value)

	if 0 == len(s.Path) {
		s.Path = tmpBase
	}

	challengeBase := filepath.Join(s.Path, ch.Identifier.Value, challengeDir)
	_ = os.MkdirAll(challengeBase, 0700)
	tokenPath := filepath.Join(challengeBase, ch.Token)
	return ioutil.WriteFile(tokenPath, []byte(ch.KeyAuthorization), 0600)
}

// CleanUp deletes an HTTP-01 Challenge Token
func (s *Solver) CleanUp(ctx context.Context, ch acme.Challenge) error {
	log.Println("CleanUp HTTP-01 (fs) challenge solution for", ch.Identifier.Value)

	if 0 == len(s.Path) {
		s.Path = tmpBase
	}

	// always try to remove, as there's no harm
	tokenPath := filepath.Join(s.Path, ch.Identifier.Value, challengeDir, ch.Token)
	_ = os.Remove(tokenPath)

	return nil
}
