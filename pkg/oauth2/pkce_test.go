package oauth2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateCodeChallenge(t *testing.T) {
	got := generateCodeChallenge("ks02i3jdikdo2k0dkfodf3m39rjfjsdk0wk349rj3jrhf")
	want := "2i0WFA-0AerkjQm4X4oDEhqA17QIAKNjXpagHBXmO_U"
	require.Equal(t, want, got)
}

func TestRandomStringGeneratesANewRandomString(t *testing.T) {
	first := randomString()
	second := randomString()
	require.NotEqual(t, first, second)
}
