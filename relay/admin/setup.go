package admin

import (
	"log"
	"os"
)

var (
	loginfo       *log.Logger
	logdebug      *log.Logger
	logFlags      = log.Ldate | log.Lmicroseconds | log.Lshortfile
	transactionID int64
)

func init() {
	loginfo = log.New(os.Stdout, "INFO: envelope: ", logFlags)
	logdebug = log.New(os.Stdout, "DEBUG: envelope:", logFlags)
	transactionID = 1
}
