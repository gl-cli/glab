package oauth2

import (
	"crypto/sha256"
	"encoding/base64"
	"math/rand"
	"time"
)

const (
	charset = "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	length = 45
)

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func stringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func randomString() string {
	return stringWithCharset(length, charset)
}

func genSha256(codeVerifier string) []byte {
	hashFn := sha256.New()

	hashFn.Write([]byte(codeVerifier))
	b := hashFn.Sum(nil)

	return b
}

func generateCodeChallenge(codeVerifier string) string {
	sha := genSha256(codeVerifier)
	return base64.RawURLEncoding.EncodeToString(sha)
}
