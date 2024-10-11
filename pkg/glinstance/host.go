package glinstance

import (
	"errors"
	"fmt"
	"strings"
)

const (
	defaultHostname = "gitlab.com"
	defaultProtocol = "https"
	defaultClientId = "41d48f9422ebd655dd9cf2947d6979681dfaddc6d0c56f7628f6ada59559af1e"
)

var (
	hostnameOverride string
	protocolOverride string
)

// Default returns the host name of the default GitLab instance
func Default() string {
	return defaultHostname
}

// DefaultProtocol returns the protocol of the default GitLab instance
func DefaultProtocol() string {
	return defaultProtocol
}

func DefaultClientID() string {
	return defaultClientId
}

// OverridableDefault is like Default, except it is overridable by the GITLAB_HOST environment variable
func OverridableDefault() string {
	if hostnameOverride != "" {
		return hostnameOverride
	}
	return Default()
}

// OverridableDefaultProtocol is like DefaultProtocol, except it is overridable by the protocol found in
// the value of the GITLAB_HOST environment variable if a fully qualified URL is given as value
func OverridableDefaultProtocol() string {
	if protocolOverride != "" {
		return protocolOverride
	}
	return DefaultProtocol()
}

// OverrideDefault overrides the value returned from OverridableDefault. This should only ever be
// called from the main runtime path, not tests.
func OverrideDefault(newhost string) {
	hostnameOverride = newhost
}

// OverrideDefaultProtocol overrides the value returned from OverridableDefaultProtocol. This should only ever be
// called from the main runtime path, not tests.
func OverrideDefaultProtocol(newProtocol string) {
	protocolOverride = newProtocol
}

// IsSelfHosted reports whether a non-normalized host name looks like a Self-hosted GitLab instance
// staging.gitlab.com is considered self-hosted
func IsSelfHosted(h string) bool {
	return NormalizeHostname(h) != Default()
}

// NormalizeHostname returns the canonical host name of a GitLab instance
// Note: GitLab does not allow subdomains on gitlab.com https://gitlab.com/gitlab-org/gitlab/-/issues/26703
func NormalizeHostname(h string) string {
	return strings.ToLower(h)
}

// StripHostProtocol strips the url protocol and returns the hostname and the protocol
func StripHostProtocol(h string) (hostname, protocol string) {
	hostname = NormalizeHostname(h)
	if strings.HasPrefix(hostname, "http://") {
		protocol = "http"
	} else {
		protocol = "https"
	}
	hostname = strings.TrimPrefix(hostname, protocol)
	hostname = strings.Trim(hostname, ":/")
	return
}

// APIEndpoint returns the REST API endpoint prefix for a GitLab instance :)
func APIEndpoint(hostname, protocol string) string {
	if protocol == "" {
		protocol = OverridableDefaultProtocol()
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

func HostnameValidator(v interface{}) error {
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
