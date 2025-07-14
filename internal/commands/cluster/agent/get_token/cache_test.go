package get_token

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func TestCache_unpopulated(t *testing.T) {
	// GIVEN
	createdPAT := &gitlab.PersonalAccessToken{
		Name:      "sentinel",
		Token:     "redacted",
		ExpiresAt: gitlab.Ptr(gitlab.ISOTime{}),
	}
	c := cache{
		dir:    testDir(t),
		prefix: "test-",
		createFunc: func() (*gitlab.PersonalAccessToken, error) {
			return createdPAT, nil
		},
	}

	// WHEN
	pat, err := c.get()

	// THEN
	require.NoError(t, err)
	require.Equal(t, createdPAT, pat)
}

func TestCache_createError(t *testing.T) {
	// GIVEN
	expectedErr := errors.New("sentinel")
	c := cache{
		dir:    testDir(t),
		prefix: "test-",
		createFunc: func() (*gitlab.PersonalAccessToken, error) {
			return nil, expectedErr
		},
	}

	// WHEN
	_, err := c.get()

	// THEN
	require.ErrorIs(t, err, expectedErr)
}

func TestCache_hit(t *testing.T) {
	// GIVEN
	c := cache{
		dir:    testDir(t),
		prefix: "test-",
		createFunc: func() (*gitlab.PersonalAccessToken, error) {
			return nil, errors.New("sentinel - should not be called")
		},
	}

	// populate cache
	err := c.write(&gitlab.PersonalAccessToken{
		Token:     "any-token",
		ExpiresAt: gitlab.Ptr(gitlab.ISOTime(time.Now().Add(1 * time.Minute))),
	})
	require.NoError(t, err)

	// WHEN
	pat, err := c.get()

	// THEN
	require.NoError(t, err)
	require.Equal(t, pat.Token, "any-token")
}

func testDir(t *testing.T) *os.Root {
	t.Helper()

	d, err := os.OpenRoot(t.TempDir())
	require.NoError(t, err)

	return d
}
