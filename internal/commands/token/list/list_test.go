package list

import (
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
	cmd := NewCmdList(factory)
	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

var projectAccessTokenResponse = heredoc.Doc(`
	[
		{
			"id": 10179584,
			"user_id": 21973696,
			"name": "sadfsdfsdf",
			"scopes": [
				"api",
				"read_api"
			],
			"created_at": "2024-07-07T07:59:35.767Z",
			"description": "example description",
			"expires_at": "2024-08-06",
			"active": true,
			"revoked": false,
			"access_level": 10
		}
	]
`)

func TestListProjectAccessTokenAsText(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/access_tokens",
		httpmock.NewStringResponse(http.StatusOK, projectAccessTokenResponse))
	output, err := runCommand(t, fakeHTTP, "")
	if err != nil {
		t.Errorf("error running command `token list`: %v", err)
	}

	out := output.String()

	assert.Equal(t, heredoc.Doc(`
		ID       NAME       DESCRIPTION         ACCESS_LEVEL ACTIVE  REVOKED  CREATED_AT           EXPIRES_AT LAST_USED_AT SCOPES      
		10179584 sadfsdfsdf example description guest        true    false    2024-07-07T07:59:35Z 2024-08-06 -           api,read_api
	`), out)
	assert.Empty(t, output.Stderr())
}

func TestListProjectAccessTokenAsJSON(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/access_tokens",
		httpmock.NewStringResponse(http.StatusOK, projectAccessTokenResponse))
	output, err := runCommand(t, fakeHTTP, "--output json")
	if err != nil {
		t.Errorf("error running command `token list --output json`: %v", err)
	}
	assert.Empty(t, output.Stderr())
	assert.JSONEq(t, projectAccessTokenResponse, output.String())
}

var groupAccessTokenResponse = heredoc.Doc(`
		[
				{
					"id": 10179685,
					"user_id": 21973881,
					"name": "sadfsdfsdf",
					"scopes": [
						"read_api"
					],
					"created_at": "2024-07-07T08:41:16.287Z",
					"description": "example description",
					"expires_at": "2024-08-06",
					"active": true,
					"revoked": false,
					"access_level": 10
				}
		]
	`)

func TestListGroupAccessTokenAsText(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/groups/GROUP/access_tokens",
		httpmock.NewStringResponse(http.StatusOK, groupAccessTokenResponse))
	output, err := runCommand(t, fakeHTTP, "--group GROUP")
	if err != nil {
		t.Errorf("error running command `token list --group GROUP`: %v", err)
	}

	out := output.String()

	assert.Equal(t, heredoc.Doc(`
		ID       NAME       DESCRIPTION         ACCESS_LEVEL ACTIVE  REVOKED  CREATED_AT           EXPIRES_AT LAST_USED_AT SCOPES  
		10179685 sadfsdfsdf example description guest        true    false    2024-07-07T08:41:16Z 2024-08-06 -           read_api
	`), out)
	assert.Empty(t, output.Stderr())
}

func TestListGroupAccessTokenAsJSON(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/groups/GROUP/access_tokens",
		httpmock.NewStringResponse(http.StatusOK, groupAccessTokenResponse))

	output, err := runCommand(t, fakeHTTP, "--group GROUP --output json")
	if err != nil {
		t.Errorf("error running command `token list --group GROUP --output json`: %v", err)
	}
	assert.Empty(t, output.Stderr())
	assert.JSONEq(t, groupAccessTokenResponse, output.String())
}

var personalAccessTokenResponse = heredoc.Doc(`
			[
				{
					"id": 9860015,
					"name": "awsssm",
					"revoked": false,
					"created_at": "2024-05-29T07:25:56.846Z",
					"description": "example description 1",
					"scopes": [
						"api"
					],
					"user_id": 926857,
					"active": false,
					"expires_at": "2024-06-28"
				},
				{
					"id": 9860076,
					"name": "glab",
					"revoked": false,
					"created_at": "2024-05-29T07:34:14.044Z",
					"description": "example description 2",
					"scopes": [
						"api"
					],
					"user_id": 926857,
					"last_used_at": "2024-06-05T17:32:34.466Z",
					"active": false,
					"expires_at": "2024-06-28"
				},
				{
					"id": 10171440,
					"name": "api",
					"revoked": false,
					"created_at": "2024-07-05T10:02:37.182Z",
					"description": "example description 3",
					"scopes": [
						"api"
					],
					"user_id": 926857,
					"last_used_at": "2024-07-07T20:02:49.595Z",
					"active": true,
					"expires_at": "2024-08-04"
				}
			]
		`)

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

func TestListPersonalAccessTokenAsText(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/personal_access_tokens",
		httpmock.NewStringResponse(http.StatusOK, personalAccessTokenResponse))
	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/user",
		httpmock.NewStringResponse(http.StatusOK, userResponse))

	output, err := runCommand(t, fakeHTTP, "--user @me")
	if err != nil {
		t.Errorf("error running command `token list --user @me`: %v", err)
	}

	out := output.String()

	assert.Equal(t, heredoc.Doc(`
		ID       NAME   DESCRIPTION           ACCESS_LEVEL ACTIVE  REVOKED  CREATED_AT           EXPIRES_AT LAST_USED_AT         SCOPES 
		9860015  awsssm example description 1 -            false   false    2024-05-29T07:25:56Z 2024-06-28 -                    api    
		9860076  glab   example description 2 -            false   false    2024-05-29T07:34:14Z 2024-06-28 2024-06-05T17:32:34Z api    
		10171440 api    example description 3 -            true    false    2024-07-05T10:02:37Z 2024-08-04 2024-07-07T20:02:49Z api    
	`), out)
	assert.Empty(t, output.Stderr())
}

func TestListActivePersonalAccessTokenAsText(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/personal_access_tokens",
		httpmock.NewStringResponse(http.StatusOK, personalAccessTokenResponse))
	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/user",
		httpmock.NewStringResponse(http.StatusOK, userResponse))

	output, err := runCommand(t, fakeHTTP, "--user @me --active")
	if err != nil {
		t.Errorf("error running command `token list --user @me`: %v", err)
	}

	out := output.String()

	assert.Equal(t, heredoc.Doc(`
		ID       NAME  DESCRIPTION           ACCESS_LEVEL ACTIVE  REVOKED  CREATED_AT           EXPIRES_AT LAST_USED_AT         SCOPES 
		10171440 api   example description 3 -            true    false    2024-07-05T10:02:37Z 2024-08-04 2024-07-07T20:02:49Z api    
	`), out)
	assert.Empty(t, output.Stderr())
}

func TestListPersonalAccessTokenAsJSON(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/personal_access_tokens",
		httpmock.NewStringResponse(http.StatusOK, personalAccessTokenResponse))
	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/user",
		httpmock.NewStringResponse(http.StatusOK, userResponse))

	output, err := runCommand(t, fakeHTTP, "--user @me --output json")
	if err != nil {
		t.Errorf("error running command `token list --user @me`: %v", err)
	}

	assert.Empty(t, output.Stderr())
	assert.JSONEq(t, personalAccessTokenResponse, output.String())
}

var personalAccessTokenResponseWithoutExpiration = heredoc.Doc(`
			[
				{
					"id": 1,
					"name": "awsssm",
					"revoked": false,
					"created_at": "2024-05-29T07:25:56.846Z",
					"scopes": [
						"api"
					],
					"user_id": 926857,
					"active": false,
					"expires_at": null
				},
				{
					"id": 2,
					"name": "glab",
					"revoked": false,
					"created_at": "2024-05-29T07:34:14.044Z",
					"scopes": [
						"api"
					],
					"user_id": 926857,
					"last_used_at": "2024-06-05T17:32:34.466Z",
					"active": false,
					"expires_at": "2024-06-28"
				},
				{
					"id": 3,
					"name": "api",
					"revoked": false,
					"created_at": "2024-07-05T10:02:37.182Z",
					"scopes": [
						"api"
					],
					"user_id": 926857,
					"last_used_at": "2024-07-07T20:02:49.595Z",
					"active": true,
					"expires_at": "2024-08-04"
				}
			]
		`)

func TestListPersonalAccessTokenWithoutExpirationAsText(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/personal_access_tokens",
		httpmock.NewStringResponse(http.StatusOK, personalAccessTokenResponseWithoutExpiration))
	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/user",
		httpmock.NewStringResponse(http.StatusOK, userResponse))

	output, err := runCommand(t, fakeHTTP, "--user @me")
	if err != nil {
		t.Errorf("error running command `token list --user @me`: %v", err)
	}

	out := output.String()

	assert.Equal(t, heredoc.Doc(`
		ID  NAME   DESCRIPTION  ACCESS_LEVEL ACTIVE  REVOKED  CREATED_AT           EXPIRES_AT LAST_USED_AT         SCOPES 
		1   awsssm -            -            false   false    2024-05-29T07:25:56Z -          -                    api    
		2   glab   -            -            false   false    2024-05-29T07:34:14Z 2024-06-28 2024-06-05T17:32:34Z api    
		3   api    -            -            true    false    2024-07-05T10:02:37Z 2024-08-04 2024-07-07T20:02:49Z api    
	`), out)
	assert.Empty(t, output.Stderr())
}
