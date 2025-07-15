package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	dockerconfig "github.com/docker/cli/cli/config"

	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
)

var helperShScript = []byte(`#!/bin/sh -eu
glab auth docker-helper "$@"
`)

func configureDocker(iostreams *iostreams.IOStreams, cfg config.Config) error {
	glabPath, err := exec.LookPath("glab")
	if err != nil {
		return fmt.Errorf("looking up parent directory of glab binary: %w", err)
	}

	glabParentDir := filepath.Dir(glabPath)
	wrapperPath := filepath.Join(glabParentDir, helperFullName)
	err = os.WriteFile(wrapperPath, helperShScript, 0o700)
	if err != nil {
		return fmt.Errorf("writing helper script: %w", err)
	}

	dockerConfig, err := dockerconfig.Load("")
	if err != nil {
		return fmt.Errorf("reading current docker config: %w", err)
	}

	// WARNING: This must be added to avoid accessing an uninitialized
	// map. This happens when someone hasn't used a cred helper already
	// and isn't handled by the Docker configuration module.
	// See https://gitlab.com/gitlab-org/cli/-/issues/7921
	if dockerConfig.CredentialHelpers == nil {
		dockerConfig.CredentialHelpers = make(map[string]string)
	}

	hostnames, err := cfg.Hosts()
	if err != nil {
		return fmt.Errorf("fetching list of hosts handled by glab: %w", err)
	}

	var configuredDomains []string
	for _, hostname := range hostnames {
		domains, _, _ := cfg.GetWithSource(hostname, "container_registry_domains", false)
		for _, domain := range parseDomains(domains) {
			configuredDomains = append(configuredDomains, domain)
			dockerConfig.CredentialHelpers[domain] = helperShortName
		}
	}

	for _, domain := range configuredDomains {
		fmt.Fprintf(iostreams.StdOut, "%s Configured Docker credential helper for %s\n", iostreams.Color().GreenCheck(), domain)
	}

	err = dockerConfig.Save()
	if err != nil {
		return fmt.Errorf("registering glab docker credential helper: %w", err)
	}

	if len(configuredDomains) == 0 {
		return fmt.Errorf(
			"no hosts were configured - " +
				"ensure you've logged in via oauth2 and configured " +
				"at least one container registry domain for a host")
	}

	return nil
}
