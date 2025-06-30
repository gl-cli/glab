package oauth2

import "gitlab.com/gitlab-org/cli/internal/config"

type stubConfig struct {
	hosts map[string]map[string]string
}

func (s stubConfig) Get(host string, key string) (string, error) {
	return s.hosts[host][key], nil
}

func (s stubConfig) GetWithSource(string, string, bool) (string, string, error) { return "", "", nil }
func (s stubConfig) Set(host string, key string, value string) error {
	if _, ok := s.hosts[host]; !ok {
		s.hosts[host] = make(map[string]string)
	}
	s.hosts[host][key] = value

	return nil
}

func (s stubConfig) UnsetHost(string)                      {}
func (s stubConfig) Hosts() ([]string, error)              { return nil, nil }
func (s stubConfig) Aliases() (*config.AliasConfig, error) { return nil, nil }
func (s stubConfig) Local() (*config.LocalConfig, error)   { return nil, nil }
func (s stubConfig) Write() error                          { return nil }
func (s stubConfig) WriteAll() error                       { return nil }
