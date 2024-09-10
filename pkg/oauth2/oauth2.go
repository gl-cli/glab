package oauth2

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/pkg/glinstance"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/pkg/utils"
)

const (
	redirectURI = "http://localhost:7171/auth/redirect"
	scopes      = "openid+profile+read_user+write_repository+api"
)

func oAuthClientID(cfg config.Config, hostname string) (string, error) {
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
	return glinstance.DefaultClientID(), nil
}

func StartFlow(cfg config.Config, io *iostreams.IOStreams, hostname string) (string, error) {
	authURL := fmt.Sprintf("https://%s/oauth/authorize", hostname)

	clientID, err := oAuthClientID(cfg, hostname)
	if err != nil {
		return "", err
	}

	state := GenerateCodeVerifier()
	codeVerifier := GenerateCodeVerifier()
	codeChallenge := GenerateCodeChallenge(codeVerifier)
	completeAuthURL := fmt.Sprintf(
		"%s?client_id=%s&redirect_uri=%s&response_type=code&state=%s&scope=%s&code_challenge=%s&code_challenge_method=S256",
		authURL, clientID, redirectURI, state, scopes, codeChallenge)

	tokenCh := handleAuthRedirect(io, "0.0.0.0", codeVerifier, hostname, "https", clientID, state)
	defer close(tokenCh)

	browser, _ := cfg.Get(hostname, "browser")
	if err := utils.OpenInBrowser(completeAuthURL, browser); err != nil {
		fmt.Fprintf(io.StdErr, "Failed opening a browser at %s\n", completeAuthURL)
		fmt.Fprintf(io.StdErr, "Encountered error: %s\n", err)
		fmt.Fprint(io.StdErr, "Try entering the URL in your browser manually.\n")
	}
	token := <-tokenCh
	if token == nil {
		return "", fmt.Errorf("authentication failed: no token received")
	}

	err = token.SetConfig(hostname, cfg)
	if err != nil {
		return "", err
	}

	return token.AccessToken, nil
}

func handleAuthRedirect(io *iostreams.IOStreams, listenHostname, codeVerifier, hostname, protocol, clientID, originalState string) chan *AuthToken {
	tokenCh := make(chan *AuthToken)

	server := &http.Server{Addr: listenHostname + ":7171"}

	http.HandleFunc("/auth/redirect", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		if state != originalState {
			fmt.Fprintf(io.StdErr, "Error: Invalid state")
			tokenCh <- nil
			return
		}

		token, err := requestToken(hostname, protocol, clientID, code, codeVerifier)
		if err != nil {
			fmt.Fprintf(io.StdErr, "Error occured requesting access token %s.", err)
			tokenCh <- nil
			return
		}

		_, _ = w.Write([]byte("You have authenticated successfully. You can now close this browser window."))
		tokenCh <- token
		_ = server.Shutdown(context.Background())
	})

	go func() {
		err := http.ListenAndServe(listenHostname+":7171", nil)
		if err != nil {
			fmt.Fprintf(io.StdErr, "Error occured while setting up server %s.", err)
			tokenCh <- nil
		}
	}()

	return tokenCh
}

func requestToken(hostname, protocol, clientID, code, codeVerifier string) (*AuthToken, error) {
	tokenURL := fmt.Sprintf("%s://%s/oauth/token", protocol, hostname)

	form := url.Values{
		"client_id":     []string{clientID},
		"code":          []string{code},
		"grant_type":    []string{"authorization_code"},
		"redirect_uri":  []string{redirectURI},
		"code_verifier": []string{codeVerifier},
	}

	resp, err := http.PostForm(tokenURL, form)

	if resp.StatusCode == http.StatusBadRequest {
		respBody, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("bad request: %s\n", string(respBody))
	}

	if err != nil {
		return nil, err
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	at := AuthToken{}

	err = json.Unmarshal(respBytes, &at)
	if err != nil {
		return nil, err
	}

	at.CalcExpiresDate()
	at.CodeVerifier = codeVerifier

	return &at, nil
}

func RefreshToken(hostname string, cfg config.Config, protocol string) error {
	token, err := tokenFromConfig(hostname, cfg)
	if err != nil {
		return err
	}

	// Check if token has expired
	if token.ExpiryDate.After(time.Now()) {
		return nil
	}

	clientID, err := oAuthClientID(cfg, hostname)
	if err != nil {
		return err
	}

	form := url.Values{
		"client_id":     []string{clientID},
		"grant_type":    []string{"refresh_token"},
		"refresh_token": []string{token.RefreshToken},
		"redirect_uri":  []string{redirectURI},
		"code_verifier": []string{token.CodeVerifier},
	}

	tokenURL := fmt.Sprintf("%s://%s/oauth/token", protocol, hostname)
	resp, err := http.PostForm(tokenURL, form)
	if err != nil {
		return err
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	at := AuthToken{}

	err = json.Unmarshal(respBytes, &at)
	if err != nil {
		return err
	}

	at.CalcExpiresDate()

	err = at.SetConfig(hostname, cfg)
	if err != nil {
		return err
	}

	err = cfg.Write()
	if err != nil {
		return err
	}

	return nil
}
