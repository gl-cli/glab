package archive

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"gitlab.com/gitlab-org/cli/test"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
)

func TestMain(m *testing.M) {
	cmdtest.InitTest(m, "repo_archive_test")
}

func Test_repoArchive_Integration(t *testing.T) {
	glTestHost := test.GetHostOrSkip(t)
	t.Setenv("GITLAB_HOST", glTestHost)
	repo := cmdtest.CopyTestRepo(t, "repo_archive_test")

	type argFlags struct {
		format string
		sha    string
		repo   string
		dest   string
	}

	tests := []struct {
		name    string
		args    argFlags
		wantMsg []string
		wantErr bool
	}{
		{
			name:    "Has invalid format",
			args:    argFlags{"asp", "master", "cli-automated-testing/test", "test"},
			wantMsg: []string{"format must be one of"},
			wantErr: true,
		},
		{
			name:    "Has valid format",
			args:    argFlags{"zip", "master", "cli-automated-testing/test", "test"},
			wantMsg: []string{"Cloning...", "Complete... test.zip"},
		},
		{
			name:    "Repo is invalid",
			args:    argFlags{"zip", "master", "cli-automated-testing/testzz", "test"},
			wantMsg: []string{"404 Not Found"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nbusy := 0
			for {
				cmd := exec.Command(cmdtest.GlabBinaryPath, "repo", "archive", tt.args.repo, tt.args.dest, "--format", tt.args.format, "--sha", tt.args.sha)
				cmd.Dir = repo
				b, err := cmd.CombinedOutput()

				// Sleep to avoid the "text file busy" race condition.
				// This can happen on Unix when another process has the binary
				// we want to execute open for writing. In this case, it happens
				// because the binary was built and then executed to run the
				// integration tests by using fork+exec. This can result in
				// the error code ETXTBSY and message "text file busy".
				// Following Go upstream with hardcoding the error string and
				// sleeping for a little bit to retry the command after the
				// file lock has been released hopefully.
				// https://golang.org/issue/3001
				// https://go.googlesource.com/go/+/go1.9.5/src/cmd/go/internal/work/build.go#2018
				if err != nil && nbusy < 3 && strings.Contains(err.Error(), "text file busy") {
					time.Sleep(100 * time.Millisecond << uint(nbusy))
					nbusy++
					continue
				}

				out := string(b)
				t.Log(out)

				if err != nil {
					t.Log(err)
					if !tt.wantErr {
						t.Fatal(err)
					}
				}

				for _, msg := range tt.wantMsg {
					assert.Contains(t, out, msg)
				}
				return
			}
		})
	}
}
