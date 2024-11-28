package create

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"

	"gitlab.com/gitlab-org/cli/commands/cmdtest"

	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper, isTTY bool, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(isTTY, "")

	factory := cmdtest.InitFactory(ios, rt)

	_, _ = factory.HttpClient()

	cmd := NewCmdCreate(factory)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestReleaseCreate(t *testing.T) {
	tests := []struct {
		name string
		cli  string

		expectedDescription string
		expectedTagMessage  string
	}{
		{
			name: "when a release is created",
			cli:  "0.0.1",
		},
		{
			name:                "when a release is created with a description",
			cli:                 `0.0.1 --notes "bugfix release"`,
			expectedDescription: "bugfix release",
		},
		{
			name:               "when a release is created with a tag message",
			cli:                `0.0.1 --tag-message "tag message"`,
			expectedTagMessage: "tag message",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

			fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/releases/0%2E0%2E1",
				httpmock.NewStringResponse(http.StatusNotFound, `{"message":"404 Not Found"}`))

			fakeHTTP.RegisterResponder(http.MethodPost, "/api/v4/projects/OWNER/REPO/releases",
				func(req *http.Request) (*http.Response, error) {
					rb, _ := io.ReadAll(req.Body)

					assert.Contains(t, string(rb), `"tag_name":"0.0.1"`)

					if tc.expectedDescription != "" {
						assert.Contains(t, string(rb), `"description":"bugfix release"`)
					}
					if tc.expectedTagMessage != "" {
						assert.Contains(t, string(rb), `"tag_message":"tag message"`)
					}
					resp, _ := httpmock.NewStringResponse(http.StatusCreated,
						`{
							"name": "test_release",
							"tag_name": "0.0.1",
							"description": "bugfix release",
							"created_at": "2023-01-19T02:58:32.622Z",
							"released_at": "2023-01-19T02:58:32.622Z",
							"upcoming_release": false,
							"tag_path": "/OWNER/REPO/-/tags/0.0.1",
							"_links": {
								"self": "https://gitlab.com/OWNER/REPO/-/releases/0.0.1"
							}
						}`)(req)
					return resp, nil
				},
			)

			output, err := runCommand(fakeHTTP, false, tc.cli)

			if assert.NoErrorf(t, err, "error running command `create %s`: %v", tc.cli, err) {
				assert.Contains(t, output.Stderr(), `• Validating tag 0.0.1

• Creating or updating release repo=OWNER/REPO tag=0.0.1
✓ Release created:	url=https://gitlab.com/OWNER/REPO/-/releases/0.0.1
✓ Release succeeded after`)
				assert.Empty(t, output.String())
			}
		})
	}
}

func TestReleaseCreateWithFiles(t *testing.T) {
	tests := []struct {
		name string
		cli  string

		wantType     bool
		expectedType string
		expectedOut  string
	}{
		{
			name: "when a release is created and a file is uploaded using filename only",
			cli:  "0.0.1 testdata/test_file.txt",

			wantType: false,
		},
		{
			name: "when a release is created and a filename is uploaded with display name and type",
			cli:  "0.0.1 testdata/test_file.txt#test_file#other",

			wantType:     true,
			expectedType: `"link_type":"other"`,
		},
		{
			name: "when a release is created and a filename is uploaded with a type",
			cli:  "0.0.1 testdata/test_file.txt##package",

			wantType:     true,
			expectedType: `"link_type":"package"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

			fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/releases/0%2E0%2E1",
				httpmock.NewStringResponse(http.StatusNotFound, `{"message":"404 Not Found"}`))

			fakeHTTP.RegisterResponder(http.MethodPost, "/api/v4/projects/OWNER/REPO/releases",
				func(req *http.Request) (*http.Response, error) {
					rb, _ := io.ReadAll(req.Body)

					assert.Contains(t, string(rb), `"tag_name":"0.0.1"`)

					resp, _ := httpmock.NewStringResponse(http.StatusCreated,
						`{
							"name": "test_release",
							"tag_name": "0.0.1",
							"description": null,
							"created_at": "2023-01-19T02:58:32.622Z",
							"released_at": "2023-01-19T02:58:32.622Z",
							"upcoming_release": false,
							"tag_path": "/OWNER/REPO/-/tags/0.0.1",
							"_links": {
								"self": "https://gitlab.com/OWNER/REPO/-/releases/0.0.1"
							}
						}`)(req)
					return resp, nil
				},
			)

			fakeHTTP.RegisterResponder(http.MethodPost, "/api/v4/projects/OWNER/REPO/uploads",
				httpmock.NewStringResponse(http.StatusCreated,
					`{
							  "alt": "test_file",
							  "url": "/uploads/66dbcd21ec5d24ed6ea225176098d52b/testdata/test_file.txt",
							  "full_path": "/namespace1/project1/uploads/66dbcd21ec5d24ed6ea225176098d52b/testdata/test_file.txt",
							  "markdown": "![test_file](/uploads/66dbcd21ec5d24ed6ea225176098d52b/testdata/test_file.txt)"
							}`))

			fakeHTTP.RegisterResponder(http.MethodPost, "/api/v4/projects/OWNER/REPO/releases/0%2E0%2E1/assets/links",
				func(req *http.Request) (*http.Response, error) {
					rb, _ := io.ReadAll(req.Body)

					if tc.wantType {
						assert.Contains(t, string(rb), tc.expectedType)
					} else {
						assert.NotContains(t, string(rb), "link_type")
					}

					resp, _ := httpmock.NewStringResponse(http.StatusCreated, `{
						"id":2,
						"name":"test_file.txt",
						"url":"https://gitlab.example.com/mynamespace/hello/-/jobs/688/artifacts/raw/testdata/test_file.txt",
						"direct_asset_url":"https://gitlab.example.com/mynamespace/hello/-/releases/0.0.1/downloads/testdata/test_file.txt",
						"link_type":"other"
						}`)(req)
					return resp, nil
				},
			)

			output, err := runCommand(fakeHTTP, false, tc.cli)

			if assert.NoErrorf(t, err, "error running command `create %s`: %v", tc.cli, err) {
				assert.Contains(t, output.Stderr(), `• Validating tag 0.0.1

• Creating or updating release repo=OWNER/REPO tag=0.0.1
✓ Release created:	url=https://gitlab.com/OWNER/REPO/-/releases/0.0.1
• Uploading release assets repo=OWNER/REPO tag=0.0.1
• Uploading to release	file=testdata/test_file.txt name=test_file.txt
✓ Release succeeded after`)
				assert.Empty(t, output.String())
			}
		})
	}
}

func TestReleaseCreate_WithAssetsLinksJSON(t *testing.T) {
	tests := []struct {
		name           string
		cli            string
		expectedOutput string
	}{
		{
			name: "with direct_asset_path",
			cli:  `0.0.1 --assets-links='[{"name": "any-name", "url": "https://example.com/any-asset-url", "direct_asset_path": "/any-path"}]'`,
			expectedOutput: `• Validating tag 0.0.1

• Creating or updating release repo=OWNER/REPO tag=0.0.1
✓ Release created:	url=https://gitlab.com/OWNER/REPO/-/releases/0.0.1
• Uploading release assets repo=OWNER/REPO tag=0.0.1
✓ Added release asset	name=any-name url=https://gitlab.example.com/OWNER/REPO/releases/0.0.1/downloads/any-path
✓ Release succeeded after`,
		},
		{
			name: "with filepath aliased to direct_asset_path",
			cli:  `0.0.1 --assets-links='[{"name": "any-name", "url": "https://example.com/any-asset-url", "filepath": "/any-path"}]'`,
			expectedOutput: `• Validating tag 0.0.1

• Creating or updating release repo=OWNER/REPO tag=0.0.1
✓ Release created:	url=https://gitlab.com/OWNER/REPO/-/releases/0.0.1
• Uploading release assets repo=OWNER/REPO tag=0.0.1
✓ Added release asset	name=any-name url=https://gitlab.example.com/OWNER/REPO/releases/0.0.1/downloads/any-path
	! Aliased deprecated ` + "`filepath`" + ` field to ` + "`direct_asset_path`" + `. Replace ` + "`filepath`" + ` with ` + "`direct_asset_path`" + `	name=any-name
✓ Release succeeded after`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

			fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/releases/0%2E0%2E1",
				httpmock.NewStringResponse(http.StatusNotFound, `{"message":"404 Not Found"}`))

			fakeHTTP.RegisterResponder(http.MethodPost, "/api/v4/projects/OWNER/REPO/releases",
				func(req *http.Request) (*http.Response, error) {
					rb, _ := io.ReadAll(req.Body)

					assert.NotContains(t, string(rb), `"direct_asset_path":`)
					assert.NotContains(t, string(rb), `"filepath":`)

					resp, _ := httpmock.NewStringResponse(http.StatusCreated, `
						{
							"name": "test_release",
							"tag_name": "0.0.1",
							"description": null,
							"created_at": "2023-01-19T02:58:32.622Z",
							"released_at": "2023-01-19T02:58:32.622Z",
							"upcoming_release": false,
							"tag_path": "/OWNER/REPO/-/tags/0.0.1",
							"_links": {
								"self": "https://gitlab.com/OWNER/REPO/-/releases/0.0.1"
							}
						}
					`)(req)
					return resp, nil
				},
			)

			fakeHTTP.RegisterResponder(http.MethodPost, "/api/v4/projects/OWNER/REPO/releases/0%2E0%2E1/assets/links",
				func(req *http.Request) (*http.Response, error) {
					rb, _ := io.ReadAll(req.Body)

					assert.Contains(t, string(rb), `"direct_asset_path":"/any-path"`)
					assert.NotContains(t, string(rb), `"filepath":`)

					resp, _ := httpmock.NewStringResponse(http.StatusCreated, `
						{
							"id":1,
							"name":"any-name",
							"url":"https://example.com/any-asset-url",
							"direct_asset_url":"https://gitlab.example.com/OWNER/REPO/releases/0.0.1/downloads/any-path",
							"link_type":"other"
						}
					`)(req)
					return resp, nil
				},
			)

			output, err := runCommand(fakeHTTP, false, tt.cli)

			if assert.NoErrorf(t, err, "error running command `create %s`: %v", tt.cli, err) {
				assert.Contains(t, output.Stderr(), tt.expectedOutput)
				assert.Empty(t, output.String())
			}
		})
	}
}

func TestReleaseCreateWithPublishToCatalog(t *testing.T) {
	tests := []struct {
		name string
		cli  string

		wantOutput string
		wantBody   string
		wantErr    bool
		errMsg     string
	}{
		{
			name: "with version",
			cli:  "0.0.1 --publish-to-catalog",
			wantBody: `{
				"version": "0.0.1",
				"metadata": {
					"components": [
						{
							"component_type": "template",
							"name": "component-1",
							"spec": {
								"inputs": {
									"compiler": {
										"default": "gcc"
									}
								}
							}
						},
						{
							"component_type": "template",
							"name": "component-2",
							"spec": null
						},
						{
							"component_type": "template",
							"name": "component-3",
							"spec": {
								"inputs": {
									"test_framework": {
										"default": "unittest"
									}
								}
							}
						}
					]
				}
			}`,
			wantOutput: `• Publishing release tag=0.0.1 to the GitLab CI/CD catalog for repo=OWNER/REPO...
✓ Release published: url=https://gitlab.example.com/explore/catalog/my-namespace/my-component-project`,
		},
	}

	originalWd, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(filepath.Join(originalWd, "catalog", "testdata", "test-repo"))
	require.NoError(t, err)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

			fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/releases/0%2E0%2E1",
				httpmock.NewStringResponse(http.StatusNotFound, `{"message":"404 Not Found"}`))

			fakeHTTP.RegisterResponder(http.MethodPost, "/api/v4/projects/OWNER/REPO/releases",
				func(req *http.Request) (*http.Response, error) {
					resp, _ := httpmock.NewStringResponse(http.StatusCreated,
						`{
							"name": "test_release",
							"tag_name": "0.0.1",
							"description": "bugfix release",
							"created_at": "2023-01-19T02:58:32.622Z",
							"released_at": "2023-01-19T02:58:32.622Z",
							"upcoming_release": false,
							"tag_path": "/OWNER/REPO/-/tags/0.0.1",
							"_links": {
								"self": "https://gitlab.com/OWNER/REPO/-/releases/0.0.1"
							}
						}`)(req)
					return resp, nil
				},
			)

			if tc.wantBody != "" {
				fakeHTTP.RegisterResponder(http.MethodPost, "/api/v4/projects/OWNER/REPO/catalog/publish",
					func(req *http.Request) (*http.Response, error) {
						body, _ := io.ReadAll(req.Body)

						assert.JSONEq(t, tc.wantBody, string(body))

						response := httpmock.NewJSONResponse(http.StatusOK, map[string]interface{}{
							"catalog_url": "https://gitlab.example.com/explore/catalog/my-namespace/my-component-project",
						})

						return response(req)
					},
				)
			}

			output, err := runCommand(fakeHTTP, false, tc.cli)

			if tc.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tc.errMsg, err.Error())
			} else {
				assert.NoError(t, err)
				assert.Contains(t, output.Stderr(), tc.wantOutput)
			}
		})
	}

	err = os.Chdir(originalWd)
	require.NoError(t, err)
}
