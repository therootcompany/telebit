package logging

import (
	"io"
	"log"
)

// Logging structure used for setup of logging
var (
	logflags int
	loginfo  *log.Logger
	logfatal *log.Logger
)

// Init configure logging structures
func Init(writer io.Writer, flags int) {
	loginfo = log.New(writer, "INFO: ", flags)
	logfatal = log.New(writer, "INFO: ", flags)
}

// Get loggingers
func Get() (linfo *log.Logger, lfatal *log.Logger) {
	linfo = loginfo
	lfatal = logfatal
	return
}
