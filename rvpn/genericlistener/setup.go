package genericlistener

import (
	"log"
	"os"
)

var (
	loginfo  *log.Logger
	logdebug *log.Logger
	logFlags = log.Ldate | log.Lmicroseconds | log.Lshortfile
)

func init() {
	loginfo = log.New(os.Stdout, "INFO: external: ", logFlags)
	logdebug = log.New(os.Stdout, "DEBUG: external:", logFlags)
}
