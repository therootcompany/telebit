package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"git.coolaj86.com/coolaj86/go-telebitd/mplexer/mgmt/authstore"

	"github.com/denisbrodbeck/machineid"
	jwt "github.com/dgrijalva/jwt-go"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	var secret string

	if len(os.Args) >= 2 {
		secret = os.Args[1]
	}
	if "" == secret {
		secret = os.Getenv("SECRET")
	}
	if "" == secret {
		fmt.Fprintf(os.Stderr, "Usage: signjwt <secret>")
		os.Exit(1)
		return
	}

	if len(os.Args) >= 3 {
		muid, err := machineid.ProtectedID("test-id|" + secret)
		if nil != err {
			panic(err)
		}
		muidBytes, _ := hex.DecodeString(muid)
		muid = base64.RawURLEncoding.EncodeToString(muidBytes)
		fmt.Println(
			muid,
			authstore.ToPublicKeyString(muid),
		)
		return
	}

	b := make([]byte, 16)
	_, _ = rand.Read(b)
	claims := &jwt.StandardClaims{
		Id:        base64.RawURLEncoding.EncodeToString(b),
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(5 * time.Minute).Unix(),
	}
	tok, err := getToken(secret, claims)
	if nil != err {
		fmt.Fprintf(os.Stderr, "signing error: %s", err)
		os.Exit(1)
		return
	}

	fmt.Println(tok)
}

func getToken(secret string, tokenData *jwt.StandardClaims) (token string, err error) {
	keyID := authstore.ToPublicKeyString(secret)

	fmt.Fprintf(os.Stderr, "secret: %s\n", secret)
	fmt.Fprintf(os.Stderr, "kid: %s\n", keyID)

	jwtToken := &jwt.Token{
		Header: map[string]interface{}{
			"kid": keyID,
			"typ": "JWT",
			"alg": jwt.SigningMethodHS256.Alg(),
		},
		Claims: tokenData,
		Method: jwt.SigningMethodHS256,
	}

	if token, err = jwtToken.SignedString([]byte(secret)); err != nil {
		return "", err
	}
	return token, nil
}
