package glinstance

import (
	"errors"
	"fmt"
	"strings"
)

const (
	DefaultHostname = "gitlab.com"
	DefaultProtocol = "https"
	DefaultClientID = "41d48f9422ebd655dd9cf2947d6979681dfaddc6d0c56f7628f6ada59559af1e"
)

// IsSelfHosted reports whether a non-normalized host name looks like a GitLab Self-Managed instance
// staging.gitlab.com is considered self-managed
func IsSelfHosted(h string) bool {
	return NormalizeHostname(h) != DefaultHostname
}

// NormalizeHostname returns the canonical host name of a GitLab instance
// Note: GitLab does not allow subdomains on gitlab.com https://gitlab.com/gitlab-org/gitlab/-/issues/26703
func NormalizeHostname(h string) string {
	return strings.ToLower(h)
}

// StripHostProtocol strips the url protocol and returns the hostname and the protocol
func StripHostProtocol(h string) (string, string) {
	hostname := NormalizeHostname(h)
	var protocol string
	if strings.HasPrefix(hostname, "http://") {
		protocol = "http"
	} else {
		protocol = "https"
	}
	hostname = strings.TrimPrefix(hostname, protocol)
	hostname = strings.Trim(hostname, ":/")
	return hostname, protocol
}

// APIEndpoint returns the REST API endpoint prefix for a GitLab instance :)
func APIEndpoint(hostname, protocol string, apiHost string) string {
	if apiHost != "" {
		hostname = apiHost
	}

	if IsSelfHosted(hostname) {
		return fmt.Sprintf("%s://%s/api/v4/", protocol, hostname)
	}
	return "https://gitlab.com/api/v4/"
}

// GraphQLEndpoint returns the GraphQL API endpoint prefix for a GitLab instance :)
func GraphQLEndpoint(hostname, protocol string) string {
	if protocol == "" {
		protocol = "https"
	}
	if IsSelfHosted(hostname) {
		return fmt.Sprintf("%s://%s/api/graphql/", protocol, hostname)
	}
	return "https://gitlab.com/api/graphql/"
}

func HostnameValidator(v any) error {
	hostname, valid := v.(string)
	if !valid {
		return errors.New("hostname is not a string")
	}

	if len(strings.TrimSpace(hostname)) < 1 {
		return errors.New("a value is required")
	}
	if strings.ContainsRune(hostname, '/') || strings.ContainsRune(hostname, ':') {
		return errors.New("invalid hostname")
	}
	return nil
}
