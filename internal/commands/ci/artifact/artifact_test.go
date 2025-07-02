package artifact

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"

	"gitlab.com/gitlab-org/cli/internal/api"

	"github.com/google/shlex"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/git"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
)

func doesFileExist(fileName string) bool {
	_, err := os.Stat(fileName)
	return err == nil
}

func createZipFile(t *testing.T, filename string) (string, string) {
	tempPath, err := os.MkdirTemp("/tmp", "testing_directory")
	require.NoError(t, err)

	archive, err := os.CreateTemp(tempPath, "test.*.zip")
	require.NoError(t, err)
	defer archive.Close()

	zipWriter := zip.NewWriter(archive)

	f1, err := os.Open("./testdata/file.txt")
	if err != nil {
		panic(err)
	}
	defer f1.Close()

	w1, err := zipWriter.Create(filename)
	if err != nil {
		panic(err)
	}
	if _, err := io.Copy(w1, f1); err != nil {
		panic(err)
	}

	zipWriter.Close()

	return tempPath, archive.Name()
}

func makeTestFactory(t *testing.T) (cmdutils.Factory, *httpmock.Mocker) {
	fakeHTTP := &httpmock.Mocker{
		MatchURL: httpmock.PathAndQuerystring,
	}

	io, _, _, _ := cmdtest.TestIOStreams()

	client := func(token, hostname string) (*api.Client, error) { // nolint:unparam
		return cmdtest.NewTestApiClient(t, &http.Client{Transport: fakeHTTP}, token, hostname), nil
	}

	// FIXME as mentioned in ./commands/auth/status/status_test.go,
	// httpmock seems to require a quick test run before it will work
	_, err := client("", "gitlab.com")
	if err != nil {
		return nil, nil
	}

	a, err := client("xxxx", "gitlab.com")
	if err != nil {
		return nil, nil
	}

	factory := cmdtest.NewTestFactory(io,
		cmdtest.WithConfig(config.NewBlankConfig()),
		cmdtest.WithGitLabClient(a.Lab()),
		cmdtest.WithBaseRepo("OWNER", "REPO"),
		func(f *cmdtest.Factory) {
			f.RemotesStub = func() (glrepo.Remotes, error) {
				return glrepo.Remotes{
					{
						Remote: &git.Remote{Name: "origin"},
						Repo:   glrepo.New("OWNER", "REPO", glinstance.DefaultHostname),
					},
				}, nil
			}
		},
		cmdtest.WithBranch("feature"),
	)

	return factory, fakeHTTP
}

func createSymlinkZip(t *testing.T) (string, string) {
	tempPath, err := os.MkdirTemp("/tmp", "testing_directory")
	require.NoError(t, err)

	archive, err := os.CreateTemp(tempPath, "test.*.zip")
	require.NoError(t, err)
	defer archive.Close()

	immutableFile, err := os.CreateTemp(tempPath, "immutable_file*.txt")
	require.NoError(t, err)

	immutableText := "Immutable text! GitLab is cool"
	_, err = immutableFile.WriteString(immutableText)
	require.NoError(t, err)

	err = os.Symlink(immutableFile.Name(), tempPath+"/symlink_file.txt")
	require.NoError(t, err)

	zipWriter := zip.NewWriter(archive)

	fixtureFile, err := os.Open("./testdata/file.txt")
	if err != nil {
		panic(err)
	}
	defer fixtureFile.Close()

	zipFile, err := zipWriter.Create("symlink_file.txt")
	if err != nil {
		panic(err)
	}
	if _, err := io.Copy(zipFile, fixtureFile); err != nil {
		panic(err)
	}

	zipWriter.Close()

	return tempPath, archive.Name()
}

func Test_NewCmdRun(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		want       string
		customPath string
		wantErr    bool
	}{
		{
			name:     "A regular filename",
			filename: "cli-v1.22.0.json",
			want:     "cli-v1.22.0.json",
		},
		{
			name:     "A regular filename in a directory",
			filename: "cli/cli-v1.22.0.json",
			want:     "cli/cli-v1.22.0.json",
		},
		{
			name:     "A filename with directory traversal",
			filename: "cli-v1.../../22.0.zip",
			wantErr:  true,
		},
		{
			name:     "A particularly nasty filename",
			filename: "..././..././..././etc/password_file",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory, fakeHTTP := makeTestFactory(t)
			tempPath, tempFileName := createZipFile(t, tt.filename)
			// defer os.Remove(tempFileName)

			fakeHTTP.RegisterResponder(http.MethodGet, `https://gitlab.com/api/v4/projects/OWNER%2FREPO/jobs/artifacts/main/download?job=secret_detection`,
				httpmock.NewFileResponse(http.StatusOK, tempFileName))

			cmd := NewCmdRun(factory)

			argv, err := shlex.Split("main secret_detection")
			if err != nil {
				t.Fatal(err)
			}

			cmd.SetArgs(argv)

			err = cmd.Flags().Set("path", tempPath)
			if err != nil {
				return
			}

			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			_, err = cmd.ExecuteC()
			filePathWanted := filepath.Join(tempPath, tt.want)

			if tt.wantErr {
				assert.Error(t, err, "Should error out if a path doesn't exist")
				return
			}

			assert.NoError(t, err, "Should not have errors")
			assert.True(t, doesFileExist(filePathWanted), "File should exist")

			if err != nil {
				t.Fatal(err)
			}
		})
	}

	t.Run("symlink can't overwrite", func(t *testing.T) {
		factory, fakeHTTP := makeTestFactory(t)

		tempPath, tempFileName := createSymlinkZip(t)
		defer os.Remove(tempFileName)

		fakeHTTP.RegisterResponder(http.MethodGet, `https://gitlab.com/api/v4/projects/OWNER%2FREPO/jobs/artifacts/main/download?job=secret_detection`,
			httpmock.NewFileResponse(http.StatusOK, tempFileName))

		cmd := NewCmdRun(factory)

		argv, err := shlex.Split("main secret_detection")
		if err != nil {
			t.Fatal(err)
		}

		cmd.SetArgs(argv)

		err = cmd.Flags().Set("path", tempPath)
		if err != nil {
			return
		}

		cmd.SetIn(&bytes.Buffer{})
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)

		_, err = cmd.ExecuteC()
		assert.Error(t, err, "file in artifact would overwrite a symbolic link- cannot extract")
	})
}
