package generate

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ed25519"
)

func getEd25519PrivateKey() crypto.PrivateKey {
	_, pk, _ := ed25519.GenerateKey(rand.Reader)
	return &pk
}

func getEcdsaPrivateKey(curve elliptic.Curve) crypto.PrivateKey {
	pk, _ := ecdsa.GenerateKey(curve, rand.Reader)
	return pk
}

func getRsaPrivateKey(bits int) crypto.PrivateKey {
	pk, _ := rsa.GenerateKey(rand.Reader, bits)
	return pk
}

func TestSigningMethods(t *testing.T) {
	testCases := []struct {
		key                   crypto.PrivateKey
		expectedSigningMethod jwt.SigningMethod
		shouldError           bool
	}{
		{
			key:                   getEd25519PrivateKey(),
			expectedSigningMethod: jwt.SigningMethodEdDSA,
		},
		{
			key:         getEcdsaPrivateKey(elliptic.P256()),
			shouldError: true, // GitLab only support the _sk variant which isn't supported by glab yet
		},
		{
			key:         getEcdsaPrivateKey(elliptic.P384()),
			shouldError: true, // GitLab only support the _sk variant which isn't supported by glab yet
		},
		{
			key:         getEcdsaPrivateKey(elliptic.P521()),
			shouldError: true, // GitLab only support the _sk variant which isn't supported by glab yet
		},
		{
			key:         getEcdsaPrivateKey(elliptic.P224()),
			shouldError: true, // ssh-keygen doesn't do 224 bit keys
		},
		{
			key:         getRsaPrivateKey(2047),
			shouldError: true,
		},
		{
			key:         getRsaPrivateKey(8193),
			shouldError: true,
		},
		{
			key:                   getRsaPrivateKey(2048),
			expectedSigningMethod: jwt.SigningMethodRS512,
		},
		{
			key:                   getRsaPrivateKey(8192),
			expectedSigningMethod: jwt.SigningMethodRS512,
		},
		{
			key:         crypto.PrivateKey(1),
			shouldError: true,
		},
	}

	for _, testCase := range testCases {
		signingMethod, err := getSigningMethod(testCase.key)
		if testCase.shouldError {
			assert.Error(t, err)
		} else {
			assert.Nil(t, err)
			assert.Equal(t, testCase.expectedSigningMethod, signingMethod)
		}
	}
}

type TestPasswordReader struct {
	password string
}

func NewTestPasswordReader(password string) *TestPasswordReader {
	return &TestPasswordReader{password}
}

func (pr TestPasswordReader) Read() ([]byte, error) {
	return []byte(pr.password), nil
}

func TestLoadPrivateKey(t *testing.T) {
	testData := []struct {
		path           string
		passwordReader PasswordReader
		shouldError    bool
	}{
		{
			path:        "./testdata/file_does_not_exist",
			shouldError: true,
		},
		{
			path: "./testdata/no_password.pem",
		},
		{
			path:           "./testdata/with_password.pem",
			passwordReader: NewTestPasswordReader("test_password"),
		},
		{
			path:           "./testdata/with_password.pem",
			passwordReader: NewTestPasswordReader("wrong_password"),
			shouldError:    true,
		},
		{
			path:        "./testdata/not_a_key.txt",
			shouldError: true,
		},
	}

	for _, testCase := range testData {
		privateKey, err := loadPrivateKey(testCase.path, testCase.passwordReader)
		if testCase.shouldError {
			assert.Error(t, err)
		} else {
			assert.IsType(t, &rsa.PrivateKey{}, privateKey)
		}
	}
}
