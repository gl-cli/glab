//go:build !integration

package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_CheckPathExists(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		dir := t.TempDir()

		got := CheckPathExists(dir)
		assert.True(t, got)
	})
	t.Run("doesnt-exist", func(t *testing.T) {
		got := CheckPathExists("/Path/Not/Exist")
		assert.False(t, got)
	})
}

func Test_CheckFileExists(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "")
	if err != nil {
		t.Skipf("Unexpected error creating temporary file for testing = %s", err)
	}
	fPath := file.Name()
	require.NoError(t, file.Close())

	t.Run("exists", func(t *testing.T) {
		got := CheckFileExists(fPath)
		assert.True(t, got)
	})

	t.Run("doesnt-exist", func(t *testing.T) {
		got := CheckFileExists("/Path/Not/Exist")
		assert.False(t, got)
	})
}

func Test_BackupConfigFile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		file, err := os.CreateTemp(t.TempDir(), "")
		if err != nil {
			t.Skipf("Unexpected error creating temporary file for testing = %s", err)
		}
		fPath := file.Name()
		require.NoError(t, file.Close())

		err = BackupConfigFile(fPath)
		if err != nil {
			t.Errorf("Unexpected error = %s", err)
		}

		got := CheckFileExists(fPath + ".bak")
		assert.True(t, got)
	})
	t.Run("failure", func(t *testing.T) {
		err := BackupConfigFile("/Path/Not/Exist")
		assert.EqualError(t, err, "rename /Path/Not/Exist /Path/Not/Exist.bak: no such file or directory")
	})
}
