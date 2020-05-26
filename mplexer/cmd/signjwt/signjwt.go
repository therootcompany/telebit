package main

import (
	"fmt"
	"os"

	jwt "github.com/dgrijalva/jwt-go"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	var secret string

	if len(os.Args) == 2 {
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

	tok, err := getToken(secret, []string{})
	if nil != err {
		fmt.Fprintf(os.Stderr, "signing error: %s", err)
		os.Exit(1)
		return
	}

	fmt.Println(tok)
}

func getToken(secret string, domains []string) (token string, err error) {
	tokenData := jwt.MapClaims{"domains": domains}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenData)
	if token, err = jwtToken.SignedString([]byte(secret)); err != nil {
		return "", err
	}
	return token, nil
}
