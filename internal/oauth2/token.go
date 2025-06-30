package oauth2

import (
	"time"

	"gitlab.com/gitlab-org/cli/internal/config"
)

type AuthToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int       `json:"expires_in"`
	ExpiryDate   time.Time `json:"expires_date"`
	CodeVerifier string    `json:"code_verifier"`
}

func (t *AuthToken) CalcExpiresDate() {
	t.ExpiryDate = time.Now().Add(time.Second * time.Duration(t.ExpiresIn))
}

func tokenFromConfig(hostname string, cfg config.Config) (*AuthToken, error) {
	result := &AuthToken{}
	var err error

	expiryDateString, err := cfg.Get(hostname, "oauth2_expiry_date")
	if err != nil {
		return nil, err
	}

	result.ExpiryDate, err = time.Parse(time.RFC822, expiryDateString)
	if err != nil {
		return nil, err
	}

	result.RefreshToken, err = cfg.Get(hostname, "oauth2_refresh_token")
	if err != nil {
		return nil, err
	}

	result.CodeVerifier, err = cfg.Get(hostname, "oauth2_code_verifier")
	if err != nil {
		return nil, err
	}

	result.AccessToken, err = cfg.Get(hostname, "token")
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (token *AuthToken) SetConfig(hostname string, cfg config.Config) error {
	err := cfg.Set(hostname, "is_oauth2", "true")
	if err != nil {
		return err
	}

	err = cfg.Set(hostname, "oauth2_refresh_token", token.RefreshToken)
	if err != nil {
		return err
	}

	token.CalcExpiresDate()
	err = cfg.Set(hostname, "oauth2_expiry_date", token.ExpiryDate.Format(time.RFC822))
	if err != nil {
		return err
	}

	err = cfg.Set(hostname, "token", token.AccessToken)
	if err != nil {
		return err
	}

	err = cfg.Set(hostname, "oauth2_code_verifier", token.CodeVerifier)
	if err != nil {
		return err
	}

	return nil
}
