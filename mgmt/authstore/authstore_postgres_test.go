package authstore

import "strings"

var connStr = "postgres://postgres:postgres@localhost/postgres"

func init() {
	// TODO url.Parse
	if strings.Contains(connStr, "@localhost/") || strings.Contains(connStr, "@localhost:") {
		connStr += "?sslmode=disable"
	} else {
		connStr += "?sslmode=required"
	}
}
