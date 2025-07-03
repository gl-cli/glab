package generate

import (
	"crypto"
	"crypto/elliptic"
	"crypto/rsa"
	"path/filepath"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:generate go run testdata/generate_test_keys.go

// Helper function to load pre-generated test keys
func loadTestKey(t *testing.T, filename string) crypto.PrivateKey {
	t.Helper()

	key, err := loadPrivateKey(filepath.Join("testdata", filename), nil)
	require.NoError(t, err)
	return key
}

func getEd25519PrivateKey(t *testing.T) crypto.PrivateKey {
	t.Helper()

	return loadTestKey(t, "ed25519_key.pem")
}

func getEcdsaPrivateKey(t *testing.T, curve elliptic.Curve) crypto.PrivateKey {
	t.Helper()

	var filename string
	switch curve {
	case elliptic.P256():
		filename = "ecdsa_p256_key.pem"
	case elliptic.P384():
		filename = "ecdsa_p384_key.pem"
	case elliptic.P521():
		filename = "ecdsa_p521_key.pem"
	case elliptic.P224():
		filename = "ecdsa_p224_key.pem"
	default:
		t.Fatalf("unsupported ecdsa key for curve %v", curve)
	}
	return loadTestKey(t, filename)
}

func getRsaPrivateKey(t *testing.T, bits int) crypto.PrivateKey {
	t.Helper()

	var filename string
	switch bits {
	case 2047:
		filename = "rsa_2047_key.pem"
	case 8193:
		filename = "rsa_8193_key.pem"
	case 2048:
		filename = "rsa_2048_key.pem"
	case 8192:
		filename = "rsa_8192_key.pem"
	default:
		t.Fatalf("unsupported RSA key size %d", bits)
	}
	return loadTestKey(t, filename)
}

func TestSigningMethods(t *testing.T) {
	tests := []struct {
		name                  string
		key                   crypto.PrivateKey
		expectedSigningMethod jwt.SigningMethod
		shouldError           bool
	}{
		{
			name:                  "supported ed25519",
			key:                   getEd25519PrivateKey(t),
			expectedSigningMethod: jwt.SigningMethodEdDSA,
		},
		{
			name:        "unsupported Ecdsa P256",
			key:         getEcdsaPrivateKey(t, elliptic.P256()),
			shouldError: true, // GitLab only support the _sk variant which isn't supported by glab yet
		},
		{
			name:        "unsupported Ecdsa P384",
			key:         getEcdsaPrivateKey(t, elliptic.P384()),
			shouldError: true, // GitLab only support the _sk variant which isn't supported by glab yet
		},
		{
			name:        "unsupported Ecdsa 521",
			key:         getEcdsaPrivateKey(t, elliptic.P521()),
			shouldError: true, // GitLab only support the _sk variant which isn't supported by glab yet
		},
		{
			name:        "unsupported Ecdsa 224",
			key:         getEcdsaPrivateKey(t, elliptic.P224()),
			shouldError: true, // ssh-keygen doesn't do 224 bit keys
		},
		{
			name:        "unsupported RSA 2047",
			key:         getRsaPrivateKey(t, 2047),
			shouldError: true,
		},
		{
			name:        "unsupported RSA 8193",
			key:         getRsaPrivateKey(t, 8193),
			shouldError: true,
		},
		{
			name:                  "supported RSA 2048",
			key:                   getRsaPrivateKey(t, 2048),
			expectedSigningMethod: jwt.SigningMethodRS512,
		},
		{
			name:                  "supported RSA 8192",
			key:                   getRsaPrivateKey(t, 8192),
			expectedSigningMethod: jwt.SigningMethodRS512,
		},
		{
			name:        "unsupported unknown private key",
			key:         crypto.PrivateKey(1),
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signingMethod, err := getSigningMethod(tt.key)
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedSigningMethod, signingMethod)
			}
		})
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
