package packer

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
	loginfo = log.New(os.Stdout, "INFO: packer: ", logFlags)
	logdebug = log.New(os.Stdout, "DEBUG: packer:", logFlags)
}
