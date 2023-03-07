package lint

import (
	"testing"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/alecthomas/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
)

func Test_lintRun(t *testing.T) {
	io, _, stdout, stderr := iostreams.Test()
	fac := cmdtest.StubFactory("")
	fac.IO = io
	fac.IO.StdErr = stderr
	fac.IO.StdOut = stdout

	tests := []struct {
		name    string
		path    string
		StdOut  string
		StdErr  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "with invalid path specified",
			path:    "WRONG_PATH",
			StdOut:  "",
			StdErr:  "Getting contents in WRONG_PATH\n",
			wantErr: true,
			errMsg:  "WRONG_PATH: no such file or directory",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := lintRun(fac, tt.path)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("lintRun() error = %v, wantErr %v", err, tt.wantErr)
				}
				assert.Equal(t, tt.errMsg, err.Error())
			}

			assert.Equal(t, tt.StdErr, stderr.String())
			assert.Equal(t, tt.StdOut, stdout.String())
		})
	}
}
