package docker

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/docker/docker-credential-helpers/credentials"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/oauth2"
)

var _ credentials.Helper = (*Helper)(nil)

// Helper implements a docker-credential-* helper interface.
// It assists in logging into a registry.
type Helper struct {
	client *http.Client
	cfg    config.Config
}

// Get fetches the glab auth token for the given registryURL.
func (h *Helper) Get(registryURL string) (string, string, error) {
	hostname, err := h.findAssociatedHostname(registryURL)
	if err != nil {
		return "", "", err
	}

	ts, err := oauth2.NewConfigTokenSource(h.cfg, h.client, glinstance.DefaultProtocol, hostname)
	if err != nil {
		return "", "", fmt.Errorf("failed to get OAuth2 token source to potentially refresh token: %w", err)
	}
	if _, err := ts.Token(); err != nil {
		return "", "", fmt.Errorf("refreshing oauth2 token: %w", err)
	}

	// Config.Get will automatically search the env which will return
	// the $USER variable in *nix like systems which we do not want.
	user, _, err := h.cfg.GetWithSource(hostname, "user", false)
	if err != nil {
		return "", "", err
	}

	// Config.Get will automatically search the env which will return
	// the $GITLAB_TOKEN variable which we do not want. We want to
	// get the token from the config file!
	token, _, err := h.cfg.GetWithSource(hostname, "token", false)
	if err != nil {
		return "", "", err
	}

	if user == "" {
		return "", "", fmt.Errorf("glab user for this registryURL (hostname) is empty")
	}

	if token == "" {
		return "", "", fmt.Errorf("glab token for this registryURL (hostname) is empty")
	}

	return user, token, nil
}

// findAssociatedHostname takes a GitLab container registry URL
// and finds its associated GitLab instance's hostname.
func (h *Helper) findAssociatedHostname(registryURL string) (string, error) {
	hostnames, err := h.cfg.Hosts()
	if err != nil {
		return "", err
	}

	for _, hostname := range hostnames {
		containerRegistryDomains, _, _ := h.cfg.GetWithSource(hostname, "container_registry_domains", false)
		if slices.Contains(parseDomains(containerRegistryDomains), registryURL) {
			return hostname, nil
		}
	}

	return "", fmt.Errorf("no hostname associated with registryURL: %s", registryURL)
}

type helperError struct {
	action    string
	serverURL string
}

func (e helperError) Error() string {
	msg := "glab auth docker-helper does not support manual registry " + e.action + "s. "
	msg += "Remove the glab credential helper for " + e.serverURL + " "
	msg += "if you'd like to manually handle credentials for this registry."
	return msg
}

func (h *Helper) Add(creds *credentials.Credentials) error {
	return helperError{action: "login", serverURL: creds.ServerURL}
}

func (h *Helper) Delete(serverURL string) error {
	return helperError{action: "logout", serverURL: serverURL}
}

func (h *Helper) List() (map[string]string, error) {
	return nil, errors.New("glab auth docker-helper does not support listing registries")
}

func parseDomains(domains string) []string {
	if domains == "" {
		return nil
	}
	result := strings.Split(domains, ",")
	for i, domain := range result {
		result[i] = strings.TrimSpace(domain)
	}
	return result
}
