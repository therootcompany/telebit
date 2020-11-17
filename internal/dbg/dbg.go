package dbg

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

// Verbose is a flag for whether or not verbose logging should be activated
var Verbose bool

// Debug does not truncate byte strings
var Debug bool

// VerboseVerbose does not truncate strings
var VerboseVerbose bool

// OutFile is the output path for StdOut
var OutFile *os.File

// ErrFile is the output path for StdErr
var ErrFile *os.File

type out int

const (
	stdout out = iota
	stderr
)

type output struct {
	out   out
	msg   string
	other []interface{}
}

var log chan output

func init() {
	log = make(chan output)

	Init()

	go func() {
		for {
			o := <-log
			// because OutFile and ErrFile may be the same
			msg := strings.TrimSuffix(o.msg, "\n") + "\n"
			b := []byte(fmt.Sprintf(msg, o.other...))
			if stdout == o.out {
				OutFile.Write(b)
			} else {
				ErrFile.Write(b)
			}
		}
	}()
}

// Printf will print to log.Printf
func Printf(tpl string, other ...interface{}) {
	log <- output{
		out:   stdout,
		msg:   tpl,
		other: other,
	}
}

// Warnf will print to fmt.Fprintf(stderr)
func Warnf(tpl string, other ...interface{}) {
	log <- output{
		out:   stderr,
		msg:   "[warn] " + tpl,
		other: other,
	}
}

// Debugf will print to fmt.Fprintf(std)
func Debugf(tpl string, other ...interface{}) {
	log <- output{
		out:   stderr,
		msg:   "[debug] " + tpl,
		other: other,
	}
}

// Init will set debug vars from ENVs and print out whatever is set
func Init() {
	if nil == OutFile {
		OutFile = os.Stdout
	}
	if nil == ErrFile {
		ErrFile = os.Stderr
	}

	if !Verbose {
		if "true" == os.Getenv("VERBOSE") {
			Printf("VERBOSE=true")
			Verbose = true
		}
		if Verbose {
			fmt.Fprintf(os.Stderr, "VERBOSE: extra logging enabled\n")
		}
	}

	if !VerboseVerbose {
		if "true" == strings.ToLower(os.Getenv("VERBOSE_VERBOSE")) {
			Printf("VERBOSE_VERBOSE=true")
			VerboseVerbose = true
		} else if "true" == strings.ToLower(os.Getenv("VERBOSE_RAW")) {
			Printf("VERBOSE_RAW=true # Deprecated: Use VERBOSE_VERBOSE=true")
			VerboseVerbose = true
		}
		if VerboseVerbose {
			fmt.Fprintf(os.Stderr, "VERBOSE_VERBOSE: output will NOT be truncated\n")
		}
	}

	if !Debug {
		if "true" == strings.ToLower(os.Getenv("DEBUG")) {
			Printf("DEBUG=true")
			Debug = true
		} else if "true" == strings.ToLower(os.Getenv("VERBOSE_BYTES")) {
			Printf("VERBOSE_BYTES=true # Deprecated: Use DEBUG=true")
			Debug = true
		}
		if Debug {
			fmt.Fprintf(os.Stderr, "DEBUG: byte output will be printed as hex\n")
		}
	}
}

// Trunc will take up to the first and last 20 bytes of the input to product 80 char hex output
func Trunc(b []byte, n int) string {
	bin := b[:n]
	if Debug || VerboseVerbose {
		if Debug {
			return hex.EncodeToString(bin)
		}
		return string(bin)
	}
	if n > 40 {
		return hex.EncodeToString(bin[:19]) + ".." + hex.EncodeToString(bin[n-19:])
	}
	return hex.EncodeToString(bin)
}
