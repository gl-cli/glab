//go:build !windows
// +build !windows

package config

import "io/fs"

func HasSecurePerms(m fs.FileMode) bool {
	return m == 0o600
}
