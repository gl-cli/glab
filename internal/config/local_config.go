package config

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type LocalConfig struct {
	ConfigMap
	Parent Config
}

// for later use we might prefer relative paths
// GitDir will find a directory or just return ".git"
func GitDir(preferRelative bool) []string {
	var err error
	var out strings.Builder

	// `git rev-parse --git-dir` since git v1.2.2
	cmd := exec.Command("git", "rev-parse", "--git-dir")

	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		// should fail
		return []string{".git"}
	}

	gitDir := strings.TrimSpace(out.String())

	if !filepath.IsAbs(gitDir) {
		// gitDir is relative
		if !preferRelative {
			// but we prefer absolute
			workDir, err := os.Getwd()
			if err == nil {
				return []string{workDir, gitDir}
			}
		}
		// relative should work
		return []string{gitDir}
	} else {
		// gitDir is absolute
		if preferRelative {
			// but we prefer relative
			var relativeDir string
			workDir, err := os.Getwd()
			if err == nil {
				relativeDir, err = filepath.Rel(workDir, gitDir)
			}
			if err == nil {
				return []string{relativeDir}
			}
		}
		// absolute should work
		return []string{gitDir}
	}
}

// LocalConfigDir returns the local config path in map
// which must be joined for complete path
var LocalConfigDir = func() []string {
	return append(GitDir(true), "glab-cli")
}

// LocalConfigFile returns the config file name with full path
var LocalConfigFile = func() string {
	configFile := append(LocalConfigDir(), "config.yml")
	return filepath.Join(configFile...)
}

func (a *LocalConfig) Get(key string) (string, bool) {
	key = ConfigKeyEquivalence(key)
	if a.Empty() {
		return "", false
	}
	value, _ := a.GetStringValue(key)

	return value, value != ""
}

func (a *LocalConfig) Set(key, value string) error {
	key = ConfigKeyEquivalence(key)
	err := a.SetStringValue(key, value)
	if err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	err = a.Write()
	if err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func (a *LocalConfig) Delete(key string) error {
	a.RemoveEntry(key)

	return a.Write()
}

func (a *LocalConfig) Write() error {
	// Check if it's a Git repository
	if !CheckPathExists(filepath.Join(GitDir(true)...)) {
		return errors.New("not a Git repository")
	}

	localConfigBytes, err := yaml.Marshal(a.ConfigMap.Root)
	if err != nil {
		return err
	}
	err = WriteConfigFile(LocalConfigFile(), yamlNormalize(localConfigBytes))
	if err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func (a *LocalConfig) All() map[string]string {
	out := map[string]string{}

	if a.Empty() {
		return out
	}

	for i := 0; i < len(a.Root.Content)-1; i += 2 {
		key := a.Root.Content[i].Value
		value := a.Root.Content[i+1].Value
		out[key] = value
	}

	return out
}
