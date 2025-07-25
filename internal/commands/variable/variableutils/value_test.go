package variableutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
)

func Test_getValue(t *testing.T) {
	tests := []struct {
		name     string
		valueArg string
		want     string
		stdin    string
	}{
		{
			name:     "literal value",
			valueArg: "a secret",
			want:     "a secret",
		},
		{
			name:  "from stdin",
			want:  "a secret",
			stdin: "a secret",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			io, stdin, _, _ := cmdtest.TestIOStreams()

			_, err := stdin.WriteString(tt.stdin)
			assert.NoError(t, err)

			args := []string{tt.valueArg}
			value, err := GetValue(tt.valueArg, io, args)

			assert.NoError(t, err)

			assert.Equal(t, value, tt.want)
		})
	}
}
