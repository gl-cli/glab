package get_token

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testLockFilePrefix = "test-lock-"

func TestFileMutex(t *testing.T) {
	root, err := lockFileBaseDir()
	require.NoError(t, err)

	t.Cleanup(func() {
		files, err := os.ReadDir(root.Name())
		require.NoError(t, err)

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			if strings.HasPrefix(file.Name(), testLockFilePrefix) {
				err := root.Remove(file.Name())
				assert.NoError(t, err)
			}
		}
	})

	t.Run("success", func(t *testing.T) {
		// GIVEN
		expected := "test-result"

		// WHEN
		actual, err := withLock(t.Context(), lockFileName(t), func() (*string, error) {
			return &expected, nil
		})
		require.NoError(t, err)

		// THEN
		require.NotNil(t, actual)
		assert.Equal(t, expected, *actual)
	})

	t.Run("function error", func(t *testing.T) {
		// GIVEN
		expectedErr := errors.New("function error")

		// WHEN
		result, err := withLock(t.Context(), lockFileName(t), func() (*string, error) {
			return nil, expectedErr
		})

		// THEN
		require.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
		assert.Nil(t, result)
	})

	t.Run("context cancellation", func(t *testing.T) {
		// GIVEN
		ctx, cancel := context.WithCancel(t.Context())

		// WHEN
		cancel()
		_, err := withLock(ctx, lockFileName(t), func() (*string, error) {
			return nil, assert.AnError
		})

		// THEN
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("lock file is created", func(t *testing.T) {
		// GIVEN
		n := lockFileName(t)
		fm := &fileMutex{filename: n, root: root}

		// WHEN
		err := fm.lock(t.Context())

		// THEN
		require.NoError(t, err)
		require.FileExists(t, filepath.Join(root.Name(), n))
	})

	t.Run("already locked", func(t *testing.T) {
		// GIVEN
		n := lockFileName(t)
		fm := &fileMutex{filename: n, root: root}
		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()

		// WHEN
		// lock
		err := fm.lock(t.Context())
		require.NoError(t, err)

		// already locked
		err = fm.lock(ctx)

		// THEN
		require.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("stale lock cleanup", func(t *testing.T) {
		// GIVEN
		n := lockFileName(t)
		fm := &fileMutex{filename: n, root: root}

		// create lock file
		err := fm.lock(t.Context())
		require.NoError(t, err)

		// date back the lock file
		staleTime := time.Now().Add(-(staleTimeout * 2))
		err = os.Chtimes(filepath.Join(root.Name(), n), staleTime, staleTime)
		require.NoError(t, err)

		// WHEN
		// able to aquire lock, because it cleans up the stale lock file
		err = fm.lock(t.Context())

		// THEN
		assert.NoError(t, err)
	})

	t.Run("unlock", func(t *testing.T) {
		// GIVEN
		n := lockFileName(t)
		fm := &fileMutex{filename: n, root: root}

		// create lock file
		err := fm.lock(t.Context())
		require.NoError(t, err)

		// WHEN
		err = fm.unlock()

		// THEN
		assert.NoError(t, err)
		assert.NoFileExists(t, filepath.Join(root.Name(), n))
	})
}

func lockFileName(t *testing.T) string {
	t.Helper()

	return testLockFilePrefix + base64.StdEncoding.EncodeToString([]byte(t.Name()))
}
