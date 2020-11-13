package authstore

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"time"

	"git.rootprojects.org/root/telebit/dbg"
	jwt "github.com/dgrijalva/jwt-go"
)

var ErrExists = errors.New("token already exists")
var ErrNotFound = errors.New("record not found")

type Authorization struct {
	ID string `db:"id,omitempty" json:"-"`

	MachinePPID string `db:"machine_ppid,omitempty" json:"machine_ppid,omitempty"`
	PublicKey   string `db:"public_key,omitempty" json:"public_key,omitempty"`
	SharedKey   string `db:"shared_key,omitempty" json:"shared_key"`
	Slug        string `db:"slug,omitempty" json:"slug"`

	CreatedAt time.Time `db:"created_at,omitempty" json:"created_at,omitempty"`
	UpdatedAt time.Time `db:"updated_at,omitempty" json:"updated_at,omitempty"`
	DeletedAt time.Time `db:"deleted_at,omitempty" json:"-"`
}

type Store interface {
	SetMaster(secret string) error
	Add(auth *Authorization) error
	Set(auth *Authorization) error
	Active() ([]Authorization, error)
	Inactive() ([]Authorization, error)
	Touch(id string) error
	Get(id string) (*Authorization, error)
	GetBySlug(id string) (*Authorization, error)
	GetByPub(id string) (*Authorization, error)
	Delete(auth *Authorization) error
	Close() error
}

func ToPublicKeyString(secret string) string {
	pubBytes := sha256.Sum256([]byte(secret))
	pub := base64.RawURLEncoding.EncodeToString(pubBytes[:])
	if len(pub) > 24 {
		pub = pub[:24]
	}
	return pub
}

func HMACToken(secret string, leeway time.Duration, maybeExp ...int) (token string, err error) {
	keyID := ToPublicKeyString(secret)
	if dbg.Debug {
		fmt.Fprintf(os.Stderr, "[debug] keyID=%s\n", keyID)
	}

	var exp int64
	if 0 == len(maybeExp) || 0 == maybeExp[0] {
		exp = time.Now().Add(5 * time.Minute).Unix()
	} else {
		exp = time.Now().Add(time.Duration(maybeExp[0]) * time.Second).Unix()
	}

	b := make([]byte, 16)
	_, _ = rand.Read(b)
	claims := &jwt.StandardClaims{
		Id:        base64.RawURLEncoding.EncodeToString(b),
		Subject:   "", // TODO
		Issuer:    "", // TODO
		IssuedAt:  time.Now().Add(-leeway).Unix(),
		ExpiresAt: exp + int64(leeway.Seconds()),
	}

	jwtToken := &jwt.Token{
		Header: map[string]interface{}{
			"kid": keyID,
			"typ": "JWT",
			"alg": jwt.SigningMethodHS256.Alg(),
		},
		Claims: claims,
		Method: jwt.SigningMethodHS256,
	}

	if token, err = jwtToken.SignedString([]byte(secret)); err != nil {
		return "", err
	}
	return token, nil
}
