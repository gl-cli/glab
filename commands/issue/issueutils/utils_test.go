package issueutils

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
)

func Test_issueMetadataFromURL(t *testing.T) {
	tests := []struct {
		name string
		str  string
		want int
		path string
	}{
		{
			name: "valid URL",
			str:  "https://gitlab.com/namespace/repo/-/issues/1",
			want: 1,
			path: "https://gitlab.com/namespace/repo/",
		},
		{
			name: "valid URL with nested subgroup",
			str:  "https://gitlab.com/namespace/project/subproject/repo/-/issues/100",
			want: 100,
			path: "https://gitlab.com/namespace/project/subproject/repo/",
		},
		{
			name: "valid URL without dash",
			str:  "https://gitlab.com/namespace/project/subproject/repo/issues/1",
			want: 1,
			path: "https://gitlab.com/namespace/project/subproject/repo/",
		},
		{
			name: "valid incident URL",
			str:  "https://gitlab.com/namespace/repo/-/issues/incident/1",
			want: 1,
			path: "https://gitlab.com/namespace/repo/",
		},
		{
			name: "valid incident URL with nested subgroup",
			str:  "https://gitlab.com/namespace/project/subproject/repo/-/issues/incident/100",
			want: 100,
			path: "https://gitlab.com/namespace/project/subproject/repo/",
		},
		{
			name: "valid incident URL without dash",
			str:  "https://gitlab.com/namespace/project/subproject/repo/issues/incident/1",
			want: 1,
			path: "https://gitlab.com/namespace/project/subproject/repo/",
		},
		{
			name: "invalid URL with no issue number",
			str:  "https://gitlab.com/namespace/project/subproject/repo/issues",
			want: 0,
			path: "",
		},
		{
			name: "invalid incident URL with no incident number",
			str:  "https://gitlab.com/namespace/project/subproject/repo/issues/incident",
			want: 0,
			path: "",
		},
		{
			name: "invalid URL with only namespace, missing repo",
			str:  "https://gitlab.com/namespace/issues/100",
			want: 0,
			path: "",
		},
		{
			name: "invalid incident URL with only namespace, missing repo",
			str:  "https://gitlab.com/namespace/issues/incident/100",
			want: 0,
			path: "",
		},
		{
			name: "invalid issue URL",
			str:  "https://gitlab.com/namespace/repo",
			want: 0,
			path: "",
		},
		{
			name: "invalid issue URL, missing issues path",
			str:  "https://gitlab.com/namespace/project/subproject/repo/10/",
			want: 0,
			path: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, repo := issueMetadataFromURL(tt.str)
			require.Equal(t, tt.want, id)

			if tt.want != 0 && tt.path != "" {
				expectedRepo, err := glrepo.FromFullName(tt.path)
				require.NoError(t, err)
				require.Equal(t, expectedRepo, repo)
			}
		})
	}
}
