package packer

import "log"
import "os"

func init() {
	logFlags := log.Ldate | log.Lmicroseconds | log.Lshortfile
	loginfo := log.New(os.Stdout, "INFO: packer: ", logFlags)
	logdebug := log.New(os.Stdout, "DEBUG: packer:", logFlags)

	loginfo.Println("")
	logdebug.Println("")
}
