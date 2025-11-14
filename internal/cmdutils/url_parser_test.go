//go:build !integration

package cmdutils

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
)

func TestParseGitLabURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantID      int
		wantType    GitLabResourceType
		wantRepoURL string
	}{
		// Merge Request URLs
		{
			name:        "valid MR URL",
			url:         "https://gitlab.com/gitlab-org/cli/-/merge_requests/1234",
			wantID:      1234,
			wantType:    GitLabResourceMergeRequest,
			wantRepoURL: "https://gitlab.com/gitlab-org/cli/",
		},
		{
			name:        "valid MR URL without dash",
			url:         "https://gitlab.com/gitlab-org/cli/merge_requests/1234",
			wantID:      1234,
			wantType:    GitLabResourceMergeRequest,
			wantRepoURL: "https://gitlab.com/gitlab-org/cli/",
		},
		{
			name:        "valid MR URL with nested subgroup",
			url:         "https://gitlab.com/namespace/project/subproject/repo/-/merge_requests/100",
			wantID:      100,
			wantType:    GitLabResourceMergeRequest,
			wantRepoURL: "https://gitlab.com/namespace/project/subproject/repo/",
		},
		{
			name:        "valid MR URL with very deep nested subgroups",
			url:         "https://gitlab.com/gitlab-org/subgroup/othersubgroup/oh/please/make/it/stop/-/merge_requests/2091",
			wantID:      2091,
			wantType:    GitLabResourceMergeRequest,
			wantRepoURL: "https://gitlab.com/gitlab-org/subgroup/othersubgroup/oh/please/make/it/stop/",
		},
		{
			name:        "valid MR URL with self-managed instance (invent.kde.org)",
			url:         "https://invent.kde.org/kde/krita/-/merge_requests/42",
			wantID:      42,
			wantType:    GitLabResourceMergeRequest,
			wantRepoURL: "https://invent.kde.org/kde/krita/",
		},
		{
			name:        "valid MR URL with self-managed instance",
			url:         "https://gitlab.example.com/org/project/-/merge_requests/42",
			wantID:      42,
			wantType:    GitLabResourceMergeRequest,
			wantRepoURL: "https://gitlab.example.com/org/project/",
		},
		{
			name:        "MR URL with fragment (note)",
			url:         "https://gitlab.com/gitlab-org/cli/-/merge_requests/2091#note_2510904692",
			wantID:      2091,
			wantType:    GitLabResourceMergeRequest,
			wantRepoURL: "https://gitlab.com/gitlab-org/cli/",
		},
		{
			name:        "MR URL with query parameters",
			url:         "https://gitlab.com/gitlab-org/cli/-/merge_requests/1234?tab=notes",
			wantID:      1234,
			wantType:    GitLabResourceMergeRequest,
			wantRepoURL: "https://gitlab.com/gitlab-org/cli/",
		},
		{
			name:        "MR URL with both query and fragment",
			url:         "https://gitlab.com/gitlab-org/cli/-/merge_requests/1234?tab=notes#note_123",
			wantID:      1234,
			wantType:    GitLabResourceMergeRequest,
			wantRepoURL: "https://gitlab.com/gitlab-org/cli/",
		},
		{
			name:        "MR URL with trailing slash",
			url:         "https://gitlab.com/gitlab-org/cli/-/merge_requests/1234/",
			wantID:      1234,
			wantType:    GitLabResourceMergeRequest,
			wantRepoURL: "https://gitlab.com/gitlab-org/cli/",
		},
		// Invalid URLs
		{
			name:     "invalid URL - no resource ID",
			url:      "https://gitlab.com/gitlab-org/cli/-/merge_requests/",
			wantID:   0,
			wantType: "",
		},
		{
			name:     "invalid URL - not a GitLab resource",
			url:      "https://gitlab.com/gitlab-org/cli",
			wantID:   0,
			wantType: "",
		},
		{
			name:     "invalid URL - wrong protocol",
			url:      "ftp://gitlab.com/gitlab-org/cli/-/merge_requests/123",
			wantID:   0,
			wantType: "",
		},
		{
			name:     "invalid URL - malformed",
			url:      "not-a-url",
			wantID:   0,
			wantType: "",
		},
		{
			name:     "invalid URL - missing repo name",
			url:      "https://gitlab.com/gitlab-org/-/merge_requests/123",
			wantID:   0,
			wantType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseGitLabURL(tt.url, glinstance.DefaultHostname)

			if tt.wantID == 0 {
				require.Nil(t, result, "Expected nil result for invalid URL")
				return
			}

			require.NotNil(t, result, "Expected non-nil result for valid URL")
			require.Equal(t, tt.wantID, result.ID)
			require.Equal(t, tt.wantType, result.Type)

			if tt.wantRepoURL != "" {
				expectedRepo, err := glrepo.FromFullName(tt.wantRepoURL, glinstance.DefaultHostname)
				require.NoError(t, err)
				require.Equal(t, expectedRepo, result.Repo)
			}
		})
	}
}

func TestParseMergeRequestFromURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantID      int
		wantRepoURL string
	}{
		{
			name:        "valid MR URL",
			url:         "https://gitlab.com/gitlab-org/cli/-/merge_requests/1234",
			wantID:      1234,
			wantRepoURL: "https://gitlab.com/gitlab-org/cli/",
		},
		{
			name:   "invalid URL should return 0",
			url:    "not-a-url",
			wantID: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, repo := ParseMergeRequestFromURL(tt.url, glinstance.DefaultHostname)
			require.Equal(t, tt.wantID, id)

			if tt.wantID != 0 && tt.wantRepoURL != "" {
				expectedRepo, err := glrepo.FromFullName(tt.wantRepoURL, glinstance.DefaultHostname)
				require.NoError(t, err)
				require.Equal(t, expectedRepo, repo)
			} else {
				require.Nil(t, repo)
			}
		})
	}
}
