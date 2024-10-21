package rotate

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(true, "")
	factory := cmdtest.InitFactory(ios, rt)

	// TODO: shouldn't be there but the stub doesn't work without it
	_, _ = factory.HttpClient()

	cmd := NewCmdRotate(factory, nil)

	if out, err := cmdtest.ExecuteCommand(cmd, cli, stdout, stderr); err != nil {
		return nil, fmt.Errorf("error running command %s '%s', %s", cmd.Aliases[0], cli, err)
	} else {
		return out, nil
	}
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
			    "scopes": [
			      "k8s_proxy"
			    ],
			    "user_id": 926857,
			    "active": true,
			    "expires_at": "2024-08-07",
			    "token": "glpat-jRHatYQ8Fs77771111ps"
			  }
		`)

func TestRotatePersonalAccessTokenAsJSON(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/user",
		httpmock.NewStringResponse(http.StatusOK, userResponse))
	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/personal_access_tokens",
		httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf("[%s]", personalAccessTokenResponse)))
	fakeHTTP.RegisterResponder(http.MethodPost, "/api/v4/personal_access_tokens/10183862/rotate",
		httpmock.NewStringResponse(http.StatusOK, personalAccessTokenResponse))

	output, err := runCommand(fakeHTTP, "--user @me --output json my-pat")
	if err != nil {
		t.Error(err)
		return
	}

	var expect interface{}
	var actual interface{}

	if err := json.Unmarshal([]byte(personalAccessTokenResponse), &expect); err != nil {
		t.Error(err)
	}

	if err := json.Unmarshal([]byte(output.String()), &actual); err != nil {
		t.Error(err)
	}
	assert.Equal(t, expect, actual)
	assert.Empty(t, output.Stderr())
}

func TestRotatePersonalAccessTokenAsText(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/user",
		httpmock.NewStringResponse(http.StatusOK, userResponse))
	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/personal_access_tokens",
		httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf("[%s]", personalAccessTokenResponse)))
	fakeHTTP.RegisterResponder(http.MethodPost, "/api/v4/personal_access_tokens/10183862/rotate",
		httpmock.NewStringResponse(http.StatusOK, personalAccessTokenResponse))

	output, err := runCommand(fakeHTTP, "--user @me my-pat")
	if err != nil {
		t.Error(err)
		return
	}
	assert.Equal(t, "glpat-jRHatYQ8Fs77771111ps\n", output.String())
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
    "expires_at": "2024-08-07",
    "last_used_at": null,
    "active": true,
    "revoked": false,
    "token": "glpat-yz2791KMU-xxxxxxxxx",
    "access_level": 30
  }`)

func TestRotateGroupAccessTokenAsJSON(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/groups/GROUP/access_tokens",
		httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf("[%s]", groupAccessTokenResponse)))
	fakeHTTP.RegisterResponder(http.MethodPost, "/api/v4/groups/GROUP/access_tokens/10190772/rotate",
		httpmock.NewStringResponse(http.StatusOK, groupAccessTokenResponse))

	output, err := runCommand(fakeHTTP, "--group GROUP my-group-token --output json")
	if err != nil {
		t.Error(err)
		return
	}

	var expect interface{}
	var actual interface{}

	if err := json.Unmarshal([]byte(groupAccessTokenResponse), &expect); err != nil {
		t.Error(err)
	}

	if err := json.Unmarshal([]byte(output.String()), &actual); err != nil {
		t.Error(err)
	}
	assert.Equal(t, expect, actual)
	assert.Empty(t, output.Stderr())
}

func TestRotateGroupAccessTokenAsText(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/groups/GROUP/access_tokens",
		httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf("[%s]", groupAccessTokenResponse)))
	fakeHTTP.RegisterResponder(http.MethodPost, "/api/v4/groups/GROUP/access_tokens/10190772/rotate",
		httpmock.NewStringResponse(http.StatusOK, groupAccessTokenResponse))

	output, err := runCommand(fakeHTTP, "--group GROUP my-group-token")
	if err != nil {
		t.Error(err)
		return
	}

	assert.Equal(t, "glpat-yz2791KMU-xxxxxxxxx\n", output.String())
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
		"last_used_at": null,
		"expires_at": "2024-08-07",
		"active": true,
		"revoked": false,
		"token": "glpat-dfsdfjksjdfslkdfjsd",
		"access_level": 30
	}`)

func TestRotateProjectAccessTokenAsJSON(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/access_tokens",
		httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf("[%s]", projectAccessTokenResponse)))
	fakeHTTP.RegisterResponder(http.MethodPost, "/api/v4/projects/OWNER/REPO/access_tokens/10191548/rotate",
		httpmock.NewStringResponse(http.StatusOK, projectAccessTokenResponse))

	output, err := runCommand(fakeHTTP, "--output json my-project-token")
	if err != nil {
		t.Error(err)
		return
	}

	var expect interface{}
	var actual interface{}

	if err := json.Unmarshal([]byte(projectAccessTokenResponse), &expect); err != nil {
		t.Error(err)
	}

	if err := json.Unmarshal([]byte(output.String()), &actual); err != nil {
		t.Error(err)
	}
	assert.Equal(t, expect, actual)
	assert.Empty(t, output.Stderr())
}

func TestRotateProjectAccessTokenAsText(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/access_tokens",
		httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf("[%s]", projectAccessTokenResponse)))
	fakeHTTP.RegisterResponder(http.MethodPost, "/api/v4/projects/OWNER/REPO/access_tokens/10191548/rotate",
		httpmock.NewStringResponse(http.StatusOK, projectAccessTokenResponse))

	output, err := runCommand(fakeHTTP, "my-project-token")
	if err != nil {
		t.Error(err)
		return
	}

	assert.Equal(t, "glpat-dfsdfjksjdfslkdfjsd\n", output.String())
}
