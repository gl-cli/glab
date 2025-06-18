package dbg

import (
	"log"
	"os"
)

func Debug(output ...string) {
	if os.Getenv("DEBUG") != "" {
		log.Print(output)
	}
}

func Debugf(format string, v ...any) {
	if os.Getenv("DEBUG") != "" {
		log.Printf(format, v...)
	}
}
