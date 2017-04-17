package client

import (
	"log"
	"os"
)

const (
	logFlags = log.Ldate | log.Lmicroseconds | log.Lshortfile
)

var (
	loginfo  = log.New(os.Stdout, "INFO: client: ", logFlags)
	logdebug = log.New(os.Stdout, "DEBUG: client:", logFlags)
)
