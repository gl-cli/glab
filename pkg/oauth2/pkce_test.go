package oauth2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRandomStringGeneratesANewRandomString(t *testing.T) {
	first := GenerateCodeChallenge()
	second := GenerateCodeChallenge()
	require.NotEqual(t, first, second)
}
