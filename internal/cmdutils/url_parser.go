// This file implements URL parsing for GitLab resources to allow commands
// to accept full GitLab URLs instead of just resource IDs.
//
// TODO: Unify this URL parsing logic with issueutils.issueMetadataFromURL
// in a follow-up MR to avoid code duplication and ensure consistency.
//
// Supported URL formats:
// - Merge Requests: https://gitlab.com/group/project/-/merge_requests/123
// - Issues: https://gitlab.com/group/project/-/issues/456
// - Self-managed instances: https://invent.kde.org/kde/project/-/merge_requests/42
// - Deep subgroups: https://gitlab.com/group/sub1/sub2/project/-/merge_requests/123
// - URL fragments and query parameters are automatically stripped
package cmdutils

import (
	"net/url"
	"regexp"
	"strconv"

	"gitlab.com/gitlab-org/cli/internal/glrepo"
)

// GitLabResourceType represents the type of GitLab resource
type GitLabResourceType string

const (
	// GitLabResourceMergeRequest represents a merge request resource
	GitLabResourceMergeRequest GitLabResourceType = "merge_request"

	// GitLabResourceIssue represents an issue resource
	GitLabResourceIssue GitLabResourceType = "issue"
)

// GitLabResourceMetadata contains the parsed metadata from a GitLab URL
type GitLabResourceMetadata struct {
	ID   int
	Type GitLabResourceType
	Repo glrepo.Interface
}

// mergeRequestURLPathRE is a regex which matches the following patterns:
//
//	OWNER/REPO/merge_requests/id
//	OWNER/REPO/-/merge_requests/id
//	GROUP/NAMESPACE/REPO/merge_requests/id
//	GROUP/NAMESPACE/REPO/-/merge_requests/id
//	including nested subgroups:
//	GROUP/SUBGROUP/../../REPO/-/merge_requests/id
//
// This pattern avoids backtracking issues by using a more specific pattern
// instead of .* which can cause catastrophic backtracking in edge cases.
var mergeRequestURLPathRE = regexp.MustCompile(`^(/(?:[^-][^/]+/){2,})+(?:-/)?merge_requests/(\d+)(?:/.*)?$`)

// issueURLPathRE is a regex which matches the following patterns (duplicated from issueutils for consistency):
//
//	OWNER/REPO/issues/id
//	OWNER/REPO/-/issues/id
//	OWNER/REPO/-/issues/incident/id
//	GROUP/NAMESPACE/REPO/issues/id
//	GROUP/NAMESPACE/REPO/-/issues/id
//	GROUP/NAMESPACE/REPO/-/issues/incident/id
//	including nested subgroups:
//	GROUP/SUBGROUP/../../REPO/-/issues/id
//	GROUP/SUBGROUP/../../REPO/-/issues/incident/id
//
// TODO: Unify this with issueutils.issueURLPathRE in a follow-up MR.
var issueURLPathRE = regexp.MustCompile(`^(/(?:[^-][^/]+/){2,})+(?:-/)?issues/(?:incident/)?(\d+)$`)

func ParseGitLabURL(urlStr, defaultHostname string) *GitLabResourceMetadata {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil
	}

	// Allow URLs without scheme or with http/https
	if u.Scheme != "" && u.Scheme != "https" && u.Scheme != "http" {
		return nil
	}

	// Try to match merge request URL
	if m := mergeRequestURLPathRE.FindStringSubmatch(u.Path); m != nil {
		resourceID, err := strconv.Atoi(m[2])
		if err != nil {
			return nil
		}

		// Use the captured group which contains the repository path
		u.Path = m[1]

		repo, err := glrepo.FromURL(u, defaultHostname)
		if err != nil {
			return nil
		}

		return &GitLabResourceMetadata{
			ID:   resourceID,
			Type: GitLabResourceMergeRequest,
			Repo: repo,
		}
	}

	// Try to match issue URL (duplicated from issueutils for consistency)
	// TODO: Unify this with issueutils.issueMetadataFromURL in a follow-up MR.
	if m := issueURLPathRE.FindStringSubmatch(u.Path); m != nil {
		resourceID, err := strconv.Atoi(m[2])
		if err != nil {
			return nil
		}

		// Use the captured group which contains the repository path
		u.Path = m[1]

		repo, err := glrepo.FromURL(u, defaultHostname)
		if err != nil {
			return nil
		}

		return &GitLabResourceMetadata{
			ID:   resourceID,
			Type: GitLabResourceIssue,
			Repo: repo,
		}
	}

	return nil
}

// ParseMergeRequestFromURL extracts merge request ID and repository from a GitLab URL
// Returns 0 and nil if the URL is not a valid merge request URL
func ParseMergeRequestFromURL(urlStr, defaultHostname string) (int, glrepo.Interface) {
	metadata := ParseGitLabURL(urlStr, defaultHostname)
	if metadata != nil && metadata.Type == GitLabResourceMergeRequest {
		return metadata.ID, metadata.Repo
	}
	return 0, nil
}
