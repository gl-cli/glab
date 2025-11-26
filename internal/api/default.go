package api

import "strings"

var DefaultListLimit int64 = 30

// MaxPerPage is the maximum number of items per page supported by the GitLab API.
// https://docs.gitlab.com/api/rest/#offset-based-pagination
const MaxPerPage = 100

// IsTokenConfigured checks if a token is configured (non-empty after trimming whitespace)
func IsTokenConfigured(token string) bool {
	return strings.TrimSpace(token) != ""
}
