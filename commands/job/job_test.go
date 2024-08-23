package job

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
)

var tests = []struct {
	name        string
	args        string
	expectedOut string
	expectedErr string
}{
	{
		name:        "when no args should display the help message",
		args:        "",
		expectedOut: "Use \"job [command] --help\" for more information about a command.\n",
		expectedErr: "",
	},
}

func TestJobCmd(t *testing.T) {
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wantedErr := ""
			if len(test.expectedErr) > 0 {
				wantedErr = test.expectedErr
			}

			// Catching Stdout & Stderr
			oldOut := os.Stdout
			rOut, wOut, _ := os.Pipe()
			os.Stdout = wOut
			outC := make(chan string)
			go func() {
				var buf bytes.Buffer
				_, _ = io.Copy(&buf, rOut)
				outC <- buf.String()
			}()

			oldErr := os.Stderr
			rErr, wErr, _ := os.Pipe()
			os.Stderr = wErr
			errC := make(chan string)
			go func() {
				var buf bytes.Buffer
				_, _ = io.Copy(&buf, rErr)
				errC <- buf.String()
			}()

			err := NewCmdJob(&cmdutils.Factory{}).Execute()

			// Rollbacking Stdout & Stderr
			wOut.Close()
			os.Stdout = oldOut
			stdout := <-outC
			wErr.Close()
			os.Stderr = oldErr
			stderr := <-errC

			if assert.NoErrorf(t, err, "error running `job %s`: %v", test.args, err) {
				assert.Contains(t, stderr, wantedErr)
				assert.Contains(t, stdout, test.expectedOut)
			}
		})
	}
}
