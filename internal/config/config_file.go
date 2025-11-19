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

// legacyConfigDir returns the legacy config directory (~/.config/glab-cli).
// This was the default location before XDG platform-specific paths were adopted.
// Uses os.UserHomeDir() for cross-platform compatibility (works on Windows, macOS, Linux).
func legacyConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".config", "glab-cli")
}

// ConfigDir returns the config directory for writing configuration.
// It respects GLAB_CONFIG_DIR as the highest priority override.
// For backward compatibility, if a legacy config exists at ~/.config/glab-cli/,
// that location continues to be used. Otherwise, uses XDG_CONFIG_HOME.
func ConfigDir() string {
	glabDir := os.Getenv("GLAB_CONFIG_DIR")
	if glabDir != "" {
		return glabDir
	}

	// Check for legacy config location for backward compatibility
	legacyDir := legacyConfigDir()
	if legacyDir != "" {
		legacyConfigFile := filepath.Join(legacyDir, "config.yml")
		if _, err := os.Stat(legacyConfigFile); err == nil {
			return legacyDir
		}
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

// SearchConfigFile searches for an existing config file across all config paths.
// It respects GLAB_CONFIG_DIR as the highest priority override.
// Search order:
// 1. $GLAB_CONFIG_DIR/config.yml (if GLAB_CONFIG_DIR is set)
// 2. ~/.config/glab-cli/config.yml (legacy location, for backward compatibility)
// 3. $XDG_CONFIG_HOME/glab-cli/config.yml (platform-specific XDG location)
// 4. $XDG_CONFIG_DIRS/glab-cli/config.yml (system-wide configs)
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

	// Check legacy location first for backward compatibility
	legacyDir := legacyConfigDir()
	if legacyDir != "" {
		legacyConfigPath := filepath.Join(legacyDir, "config.yml")
		if _, err := os.Stat(legacyConfigPath); err == nil {
			return legacyConfigPath, nil
		}
	}

	// XDG search: user config → system configs
	configPath, err := xdg.SearchConfigFile("glab-cli/config.yml")
	if err != nil {
		return "", err
	}
	return configPath, nil
}

// checkForDuplicateConfigs warns if multiple config files exist across different locations.
// Since we don't support config merging (yet), only the first file found is used, which can
// be confusing if users have configs in multiple locations.
func checkForDuplicateConfigs() {
	// Only check if GLAB_CONFIG_DIR is not set
	if os.Getenv("GLAB_CONFIG_DIR") != "" {
		return
	}

	var existingConfigs []string
	seenPaths := make(map[string]bool)

	// Check legacy location
	legacyDir := legacyConfigDir()
	if legacyDir != "" {
		legacyConfigPath := filepath.Join(legacyDir, "config.yml")
		if _, err := os.Stat(legacyConfigPath); err == nil {
			existingConfigs = append(existingConfigs, legacyConfigPath)
			seenPaths[legacyConfigPath] = true
		}
	}

	// Check XDG user config (if not already seen)
	xdgConfigPath := filepath.Join(xdg.ConfigHome, "glab-cli", "config.yml")
	if !seenPaths[xdgConfigPath] {
		if _, err := os.Stat(xdgConfigPath); err == nil {
			existingConfigs = append(existingConfigs, xdgConfigPath)
			seenPaths[xdgConfigPath] = true
		}
	}

	// Check system-wide XDG configs (skip if already seen)
	for _, dir := range xdg.ConfigDirs {
		// Skip if it's the same as ConfigHome (already checked above)
		if dir == xdg.ConfigHome {
			continue
		}
		configPath := filepath.Join(dir, "glab-cli", "config.yml")
		if !seenPaths[configPath] {
			if _, err := os.Stat(configPath); err == nil {
				existingConfigs = append(existingConfigs, configPath)
				seenPaths[configPath] = true
			}
		}
	}

	// Warn if multiple configs exist
	if len(existingConfigs) > 1 {
		fmt.Fprintf(os.Stderr, "Warning: Multiple config files found. Only the first one will be used.\n")
		fmt.Fprintf(os.Stderr, "  Using: %s\n", existingConfigs[0])
		for _, path := range existingConfigs[1:] {
			fmt.Fprintf(os.Stderr, "  Ignoring: %s\n", path)
		}
		fmt.Fprintf(os.Stderr, "Consider consolidating to one location to avoid confusion.\n")
	}
}

// Init initialises and returns the cached configuration
func Init() (Config, error) {
	if cachedConfig != nil || configError != nil {
		return cachedConfig, configError
	}

	// Ensure the config directory exists before attempting to read/write config files.
	// This is especially important on Windows where os.ReadFile returns a different error
	// when the parent directory doesn't exist vs. when just the file doesn't exist.
	configDir := ConfigDir()
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Check for duplicate configs and warn user
	checkForDuplicateConfigs()

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
