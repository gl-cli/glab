package upload

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"

	"gitlab.com/gitlab-org/cli/commands/cmdtest"

	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper, isTTY bool, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(isTTY, "")

	factory := cmdtest.InitFactory(ios, rt)

	_, _ = factory.HttpClient()

	cmd := NewCmdUpload(factory)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestReleaseUpload(t *testing.T) {
	tests := []struct {
		name string
		cli  string

		wantType     bool
		expectedType string
		expectedOut  string
	}{
		{
			name: "when a file is uploaded using filename only, and does not send a link_type",
			cli:  "0.0.1 testdata/test_file.txt",

			wantType: false,
		},
		{
			name: "when a file is uploaded using a filename, display name and type",
			cli:  "0.0.1 testdata/test_file.txt#test_file#other",

			wantType:     true,
			expectedType: `"link_type":"other"`,
		},
		{
			name: "when a file is uploaded using a filename and type only",
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
				httpmock.NewStringResponse(http.StatusOK,
					`{
							"name": "test1",
							"tag_name": "0.0.1",
							"description": null,
							"created_at": "2023-01-19T02:58:32.622Z",
							"released_at": "2023-01-19T02:58:32.622Z",
							"upcoming_release": false,
							"tag_path": "/OWNER/REPO/-/tags/0.0.1"
						}`))

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

			if assert.NoErrorf(t, err, "error running command `release upload %s`: %v", tc.cli, err) {
				assert.Contains(t, output.Stderr(), `• Validating tag repo=OWNER/REPO tag=0.0.1
• Uploading release assets repo=OWNER/REPO tag=0.0.1
• Uploading to release	file=testdata/test_file.txt name=test_file.txt
✓ Upload succeeded after`)
				assert.Empty(t, output.String())
			}
		})
	}
}

func TestReleaseUpload_WithAssetsLinksJSON(t *testing.T) {
	tests := []struct {
		name           string
		cli            string
		expectedOutput string
	}{
		{
			name: "with direct_asset_path",
			cli:  `0.0.1 --assets-links='[{"name": "any-name", "url": "https://example.com/any-asset-url", "direct_asset_path": "/any-path"}]'`,
			expectedOutput: `• Validating tag repo=OWNER/REPO tag=0.0.1
• Uploading release assets repo=OWNER/REPO tag=0.0.1
✓ Added release asset	name=any-name url=https://gitlab.example.com/OWNER/REPO/releases/0.0.1/downloads/any-path
✓ Upload succeeded after`,
		},
		{
			name: "with filepath aliased to direct_asset_path",
			cli:  `0.0.1 --assets-links='[{"name": "any-name", "url": "https://example.com/any-asset-url", "filepath": "/any-path"}]'`,
			expectedOutput: `• Validating tag repo=OWNER/REPO tag=0.0.1
• Uploading release assets repo=OWNER/REPO tag=0.0.1
✓ Added release asset	name=any-name url=https://gitlab.example.com/OWNER/REPO/releases/0.0.1/downloads/any-path
	! Aliased deprecated ` + "`filepath`" + ` field to ` + "`direct_asset_path`" + `. Replace ` + "`filepath`" + ` with ` + "`direct_asset_path`" + `	name=any-name
✓ Upload succeeded after`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

			fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/releases/0%2E0%2E1", httpmock.NewStringResponse(http.StatusOK, `
				{
					"name": "test1",
					"tag_name": "0.0.1",
					"description": null,
					"created_at": "2023-01-19T02:58:32.622Z",
					"released_at": "2023-01-19T02:58:32.622Z",
					"upcoming_release": false,
					"tag_path": "/OWNER/REPO/-/tags/0.0.1"
				}
			`))

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

			if assert.NoErrorf(t, err, "error running command `release upload %s`: %v", tt.cli, err) {
				assert.Contains(t, output.Stderr(), tt.expectedOutput)
				assert.Empty(t, output.String())
			}
		})
	}
}
