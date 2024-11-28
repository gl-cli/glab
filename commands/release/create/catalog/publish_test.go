package catalog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_fetchTemplates(t *testing.T) {
	err := os.Chdir("./testdata/test-repo")
	require.NoError(t, err)
	t.Cleanup(func() {
		err := os.Chdir("../..")
		require.NoError(t, err)
	})

	wd, err := os.Getwd()
	require.NoError(t, err)
	want := map[string]string{
		"component-1": filepath.Join(wd, "templates/component-1.yml"),
		"component-2": filepath.Join(wd, "templates/component-2.yml"),
		"component-3": filepath.Join(wd, "templates/component-3", "template.yml"),
	}
	got, err := fetchTemplates(wd)
	require.NoError(t, err)

	for k, v := range want {
		require.Equal(t, got[k], v)
	}
}

func Test_extractComponentName(t *testing.T) {
	err := os.Chdir("./testdata/test-repo")
	require.NoError(t, err)
	t.Cleanup(func() {
		err := os.Chdir("../..")
		require.NoError(t, err)
	})

	wd, err := os.Getwd()
	require.NoError(t, err)

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "valid component path",
			path:     filepath.Join(wd, "templates/component-1.yml"),
			expected: "component-1",
		},
		{
			name:     "valid component path",
			path:     filepath.Join(wd, "templates/component-2", "template.yml"),
			expected: "component-2",
		},
		{
			name:     "invalid component path",
			path:     filepath.Join(wd, "abc_templates/component-3.yml"),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractComponentName(wd, tt.path)
			require.NoError(t, err)
			require.Equal(t, tt.expected, got)
		})
	}
}
