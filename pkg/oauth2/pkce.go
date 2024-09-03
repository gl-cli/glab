package oauth2

import (
	oauthStd "golang.org/x/oauth2"
)

func GenerateCodeVerifier() string {
	return oauthStd.GenerateVerifier()
}

func GenerateCodeChallenge() string {
	return oauthStd.S256ChallengeFromVerifier(GenerateCodeVerifier())
}
