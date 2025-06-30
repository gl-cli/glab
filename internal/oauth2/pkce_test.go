package oauth2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRandomStringGeneratesANewRandomString(t *testing.T) {
	first := GenerateCodeVerifier()
	second := GenerateCodeVerifier()
	require.NotEqual(t, first, second)
}
