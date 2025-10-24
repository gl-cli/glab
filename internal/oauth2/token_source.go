package oauth2

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"gitlab.com/gitlab-org/api/client-go/gitlaboauth2"
	"gitlab.com/gitlab-org/cli/internal/config"
	"golang.org/x/oauth2"
)

type configTokenSource struct {
	cfg        config.Config
	httpClient *http.Client
	hostname   string

	oauth2Config *oauth2.Config

	// Token is not thread-safe
	mu sync.Mutex
}

func NewConfigTokenSource(cfg config.Config, httpClient *http.Client, protocol, hostname string) (oauth2.TokenSource, error) {
	clientID, err := oauthClientID(cfg, hostname)
	if err != nil {
		return nil, err
	}

	oauth2Config := gitlaboauth2.NewOAuth2Config(fmt.Sprintf("%s://%s", protocol, hostname), clientID, redirectURL, scopes)

	token, err := unmarshal(hostname, cfg)
	if err != nil {
		return nil, err
	}

	src := &configTokenSource{
		cfg:          cfg,
		oauth2Config: oauth2Config,
		httpClient:   httpClient,
		hostname:     hostname,
	}

	return oauth2.ReuseTokenSource(token, src), nil
}

func (c *configTokenSource) Token() (*oauth2.Token, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	token, err := unmarshal(c.hostname, c.cfg)
	if err != nil {
		return nil, err
	}

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, c.httpClient)
	refreshedToken, err := c.oauth2Config.TokenSource(ctx, token).Token()
	if err != nil {
		return nil, err
	}

	err = marshal(c.hostname, c.cfg, refreshedToken)
	if err != nil {
		return nil, err
	}

	err = c.cfg.Write()
	if err != nil {
		return nil, err
	}

	return refreshedToken, nil
}
