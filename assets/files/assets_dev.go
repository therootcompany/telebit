//go:build dev

package files

import "net/http"

var Assets http.FileSystem = http.Dir("assets")
