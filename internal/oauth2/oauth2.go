package oauth2

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"time"

	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/utils"
	"golang.org/x/oauth2"
)

func StartFlow(ctx context.Context, cfg config.Config, out io.Writer, httpClient *http.Client, hostname string) (string, error) {
	oauth2Config, err := NewOAuth2Config(hostname, cfg)
	if err != nil {
		return "", err
	}

	verifier := oauth2.GenerateVerifier()
	state := oauth2.GenerateVerifier()
	tokenCh, errorCh, shutdownFunc := handleAuthRedirect(ctx, oauth2Config, httpClient, verifier, state)

	browser, _ := cfg.Get(hostname, "browser")
	authURL := oauth2Config.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
	if err := utils.OpenInBrowser(authURL, browser); err != nil {
		fmt.Fprintf(out, "Failed opening a browser at %s\n", authURL)
		fmt.Fprintf(out, "Encountered error: %s\n", err)
		fmt.Fprint(out, "Try entering the URL in your browser manually.\n")
	}

	var token *oauth2.Token
	select {
	case <-ctx.Done():
		if shutdownErr := shutdownFunc(); shutdownErr != nil {
			return "", fmt.Errorf("shutdown error: %s, during cancellation: %w", shutdownErr, ctx.Err())
		}
		return "", ctx.Err()
	case token = <-tokenCh:
		if err := shutdownFunc(); err != nil {
			return "", err
		}
	case err := <-errorCh:
		if shutdownErr := shutdownFunc(); shutdownErr != nil {
			return "", fmt.Errorf("shutdown error: %s, during authorization flow error: %w", shutdownErr, err)
		}
		return "", err
	}

	err = marshal(hostname, cfg, token)
	if err != nil {
		return "", err
	}

	return token.AccessToken, nil
}

func handleAuthRedirect(ctx context.Context, oauth2Config *oauth2.Config, httpClient *http.Client, verifier, state string) (chan *oauth2.Token, chan error, func() error) {
	tokenCh := make(chan *oauth2.Token, 1)
	errorCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/redirect", func(w http.ResponseWriter, r *http.Request) {
		retState := r.URL.Query().Get("state")
		if retState != state {
			err := fmt.Errorf("invalid state")
			errorCh <- err
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			err := fmt.Errorf("no authorization code received")
			errorCh <- err
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Check for errors
		if errorParam := r.URL.Query().Get("error"); errorParam != "" {
			err := fmt.Errorf("authorization error: %s", errorParam)
			errorCh <- err
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(ctx, oauth2.HTTPClient, httpClient)
		token, err := oauth2Config.Exchange(ctx, code, oauth2.S256ChallengeOption(verifier), oauth2.VerifierOption(verifier))
		if err != nil {
			errorCh <- err
			http.Error(w, fmt.Sprintf("Token exchange failed: %v", err), http.StatusInternalServerError)
			return
		}

		// Send success response
		tokenCh <- token
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`
            <html>
            <body style="font-family: Arial, sans-serif; text-align: center; padding: 50px;">
                <h1 style="color: green;">Authentication Successful!</h1>
                <p>You can close this window and return to the application.</p>
            </body>
            </html>
        `))
	})

	server := &http.Server{
		Addr:    ":7171",
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errorCh <- fmt.Errorf("error while listening on OAuth2 callback server: %w", err)
		}
	}()

	return tokenCh, errorCh, func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(ctx)
	}
}
