package delete

import (
	"bytes"
	"io"
	"testing"

	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/internal/config"

	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAliasDelete(t *testing.T) {
	tests := []struct {
		name       string
		config     string
		cli        string
		isTTY      bool
		wantStdout string
		wantStderr string
		wantErr    string
	}{
		{
			name:       "no aliases",
			config:     "",
			cli:        "co",
			isTTY:      true,
			wantStdout: "",
			wantStderr: "",
			wantErr:    "no such alias 'co'.",
		},
		{
			name: "delete one",
			config: heredoc.Doc(`
				aliases:
				  il: issue list
				  co: mr checkout
			`),
			cli:        "co",
			isTTY:      true,
			wantStdout: "",
			wantStderr: "✓ Deleted alias 'co'; was 'mr checkout'.\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer config.StubWriteConfig(io.Discard, io.Discard)()

			cfg := config.NewFromString(tt.config)

			ios, _, stdout, stderr := iostreams.Test()
			ios.IsaTTY = tt.isTTY
			ios.IsErrTTY = tt.isTTY

			factoryConf := &cmdtest.Factory{
				IOStub: ios,
				ConfigStub: func() (config.Config, error) {
					return cfg, nil
				},
			}

			cmd := NewCmdDelete(factoryConf, nil)

			argv, err := shlex.Split(tt.cli)
			require.NoError(t, err)
			cmd.SetArgs(argv)

			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			_, err = cmd.ExecuteC()
			if tt.wantErr != "" {
				if assert.Error(t, err) {
					assert.Equal(t, tt.wantErr, err.Error())
				}
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.wantStdout, stdout.String())
			assert.Equal(t, tt.wantStderr, stderr.String())
		})
	}
}
