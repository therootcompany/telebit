package server

import (
	"io"
	"log"
	"os"
)

var (
	//Logoutput -- passing the output writer from main
	loginfo      *log.Logger
	logdebug     *log.Logger
	logFlags     = log.Ldate | log.Lmicroseconds | log.Lshortfile
	connectionID int64
)

func init() {
	loginfo = log.New(os.Stdout, "INFO: server: ", logFlags)
	logdebug = log.New(os.Stdout, "DEBUG: server:", logFlags)
	connectionID = 0
}

//InitLogging -- after main sets up output, it will init all packages InitLogging
//I am sure I am doing this wrong, but I could not find a way to have package level
//logging with the flags I wanted and the ability to run lumberjack file management
func InitLogging(logoutput io.Writer) {
	loginfo.SetOutput(logoutput)

}
