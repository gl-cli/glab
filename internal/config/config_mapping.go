package config

import (
	"strings"
)

// ConfigKeyEquivalence returns the equivalent key that's actually used in the config file
func ConfigKeyEquivalence(key string) string {
	key = strings.ToLower(key)
	// we only have a set default for one setting right now
	switch key {
	case "gitlab_api_host":
		return "api_host"
	case "gitlab_host", "gitlab_uri", "gl_host":
		return "host"
	case "gitlab_token", "oauth_token":
		return "token"
	case "no_prompt", "prompt_disabled":
		return "no_prompt"
	case "git_remote_url_var", "git_remote_alias", "remote_alias", "remote_nickname", "git_remote_nickname":
		return "remote_alias"
	case "editor", "visual", "glab_editor":
		return "editor"
	case "client_id":
		return "client_id"
	default:
		return key
	}
}

// EnvKeyEquivalence returns the equivalent key that's used for environment variables
func EnvKeyEquivalence(key string) []string {
	key = strings.ToLower(key)
	// we only have a set default for one setting right now
	switch key {
	case "api_host":
		return []string{"GITLAB_API_HOST"}
	case "host":
		return []string{"GITLAB_HOST", "GITLAB_URI", "GL_HOST"}
	case "token":
		return []string{"GITLAB_TOKEN", "GITLAB_ACCESS_TOKEN", "OAUTH_TOKEN"}
	case "no_prompt":
		return []string{"NO_PROMPT", "PROMPT_DISABLED"}
	case "editor", "visual", "glab_editor":
		return []string{"GLAB_EDITOR", "VISUAL", "EDITOR"}
	case "remote_alias":
		return []string{"GIT_REMOTE_URL_VAR", "GIT_REMOTE_ALIAS", "REMOTE_ALIAS", "REMOTE_NICKNAME", "GIT_REMOTE_NICKNAME"}
	case "client_id":
		return []string{"GITLAB_CLIENT_ID"}
	default:
		return []string{strings.ToUpper(key)}
	}
}

func defaultFor(key string) string {
	key = strings.ToLower(key)
	// we only have a set default for one setting right now
	switch key {
	case "gitlab_host", "gitlab_uri":
		return defaultHostname
	case "git_protocol":
		return defaultGitProtocol
	case "api_protocol":
		return defaultAPIProtocol
	case "glamour_style":
		return defaultGlamourStyle
	default:
		return ""
	}
}
