package config

import "io/fs"

// FIXME: Use correct permission check
// See https://gitlab.com/gitlab-org/cli/-/issues/7588
func HasSecurePerms(_ fs.FileMode) bool {
	return true
}
