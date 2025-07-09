package oauth2

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"gitlab.com/gitlab-org/api/client-go/gitlaboauth2"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/utils"
	"golang.org/x/oauth2"
)

func StartFlow(ctx context.Context, cfg config.Config, out io.Writer, httpClient *http.Client, hostname string) (string, error) {
	clientID, err := oauthClientID(cfg, hostname)
	if err != nil {
		return "", err
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)
	baseURL := fmt.Sprintf("%s://%s", glinstance.DefaultProtocol, hostname)

	token, err := gitlaboauth2.AuthorizationFlow(ctx, baseURL, clientID, redirectURL, scopes, callbackServerListenAddr, func(url string) error {
		browser, _ := cfg.Get(hostname, "browser")
		if err := utils.OpenInBrowser(url, browser); err != nil {
			fmt.Fprintf(out, "Failed opening a browser at %s\n", url)
			fmt.Fprintf(out, "Encountered error: %s\n", err)
			fmt.Fprint(out, "Try entering the URL in your browser manually.\n")
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	err = marshal(hostname, cfg, token)
	if err != nil {
		return "", err
	}

	return token.AccessToken, nil
}
