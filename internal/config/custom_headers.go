package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// CustomHeader represents a single custom HTTP header configuration
type CustomHeader struct {
	Name         string `yaml:"name"`
	Value        string `yaml:"value,omitempty"`
	ValueFromEnv string `yaml:"valueFromEnv,omitempty"`
}

// ResolvedValue returns the actual header value, resolving environment variables if needed
func (h *CustomHeader) ResolvedValue() (string, error) {
	switch {
	case h.ValueFromEnv != "":
		value := os.Getenv(h.ValueFromEnv)
		if value == "" {
			return "", fmt.Errorf("environment variable %q for header %q is not set or empty", h.ValueFromEnv, h.Name)
		}
		return value, nil
	case h.Value != "":
		return h.Value, nil
	default:
		return "", fmt.Errorf("either ValueFromEnv or Value must be specified for a custom header")
	}
}

// GetCustomHeaders returns the custom headers for a specific host
func (hc *HostConfig) GetCustomHeaders() ([]CustomHeader, error) {
	entry, err := hc.FindEntry("custom_headers")
	if err != nil {
		if isNotFoundError(err) {
			// No custom headers configured, return nil slice
			return nil, nil
		}

		return nil, err
	}

	if entry.ValueNode.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("custom_headers must be a list")
	}

	var headers []CustomHeader
	for _, headerNode := range entry.ValueNode.Content {
		if headerNode.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("each custom header must be a mapping with 'name' and 'value' or 'valueFromEnv'")
		}

		var header CustomHeader
		if err := headerNode.Decode(&header); err != nil {
			return nil, fmt.Errorf("failed to decode custom header: %w", err)
		}

		// Validate header configuration
		if header.Name == "" {
			return nil, fmt.Errorf("custom header must have a 'name' field")
		}
		if header.Value == "" && header.ValueFromEnv == "" {
			return nil, fmt.Errorf("custom header %q must have either 'value' or 'valueFromEnv'", header.Name)
		}
		if header.Value != "" && header.ValueFromEnv != "" {
			return nil, fmt.Errorf("custom header %q cannot have both 'value' and 'valueFromEnv'", header.Name)
		}

		headers = append(headers, header)
	}

	return headers, nil
}

// ResolveCustomHeaders returns a map of resolved custom headers for a host
func (c *fileConfig) ResolveCustomHeaders(hostname string) (map[string]string, error) {
	if hostname == "" {
		return nil, nil
	}

	hostCfg, err := c.configForHost(hostname)
	if err != nil {
		if isNotFoundError(err) {
			// Host not configured, return empty map
			return nil, nil
		}

		return nil, err
	}

	headers, err := hostCfg.GetCustomHeaders()
	if err != nil {
		return nil, err
	}

	resolved := make(map[string]string, len(headers))
	for _, header := range headers {
		value, err := header.ResolvedValue()
		if err != nil {
			return nil, fmt.Errorf("failed to resolve header %q: %w", header.Name, err)
		}
		resolved[header.Name] = value
	}

	return resolved, nil
}

// ResolveCustomHeaders is a helper function that works with the Config interface
func ResolveCustomHeaders(cfg Config, hostname string) (map[string]string, error) {
	// Try to get the fileConfig implementation
	fc, ok := cfg.(*fileConfig)
	if !ok {
		// Not a fileConfig, this is an unexpected condition
		return nil, fmt.Errorf("unexpected config type: %T, expected *fileConfig", cfg)
	}

	return fc.ResolveCustomHeaders(hostname)
}
