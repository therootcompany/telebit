package authutil

import "github.com/dgrijalva/jwt-go"

// MWKey is a type guard for context.Value
type MWKey string

// Claims includes a Slug, for backwards compatibility
type Claims struct {
	Slug string `json:"slug"`
	jwt.StandardClaims
}
