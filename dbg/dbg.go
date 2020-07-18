package dbg

import (
	"encoding/hex"
	"fmt"
	"os"
)

// Debug is a flag for whether or not verbose logging should be activated
var Debug bool

var rawBytes bool
var allBytes bool

func init() {
	Init()
}

// Init will set debug vars from ENVs and print out whatever is set
func Init() {
	if !Debug {
		Debug = ("true" == os.Getenv("VERBOSE"))
	}
	if !allBytes {
		allBytes = ("true" == os.Getenv("VERBOSE_BYTES"))
	}
	if !rawBytes {
		rawBytes = ("true" == os.Getenv("VERBOSE_RAW"))
	}

	if Debug {
		fmt.Fprintf(os.Stderr, "DEBUG=true\n")
	}
	if allBytes || rawBytes {
		fmt.Fprintf(os.Stderr, "VERBOSE_BYTES=true\n")
	}
	if rawBytes {
		fmt.Fprintf(os.Stderr, "VERBOSE_RAW=true\n")
	}
}

// Trunc will take up to the first and last 20 bytes of the input to product 80 char hex output
func Trunc(b []byte, n int) string {
	bin := b[:n]
	if allBytes || rawBytes {
		if rawBytes {
			return string(bin)
		}
		return hex.EncodeToString(bin)
	}
	if n > 40 {
		return hex.EncodeToString(bin[:19]) + ".." + hex.EncodeToString(bin[n-19:])
	}
	return hex.EncodeToString(bin)
}
