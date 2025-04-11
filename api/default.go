package api

var DefaultListLimit = 30

// MaxPerPage is the maximum number of items per page supported by the GitLab API.
// https://docs.gitlab.com/api/rest/#offset-based-pagination
const MaxPerPage = 100

// IsValidToken checks if a token provided is valid.
// The token string must be 26 characters in length and have the 'glpat-'
// prefix or just be 20 characters long to be recognized as a valid personal access token.
// token can be 64 characters to include oauth tokens
// TODO: check if token has minimum scopes required by glab
func IsValidToken(token string) bool {
	return (len(token) == 26 && token[:6] == "glpat-") || len(token) == 20 || len(token) == 64
}
