package recovery_test

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/internal/recovery"
)

type S struct {
	Field   string `json:"field,omitempty"`
	Number  int    `json:"number,omitempty"`
	Boolean bool   `json:"boolean,omitempty"`
}

var sample = S{
	Field:   "field",
	Number:  123,
	Boolean: true,
}

func TestCreateRecoverFile(t *testing.T) {
	d := t.TempDir()
	t.Setenv("GLAB_CONFIG_DIR", d)

	tmpFile, err := recovery.CreateFile("repo/name", "struct.json", &sample)
	require.NoError(t, err)

	fi, err := os.Stat(tmpFile)
	require.NoError(t, err)

	require.NotZero(t, fi.Size())

	f, err := os.Open(tmpFile)
	require.NoError(t, err)
	b, err := io.ReadAll(f)
	require.NoError(t, err)

	expected, err := json.Marshal(sample)
	require.NoError(t, err)

	require.Equal(t, string(expected)+"\n", string(b))
}

func TestFromFile(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	// load file contents from ./testdata/struct.json
	d := filepath.Join(wd, "testdata")
	t.Setenv("GLAB_CONFIG_DIR", d)

	defer func() {
		// create file again because `recovery.FromFile` removes it at the end
		_, err := recovery.CreateFile("repo/name", "struct.json", sample)
		require.NoError(t, err)
	}()

	var got S

	err = recovery.FromFile("repo/name", "struct.json", &got)
	require.NoError(t, err)

	require.Equal(t, sample, got)
}

func TestRecovery(t *testing.T) {
	d := t.TempDir()
	t.Setenv("GLAB_CONFIG_DIR", d)

	_, err := recovery.CreateFile("repo/name", "struct.json", &sample)
	require.NoError(t, err)

	var got S
	err = recovery.FromFile("repo/name", "struct.json", &got)
	require.NoError(t, err)

	require.Equal(t, sample, got)
}
