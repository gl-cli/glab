package config

import (
	"io/fs"
	"runtime"
)

func HasSecurePerms(m fs.FileMode) bool {
	if runtime.GOOS == "windows" {
		return true
	} else {
		return m == 0o600
	}
}
