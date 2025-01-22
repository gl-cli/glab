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
