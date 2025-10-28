package config

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"syscall"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

var (
	cachedConfig Config
	configError  error
)

// ConfigDir returns the config directory for writing configuration.
// It respects GLAB_CONFIG_DIR as the highest priority override,
// otherwise uses XDG_CONFIG_HOME (defaulting to ~/.config).
func ConfigDir() string {
	glabDir := os.Getenv("GLAB_CONFIG_DIR")
	if glabDir != "" {
		return glabDir
	}
	return filepath.Join(xdg.ConfigHome, "glab-cli")
}

// ConfigFile returns the config file path.
// It respects GLAB_CONFIG_DIR as the highest priority override,
// otherwise returns the XDG-compliant user config file path.
// This function only determines the path without creating directories.
func ConfigFile() string {
	return filepath.Join(ConfigDir(), "config.yml")
}

// SearchConfigFile searches for an existing config file across all XDG config paths.
// It respects GLAB_CONFIG_DIR as the highest priority override.
// Search order:
// 1. $GLAB_CONFIG_DIR/config.yml (if GLAB_CONFIG_DIR is set)
// 2. $XDG_CONFIG_HOME/glab-cli/config.yml (default: ~/.config/glab-cli/config.yml)
// 3. $XDG_CONFIG_DIRS/glab-cli/config.yml (default: /etc/xdg/glab-cli/config.yml)
//
// Returns the path to the first config file found, or an error if none exist.
func SearchConfigFile() (string, error) {
	// HIGHEST PRIORITY: GLAB_CONFIG_DIR completely bypasses XDG
	if os.Getenv("GLAB_CONFIG_DIR") != "" {
		configPath := ConfigFile()
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
		// If GLAB_CONFIG_DIR is set but file doesn't exist,
		// still return this path (don't fall through to XDG)
		return configPath, os.ErrNotExist
	}

	// XDG search: user config → system configs
	configPath, err := xdg.SearchConfigFile("glab-cli/config.yml")
	if err != nil {
		return "", err
	}
	return configPath, nil
}

// Init initialises and returns the cached configuration
func Init() (Config, error) {
	if cachedConfig != nil || configError != nil {
		return cachedConfig, configError
	}
	cachedConfig, configError = ParseDefaultConfig()

	if os.IsNotExist(configError) {
		if err := cachedConfig.WriteAll(); err != nil {
			return nil, err
		}
		configError = nil
	}
	return cachedConfig, configError
}

func ParseDefaultConfig() (Config, error) {
	// Try to find existing config first (searches all XDG paths)
	configPath, err := SearchConfigFile()
	if err != nil {
		// No config found, use default writable location
		configPath = ConfigFile()
	}
	return ParseConfig(configPath)
}

var ReadConfigFile = func(filename string) ([]byte, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, pathError(err)
	}

	return data, nil
}

var WriteConfigFile = func(filename string, data []byte) error {
	err := os.MkdirAll(path.Dir(filename), 0o750)
	if err != nil {
		return pathError(err)
	}
	_, err = os.ReadFile(filename)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	err = WriteFile(filename, data, 0o600)
	return err
}

func ParseConfigFile(filename string) ([]byte, *yaml.Node, error) {
	stat, err := os.Stat(filename)
	// we want to check if there actually is a file, sometimes
	// configs are just passed via stubs
	if err == nil {
		if !HasSecurePerms(stat.Mode().Perm()) {
			return nil, nil,
				fmt.Errorf("%s has the permissions %o, but glab requires 600.\nConsider running `chmod 600 %s`",
					filename,
					stat.Mode(),
					filename,
				)
		}
	}

	data, err := ReadConfigFile(filename)
	if err != nil {
		return nil, nil, err
	}

	root, err := parseConfigData(data)
	if err != nil {
		return nil, nil, err
	}
	return data, root, err
}

func parseConfigData(data []byte) (*yaml.Node, error) {
	var root yaml.Node
	err := yaml.Unmarshal(data, &root)
	if err != nil {
		return nil, err
	}

	if len(root.Content) == 0 {
		return &yaml.Node{
			Kind:    yaml.DocumentNode,
			Content: []*yaml.Node{{Kind: yaml.MappingNode}},
		}, nil
	}
	if root.Content[0].Kind != yaml.MappingNode {
		return &root, fmt.Errorf("expected a top level map")
	}
	return &root, nil
}

func ParseConfig(filename string) (Config, error) {
	_, root, err := ParseConfigFile(filename)
	var confError error
	if err != nil {
		if os.IsNotExist(err) {
			root = NewBlankRoot()
			confError = os.ErrNotExist
		} else {
			return nil, err
		}
	}

	// Load local config file
	if _, localRoot, err := ParseConfigFile(LocalConfigFile()); err == nil {
		if len(localRoot.Content[0].Content) > 0 {
			newContent := []*yaml.Node{
				{Value: "local"},
				localRoot.Content[0],
			}
			restContent := root.Content[0].Content
			root.Content[0].Content = append(newContent, restContent...)
		}
	}

	// Load aliases config file
	if _, aliasesRoot, err := ParseConfigFile(aliasesConfigFile()); err == nil {
		if len(aliasesRoot.Content[0].Content) > 0 {
			newContent := []*yaml.Node{
				{Value: "aliases"},
				aliasesRoot.Content[0],
			}
			restContent := root.Content[0].Content
			root.Content[0].Content = append(newContent, restContent...)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	return NewConfig(root), confError
}

func pathError(err error) error {
	var pathError *os.PathError
	if errors.As(err, &pathError) && errors.Is(pathError.Err, syscall.ENOTDIR) {
		if p := findRegularFile(pathError.Path); p != "" {
			return fmt.Errorf("remove or rename regular file `%s` (must be a directory)", p)
		}
	}
	return err
}

func findRegularFile(p string) string {
	for {
		if s, err := os.Stat(p); err == nil && s.Mode().IsRegular() {
			return p
		}
		newPath := path.Dir(p)
		if newPath == p || newPath == "/" || newPath == "." {
			break
		}
		p = newPath
	}
	return ""
}
