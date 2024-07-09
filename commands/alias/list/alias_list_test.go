package list

import (
	"bytes"
	"io"
	"testing"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAliasList(t *testing.T) {
	tests := []struct {
		name       string
		config     string
		isaTTy     bool
		wantStdout string
		wantStderr string
	}{
		{
			name:       "empty",
			config:     "",
			wantStdout: "",
			isaTTy:     true,
			wantStderr: "no aliases configured.\n",
		},
		{
			name: "some",
			config: heredoc.Doc(`
				aliases:
				  co: mr checkout
				  gc: "!glab mr create -f \"$@\" | pbcopy"
			`),
			wantStdout: "co\tmr checkout\ngc\t!glab mr create -f \"$@\" | pbcopy\n",
			wantStderr: "",
			isaTTy:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: change underlying config implementation so Write is not
			// automatically called when editing aliases in-memory
			defer config.StubWriteConfig(io.Discard, io.Discard)()

			cfg := config.NewFromString(tt.config)

			ios, _, stdout, stderr := iostreams.Test()
			ios.IsaTTY = tt.isaTTy
			ios.IsErrTTY = tt.isaTTy

			factoryConf := &cmdutils.Factory{
				Config: func() (config.Config, error) {
					return cfg, nil
				},
				IO: ios,
			}

			cmd := NewCmdList(factoryConf, nil)
			cmd.SetArgs([]string{})

			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			_, err := cmd.ExecuteC()
			require.NoError(t, err)

			assert.Equal(t, tt.wantStdout, stdout.String())
			assert.Equal(t, tt.wantStderr, stderr.String())
		})
	}
}
