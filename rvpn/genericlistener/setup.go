package genericlistener

import (
	"log"
	"os"
	"runtime"
)

var (
	loginfo      *log.Logger
	logdebug     *log.Logger
	logFlags     = log.Ldate | log.Lmicroseconds | log.Lshortfile
	connectionID int64
)

func init() {
	loginfo = log.New(os.Stdout, "INFO: genericlistener: ", logFlags)
	logdebug = log.New(os.Stdout, "DEBUG: genericlistener:", logFlags)
	pc, _, _, _ := runtime.Caller(0)
	loginfo.Println(runtime.FuncForPC(pc).Name())

	connectionID = 0
}
