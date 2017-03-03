package connectiontrack

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
	loginfo = log.New(os.Stdout, "INFO: connectiontrack: ", logFlags)
	logdebug = log.New(os.Stdout, "DEBUG: connectiontrack:", logFlags)
}
