package oauth2

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

type configTokenSource struct {
	cfg        config.Config
	httpClient *http.Client
	hostname   string

	oauth2Config oauth2.Config

	// Token is not thread-safe
	mu sync.Mutex
}

func NewConfigTokenSource(cfg config.Config, httpClient *http.Client, protocol, hostname string) (oauth2.TokenSource, error) {
	clientID, err := oAuthClientID(cfg, hostname)
	if err != nil {
		return nil, err
	}

	at, err := tokenFromConfig(hostname, cfg)
	if err != nil {
		return nil, err
	}

	oauth2Config := oauth2.Config{
		ClientID:    clientID,
		RedirectURL: redirectURI,
		Endpoint:    endpoint(protocol, hostname),
		Scopes:      strings.Split(scopes, "+"),
	}

	src := &configTokenSource{
		cfg:          cfg,
		oauth2Config: oauth2Config,
		httpClient:   httpClient,
		hostname:     hostname,
	}

	cfgToken := toOAuth2Token(at)
	return oauth2.ReuseTokenSource(cfgToken, src), nil
}

func endpoint(protocol, hostname string) oauth2.Endpoint {
	if !glinstance.IsSelfHosted(hostname) {
		return endpoints.GitLab
	}

	baseURL := fmt.Sprintf("%s://%s", protocol, hostname)
	return oauth2.Endpoint{
		AuthURL:       baseURL + "/oauth/authorize",
		TokenURL:      baseURL + "/oauth/token",
		DeviceAuthURL: baseURL + "/oauth/authorize_device",
	}
}

func (c *configTokenSource) Token() (*oauth2.Token, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	at, err := tokenFromConfig(c.hostname, c.cfg)
	if err != nil {
		return nil, err
	}
	cfgToken := toOAuth2Token(at)

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, c.httpClient)
	refreshedToken, err := c.oauth2Config.TokenSource(ctx, cfgToken).Token()
	if err != nil {
		return nil, err
	}

	at = fromOAuth2Token(refreshedToken)

	err = at.SetConfig(c.hostname, c.cfg)
	if err != nil {
		return nil, err
	}

	err = c.cfg.Write()
	if err != nil {
		return nil, err
	}

	return refreshedToken, nil
}

func toOAuth2Token(at *AuthToken) *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  at.AccessToken,
		TokenType:    "", // TODO: what's the proper token type here? Default is Bearer :shrug:
		RefreshToken: at.RefreshToken,
		Expiry:       at.ExpiryDate,
		ExpiresIn:    int64(at.ExpiresIn),
	}
}

func fromOAuth2Token(ot *oauth2.Token) *AuthToken {
	return &AuthToken{
		AccessToken:  ot.AccessToken,
		RefreshToken: ot.RefreshToken,
		ExpiresIn:    int(ot.ExpiresIn),
		ExpiryDate:   ot.Expiry,
		CodeVerifier: "",
	}
}
