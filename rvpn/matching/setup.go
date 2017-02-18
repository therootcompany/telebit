package matching

import "log"
import "os"

func init() {
	logFlags := log.Ldate | log.Lmicroseconds | log.Lshortfile
	loginfo := log.New(os.Stdout, "INFO: matching: ", logFlags)
	logdebug := log.New(os.Stdout, "DEBUG: matching:", logFlags)

	loginfo.Println("")
	logdebug.Println("")
}
