package oauth2

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/utils"
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
	return glinstance.DefaultClientID, nil
}

func StartFlow(cfg config.Config, out io.Writer, httpClient *http.Client, hostname string) (string, error) {
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

	tokenCh := handleAuthRedirect(out, httpClient, "0.0.0.0", codeVerifier, hostname, "https", clientID, state)
	defer close(tokenCh)

	browser, _ := cfg.Get(hostname, "browser")
	if err := utils.OpenInBrowser(completeAuthURL, browser); err != nil {
		fmt.Fprintf(out, "Failed opening a browser at %s\n", completeAuthURL)
		fmt.Fprintf(out, "Encountered error: %s\n", err)
		fmt.Fprint(out, "Try entering the URL in your browser manually.\n")
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

func handleAuthRedirect(out io.Writer, httpClient *http.Client, listenHostname, codeVerifier, hostname, protocol, clientID, originalState string) chan *AuthToken {
	tokenCh := make(chan *AuthToken)

	server := &http.Server{Addr: listenHostname + ":7171"}

	http.HandleFunc("/auth/redirect", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		if state != originalState {
			fmt.Fprintf(out, "Error: Invalid state")
			tokenCh <- nil
			return
		}

		token, err := requestToken(httpClient, hostname, protocol, clientID, code, codeVerifier)
		if err != nil {
			fmt.Fprintf(out, "Error occured requesting access token %s.", err)
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
			fmt.Fprintf(out, "Error occured while setting up server %s.", err)
			tokenCh <- nil
		}
	}()

	return tokenCh
}

func requestToken(client *http.Client, hostname, protocol, clientID, code, codeVerifier string) (*AuthToken, error) {
	tokenURL := fmt.Sprintf("%s://%s/oauth/token", protocol, hostname)

	form := url.Values{
		"client_id":     []string{clientID},
		"code":          []string{code},
		"grant_type":    []string{"authorization_code"},
		"redirect_uri":  []string{redirectURI},
		"code_verifier": []string{codeVerifier},
	}

	resp, err := client.PostForm(tokenURL, form)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusBadRequest {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bad request: %s\n", string(respBody))
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
