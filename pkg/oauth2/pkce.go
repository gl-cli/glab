package oauth2

import (
	oauthStd "golang.org/x/oauth2"
)

func GenerateCodeVerifier() string {
	return oauthStd.GenerateVerifier()
}

func GenerateCodeChallenge(codeVerifier string) string {
	return oauthStd.S256ChallengeFromVerifier(codeVerifier)
}
