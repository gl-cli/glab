package revoke

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(t *testing.T, rt http.RoundTripper, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams(cmdtest.WithTestIOStreamsAsTTY(true))
	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithApiClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", glinstance.DefaultHostname)),
	)
	cmd := NewCmdRevoke(factory)
	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

var userResponse = heredoc.Doc(`
	{
		"id": 1,
		"username": "johndoe",
		"name": "John Doe",
		"state": "active",
		"locked": false,
		"avatar_url": "https://secure.gravatar.com/avatar/johndoe?s=80&d=identicon",
		"web_url": "https://gitlab.com/johndoe",
		"created_at": "2017-01-05T08:36:01.368Z",
		"bio": "",
		"location": "",
		"public_email": "",
		"skype": "",
		"linkedin": "",
		"twitter": "",
		"discord": "",
		"website_url": "",
		"organization": "",
		"job_title": "",
		"pronouns": null,
		"bot": false,
		"work_information": null,
		"local_time": null,
		"last_sign_in_at": "2024-07-07T06:57:16.562Z",
		"confirmed_at": "2017-01-05T08:36:24.701Z",
		"last_activity_on": "2024-07-07",
		"email": "john.doe@acme.com",
		"theme_id": null,
		"color_scheme_id": 1,
		"projects_limit": 100000,
		"current_sign_in_at": "2024-07-07T07:57:57.858Z",
		"identities": [
			{
				"provider": "google_oauth2",
				"extern_uid": "102139960402025821780",
				"saml_provider_id": null
			}
		],
		"can_create_group": true,
		"can_create_project": true,
		"two_factor_enabled": true,
		"external": false,
		"private_profile": false,
		"commit_email": "john.doe@acme.com",
		"shared_runners_minutes_limit": 2000,
		"extra_shared_runners_minutes_limit": null,
		"scim_identities": []
	}
`)

var personalAccessTokenResponse = heredoc.Doc(`
	{
		"id": 10183862,
		"name": "my-pat",
		"revoked": false,
		"created_at": "2024-07-08T01:23:04.311Z",
		"description": "",
		"scopes": [
			"k8s_proxy"
		],
		"user_id": 926857,
		"active": true,
		"expires_at": "2024-08-07",
		"token": "glpat-jRHatYQ8Fs77771111ps"
	}`)

func TestRevokePersonalAccessTokenAsJSON(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/user",
		httpmock.NewStringResponse(http.StatusOK, userResponse))
	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/personal_access_tokens",
		httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf("[%s]", personalAccessTokenResponse)))
	fakeHTTP.RegisterResponder(http.MethodDelete, "/api/v4/personal_access_tokens/10183862",
		httpmock.NewStringResponse(http.StatusOK, personalAccessTokenResponse))

	output, err := runCommand(t, fakeHTTP, "--user @me --output json my-pat")
	if err != nil {
		t.Error(err)
		return
	}
	var actual map[string]any
	var expect map[string]any
	if err := json.Unmarshal([]byte(personalAccessTokenResponse), &expect); err != nil {
		t.Error(err)
	}
	expect["revoked"] = true
	if err := json.Unmarshal([]byte(output.String()), &actual); err != nil {
		t.Error(err)
	}
	assert.Equal(t, expect, actual)
	assert.Empty(t, output.Stderr())
}

func TestRevokePersonalAccessTokenAsText(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/user",
		httpmock.NewStringResponse(http.StatusOK, userResponse))
	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/personal_access_tokens",
		httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf("[%s]", personalAccessTokenResponse)))
	fakeHTTP.RegisterResponder(http.MethodDelete, "/api/v4/personal_access_tokens/10183862",
		httpmock.NewStringResponse(http.StatusOK, personalAccessTokenResponse))

	output, err := runCommand(t, fakeHTTP, "--user @me my-pat")
	if err != nil {
		t.Error(err)
		return
	}
	assert.Equal(t, "revoked @me my-pat 10183862", output.String())
}

var groupAccessTokenResponse = heredoc.Doc(`
	{
		"id": 10190772,
		"user_id": 21989300,
		"name": "my-group-token",
		"scopes": [
			"read_registry",
			"read_repository"
		],
		"created_at": "2024-07-08T17:33:34.829Z",
		"description": "",
		"expires_at": "2024-08-07",
		"active": true,
		"revoked": false,
		"token": "glpat-yz2791KMU-xxxxxxxxx",
		"access_level": 30
	}`)

func TestRevokeGroupAccessTokenAsJSON(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/groups/GROUP/access_tokens",
		httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf("[%s]", groupAccessTokenResponse)))
	fakeHTTP.RegisterResponder(http.MethodDelete, "/api/v4/groups/GROUP/access_tokens/10190772",
		httpmock.NewStringResponse(http.StatusOK, groupAccessTokenResponse))

	output, err := runCommand(t, fakeHTTP, "--group GROUP my-group-token --output json")
	if err != nil {
		t.Error(err)
		return
	}

	var expect map[string]any
	var actual map[string]any

	if err := json.Unmarshal([]byte(groupAccessTokenResponse), &expect); err != nil {
		t.Error(err)
	}
	expect["revoked"] = true
	if err := json.Unmarshal([]byte(output.String()), &actual); err != nil {
		t.Error(err)
	}
	assert.Equal(t, expect, actual)
	assert.Empty(t, output.Stderr())
}

func TestRevokeGroupAccessTokenAsText(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/groups/GROUP/access_tokens",
		httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf("[%s]", groupAccessTokenResponse)))
	fakeHTTP.RegisterResponder(http.MethodDelete, "/api/v4/groups/GROUP/access_tokens/10190772",
		httpmock.NewStringResponse(http.StatusOK, groupAccessTokenResponse))

	output, err := runCommand(t, fakeHTTP, "--group GROUP my-group-token")
	if err != nil {
		t.Error(err)
		return
	}

	assert.Equal(t, "revoked my-group-token 10190772", output.String())
}

var projectAccessTokenResponse = heredoc.Doc(`
	{
		"id": 10191548,
		"user_id": 21990679,
		"name": "my-project-token",
		"scopes": [
			"api",
			"read_repository"
		],
		"created_at": "2024-07-08T19:47:14.727Z",
		"description": "",
		"expires_at": "2024-08-07",
		"active": true,
		"revoked": false,
		"token": "glpat-dfsdfjksjdfslkdfjsd",
		"access_level": 30
	}`)

func TestRevokeProjectAccessTokenAsJSON(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/access_tokens",
		httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf("[%s]", projectAccessTokenResponse)))
	fakeHTTP.RegisterResponder(http.MethodDelete, "/api/v4/projects/OWNER/REPO/access_tokens/10191548",
		httpmock.NewStringResponse(http.StatusOK, projectAccessTokenResponse))

	output, err := runCommand(t, fakeHTTP, "--output json my-project-token")
	if err != nil {
		t.Error(err)
		return
	}

	var expect map[string]any
	var actual map[string]any

	if err := json.Unmarshal([]byte(projectAccessTokenResponse), &expect); err != nil {
		t.Error(err)
	}
	expect["revoked"] = true
	if err := json.Unmarshal([]byte(output.String()), &actual); err != nil {
		t.Error(err)
	}
	assert.Equal(t, expect, actual)
	assert.Empty(t, output.Stderr())
}

func TestRevokeProjectAccessTokenAsText(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/access_tokens",
		httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf("[%s]", projectAccessTokenResponse)))
	fakeHTTP.RegisterResponder(http.MethodDelete, "/api/v4/projects/OWNER/REPO/access_tokens/10191548",
		httpmock.NewStringResponse(http.StatusOK, projectAccessTokenResponse))

	output, err := runCommand(t, fakeHTTP, "my-project-token")
	if err != nil {
		t.Error(err)
		return
	}

	assert.Equal(t, "revoked my-project-token 10191548", output.String())
}
