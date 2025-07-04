package oauth2

import (
	"fmt"
	"time"

	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

const (
	redirectURI = "http://localhost:7171/auth/redirect"
)

var (
	scopes = []string{"openid", "profile", "read_user", "write_repository", "api"}
)

func NewOAuth2Config(hostname string, cfg config.Config) (*oauth2.Config, error) {
	clientID, err := oauthClientID(cfg, hostname)
	if err != nil {
		return nil, err
	}

	return &oauth2.Config{
		ClientID:    clientID,
		RedirectURL: redirectURI,
		Endpoint:    endpoint(glinstance.DefaultProtocol, hostname),
		Scopes:      scopes,
	}, nil
}

func endpoint(protocol, hostname string) oauth2.Endpoint {
	if !glinstance.IsSelfHosted(hostname) {
		return endpoints.GitLab
	}

	baseURL := fmt.Sprintf("%s://%s", glinstance.DefaultProtocol, hostname)
	return oauth2.Endpoint{
		AuthURL:       baseURL + "/oauth/authorize",
		TokenURL:      baseURL + "/oauth/token",
		DeviceAuthURL: baseURL + "/oauth/authorize_device",
	}
}

func oauthClientID(cfg config.Config, hostname string) (string, error) {
	if glinstance.IsSelfHosted(hostname) {
		clientID, err := cfg.Get(hostname, "client_id")
		if err != nil {
			return "", err
		}

		if clientID == "" {
			return "", fmt.Errorf("set 'client_id' first with `glab config set client_id <client_id> -g -h %s`", hostname)
		}
		return clientID, nil
	}
	return glinstance.DefaultClientID, nil
}

func unmarshal(hostname string, cfg config.Config) (*oauth2.Token, error) {
	result := &oauth2.Token{}
	var err error

	expiryDateString, err := cfg.Get(hostname, "oauth2_expiry_date")
	if err != nil {
		return nil, err
	}

	result.Expiry, err = time.Parse(time.RFC822, expiryDateString)
	if err != nil {
		return nil, err
	}

	result.RefreshToken, err = cfg.Get(hostname, "oauth2_refresh_token")
	if err != nil {
		return nil, err
	}

	result.AccessToken, err = cfg.Get(hostname, "token")
	if err != nil {
		return nil, err
	}

	return result, nil
}

func marshal(hostname string, cfg config.Config, token *oauth2.Token) error {
	err := cfg.Set(hostname, "is_oauth2", "true")
	if err != nil {
		return err
	}

	err = cfg.Set(hostname, "oauth2_refresh_token", token.RefreshToken)
	if err != nil {
		return err
	}

	err = cfg.Set(hostname, "oauth2_expiry_date", token.Expiry.Format(time.RFC822))
	if err != nil {
		return err
	}

	err = cfg.Set(hostname, "token", token.AccessToken)
	if err != nil {
		return err
	}

	return nil
}
