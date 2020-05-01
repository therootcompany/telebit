package log

import (
	"io"
	"log"
	"os"
)

var (
	//Logoutput -- passing the output writer from main
	Loginfo  *log.Logger
	Logdebug *log.Logger
	LogFlags = log.Ldate | log.Lmicroseconds | log.Lshortfile
)

func init() {
	Loginfo = log.New(os.Stdout, "INFO: ", LogFlags)
	Logdebug = log.New(os.Stdout, "DEBUG: ", LogFlags)
}

//InitLogging -- after main sets up output, it will init all packages InitLogging
//I am sure I am doing this wrong, but I could not find a way to have package level
//logging with the flags I wanted and the ability to run lumberjack file management
func InitLogging(logoutput io.Writer) {
	Loginfo.SetOutput(logoutput)
}
