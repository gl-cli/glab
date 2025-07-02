package catalog

import (
	"fmt"
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
	cmd := NewCmdPublishCatalog(factory)
	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestPublishCatalog(t *testing.T) {
	tests := []struct {
		name           string
		tagName        string
		isValidTagName bool

		wantOutput string
		wantBody   string
		wantErr    bool
		errMsg     string
	}{
		{
			name:           "valid tag",
			tagName:        "0.0.1",
			isValidTagName: true,
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
		{
			name:    "missing tag",
			wantErr: true,
			errMsg:  "accepts 1 arg(s), received 0",
		},
		{
			name:           "invalid tag",
			tagName:        "6.6.6",
			isValidTagName: false,
			wantErr:        true,
			errMsg:         "Invalid tag 6.6.6.",
		},
	}

	originalWd, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(filepath.Join(originalWd, "testdata", "test-repo"))
	require.NoError(t, err)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

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

			if tc.tagName != "" {
				tagUrl := fmt.Sprintf("/api/v4/projects/OWNER/REPO/repository/tags/%s", tc.tagName)
				fakeHTTP.RegisterResponder(http.MethodGet, tagUrl,
					func(req *http.Request) (*http.Response, error) {
						var response httpmock.Responder
						if tc.isValidTagName {
							response = httpmock.NewJSONResponse(http.StatusOK, map[string]any{
								"name": tc.tagName,
							})
						} else {
							response = httpmock.NewJSONResponse(http.StatusNotFound, map[string]any{
								"message": "404 Tag Not Found",
							})
						}
						return response(req)
					},
				)
			}

			output, err := runCommand(t, fakeHTTP, tc.tagName)

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
