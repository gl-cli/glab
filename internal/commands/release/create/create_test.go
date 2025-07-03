package create

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"

	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"

	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(t *testing.T, rt http.RoundTripper, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams()

	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithGitLabClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", glinstance.DefaultHostname).Lab()),
	)

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

			output, err := runCommand(t, fakeHTTP, tc.cli)

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

			output, err := runCommand(t, fakeHTTP, tc.cli)

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

			output, err := runCommand(t, fakeHTTP, tt.cli)

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

	err = os.Chdir(filepath.Join(originalWd, "..", "..", "project", "publish", "catalog", "testdata", "test-repo"))
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

						response := httpmock.NewJSONResponse(http.StatusOK, map[string]any{
							"catalog_url": "https://gitlab.example.com/explore/catalog/my-namespace/my-component-project",
						})

						return response(req)
					},
				)
			}

			output, err := runCommand(t, fakeHTTP, tc.cli)

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

func TestReleaseCreate_NoUpdate(t *testing.T) {
	tests := []struct {
		name    string
		cli     string
		exists  bool
		wantErr bool
	}{
		{
			name:    "when release doesn't exist with --no-update flag",
			cli:     "0.0.1 --no-update",
			exists:  false,
			wantErr: false,
		},
		{
			name:    "when release exists with --no-update flag",
			cli:     "0.0.1 --no-update",
			exists:  true,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

			if tc.exists {
				fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/releases/0%2E0%2E1",
					httpmock.NewStringResponse(http.StatusOK, `{
						"name": "test_release",
						"tag_name": "0.0.1"
					}`))
			} else {
				fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/releases/0%2E0%2E1",
					httpmock.NewStringResponse(http.StatusNotFound, `{"message":"404 Not Found"}`))

				fakeHTTP.RegisterResponder(http.MethodPost, "/api/v4/projects/OWNER/REPO/releases",
					httpmock.NewStringResponse(http.StatusCreated, `{
						"name": "test_release",
						"tag_name": "0.0.1"
					}`))
			}

			output, err := runCommand(t, fakeHTTP, tc.cli)

			if tc.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "release for tag \"0.0.1\" already exists and --no-update flag was specified")
			} else {
				assert.NoError(t, err)
				assert.Contains(t, output.Stderr(), "Release created:")
			}
		})
	}
}

func TestReleaseCreate_MilestoneClosing(t *testing.T) {
	tests := []struct {
		name           string
		cli            string
		extraHttpStubs func(*httpmock.Mocker)
		wantOutput     string
		wantErr        bool
	}{
		{
			name: "successfully closes milestone after release",
			cli:  "0.0.1 --milestone 'v1.0'",
			extraHttpStubs: func(fakeHTTP *httpmock.Mocker) {
				fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/milestones?title=v1.0",
					httpmock.NewStringResponse(http.StatusOK, `[{
						"id": 1,
						"iid": 1,
						"title": "v1.0",
						"state": "active"
					}]`))

				fakeHTTP.RegisterResponder(http.MethodPut, "/api/v4/projects/OWNER/REPO/milestones/1",
					httpmock.NewStringResponse(http.StatusOK, `{
						"id": 1,
						"iid": 1,
						"title": "v1.0",
						"state": "closed"
					}`))
			},
			wantOutput: `• Validating tag 0.0.1

• Creating or updating release repo=OWNER/REPO tag=0.0.1
✓ Release created:	url=https://gitlab.com/OWNER/REPO/-/releases/0.0.1
✓ Closed milestone "v1.0"`,
			wantErr: false,
		},
		{
			name:           "skips milestone closing when --no-close-milestone is set",
			cli:            "0.0.1 --milestone 'v1.0' --no-close-milestone",
			extraHttpStubs: nil,
			wantOutput: `• Validating tag 0.0.1

• Creating or updating release repo=OWNER/REPO tag=0.0.1
✓ Release created:	url=https://gitlab.com/OWNER/REPO/-/releases/0.0.1
✓ Skipping closing milestones`,
			wantErr: false,
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
				httpmock.NewStringResponse(http.StatusCreated, `{
					"name": "0.0.1",
					"tag_name": "0.0.1",
					"description": "Release with milestone",
					"_links": {
						"self": "https://gitlab.com/OWNER/REPO/-/releases/0.0.1"
					}
				}`))

			if tt.extraHttpStubs != nil {
				tt.extraHttpStubs(fakeHTTP)
			}

			output, err := runCommand(t, fakeHTTP, tt.cli)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, output.Stderr(), tt.wantOutput)
			}
		})
	}
}

func TestReleaseCreate_ExperimentalNotes(t *testing.T) {
	tests := []struct {
		name                string
		cli                 string
		files               map[string]string
		wantErr             bool
		errMsg              string
		setupHTTPStubs      bool
		expectedDescription string
	}{
		{
			name:           "when experimental notes is used with notes flag",
			cli:            `0.0.1 --experimental-notes-text-or-file "test.md" --notes "test"`,
			wantErr:        true,
			errMsg:         "if any flags in the group [experimental-notes-text-or-file notes] are set none of the others can be; [experimental-notes-text-or-file notes] were all set",
			setupHTTPStubs: false,
		},
		{
			name:           "when experimental notes is used with notes-file flag",
			cli:            `0.0.1 --experimental-notes-text-or-file "test.md" --notes-file "other.md"`,
			wantErr:        true,
			errMsg:         "if any flags in the group [experimental-notes-text-or-file notes-file] are set none of the others can be; [experimental-notes-text-or-file notes-file] were all set",
			setupHTTPStubs: false,
		},
		{
			name: "when experimental notes points to existing file",
			cli:  `0.0.1 --experimental-notes-text-or-file "test.md"`,
			files: map[string]string{
				"test.md": "# Test Release\nThis is a test release.",
			},
			setupHTTPStubs:      true,
			expectedDescription: "# Test Release\nThis is a test release.",
		},
		{
			name:                "when experimental notes has non-existent file and falls back to text",
			cli:                 `0.0.1 --experimental-notes-text-or-file "This is plain text"`,
			setupHTTPStubs:      true,
			expectedDescription: "This is plain text",
		},
		{
			name:                "when experimental notes contains spaces, treats as text",
			cli:                 `0.0.1 --experimental-notes-text-or-file "This contains spaces.md"`,
			setupHTTPStubs:      true,
			expectedDescription: "This contains spaces.md",
		},
		{
			name:                "when experimental notes has leading/trailing spaces",
			cli:                 `0.0.1 --experimental-notes-text-or-file " notes.md "`,
			setupHTTPStubs:      true,
			expectedDescription: " notes.md ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			err := os.Chdir(tempDir)
			require.NoError(t, err)

			for filename, content := range tt.files {
				err := os.WriteFile(filename, []byte(content), 0o600)
				require.NoError(t, err)
			}

			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

			if tt.setupHTTPStubs {
				fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/releases/0%2E0%2E1",
					httpmock.NewStringResponse(http.StatusNotFound, `{"message":"404 Not Found"}`))

				fakeHTTP.RegisterResponder(http.MethodPost, "/api/v4/projects/OWNER/REPO/releases",
					func(req *http.Request) (*http.Response, error) {
						var reqBody map[string]any
						err := json.NewDecoder(req.Body).Decode(&reqBody)
						require.NoError(t, err)

						assert.Equal(t, tt.expectedDescription, reqBody["description"])

						resp, _ := httpmock.NewStringResponse(http.StatusCreated, `{
							"name": "test_release",
							"tag_name": "0.0.1",
							"_links": {
								"self": "https://gitlab.com/OWNER/REPO/-/releases/0.0.1"
							}
						}`)(req)
						return resp, nil
					})
			}

			output, err := runCommand(t, fakeHTTP, tt.cli)

			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
			} else {
				require.NoErrorf(t, err, "error running command `create %s`: %v", tt.cli, err)
				assert.Contains(t, output.Stderr(), "✓ Release created:")
				assert.Empty(t, output.String())
			}
		})
	}
}
