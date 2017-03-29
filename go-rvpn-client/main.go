package main

import (
	"git.daplie.com/Daplie/go-rvpn-server/rvpn/client"
	jwt "github.com/dgrijalva/jwt-go"
)

func main() {
	tokenData := jwt.MapClaims{
		"domains": []string{
			"localhost.foo.daplie.me",
			"localhost.bar.daplie.me",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenData)
	tokenStr, err := token.SignedString([]byte("abc123"))
	if err != nil {
		panic(err)
	}

	config := client.Config{
		Server:   "wss://localhost.daplie.me:9999",
		Services: map[string]int{"https": 8443},
		Token:    tokenStr,
		Insecure: true,
	}
	panic(client.Run(&config))
}
