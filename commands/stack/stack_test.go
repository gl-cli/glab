package stack

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/text"
	"gitlab.com/gitlab-org/cli/test"
)

func TestStackCmd(t *testing.T) {
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	assert.Nil(t, NewCmdStack(&cmdutils.Factory{}).Execute())

	out := test.ReturnBuffer(old, r, w)

	assert.Contains(t, out, "Stacked diffs are a way of creating small changes that build upon each other to ultimately deliver")
	assert.Contains(t, out, text.ExperimentalString)
}
