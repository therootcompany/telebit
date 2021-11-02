//go:build dev

package admin

import "net/http"

var AdminFS http.FileSystem = http.Dir("assets")
