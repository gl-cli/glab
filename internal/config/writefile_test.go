package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_WriteFile(t *testing.T) {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Skipf("unexpected error while creating temporary directory = %s", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	testCases := []struct {
		name        string
		filePath    string
		content     string
		permissions os.FileMode
		isSymlink   bool
	}{
		{
			name:        "regular",
			filePath:    "test-file",
			content:     "profclems/glab",
			permissions: 0o644,
			isSymlink:   false,
		},
		{
			name:        "config",
			filePath:    "config-file",
			content:     "profclems/glab/config",
			permissions: 0o600,
			isSymlink:   false,
		},
		{
			name:        "symlink",
			filePath:    "test-file",
			content:     "profclems/glab/symlink",
			permissions: 0o644,
			isSymlink:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fullPath := filepath.Join(dir, tc.filePath)

			if tc.isSymlink {
				symPath := filepath.Join(dir, "test-symlink")
				require.Nil(t, os.Symlink(tc.filePath, symPath), "failed to create a symlink")
				fullPath = symPath
			}

			require.Nilf(t,
				WriteFile(fullPath, []byte(tc.content), tc.permissions),
				"unexpected error for testCase %q", tc.name,
			)

			result, err := os.ReadFile(fullPath)
			require.Nilf(t, err, "failed to read file %q due to %q", fullPath, err)
			assert.Equal(t, tc.content, string(result))

			fileInfo, err := os.Lstat(fullPath)
			require.Nil(t, err, "failed to get info about the file", err)

			if tc.isSymlink {
				assert.Equal(t, os.ModeSymlink, fileInfo.Mode()&os.ModeSymlink, "this file should be a symlink")
			} else {
				assert.Equal(t, tc.permissions, fileInfo.Mode())
			}
		})
	}
}
