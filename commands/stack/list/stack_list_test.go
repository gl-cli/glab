package list

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/pkg/git"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
)

func TestStackList(t *testing.T) {
	// GIVEN
	io, _, out, _ := iostreams.Test()
	stack := git.Stack{
		Refs: map[string]git.StackRef{
			"abc": {SHA: "abc", Prev: "", Next: "123", Branch: "abc", Description: "entry 1"},
			"123": {SHA: "123", Prev: "abc", Next: "def", Branch: "123", Description: "entry 2"},
			"def": {SHA: "def", Prev: "123", Next: "", Branch: "def", Description: "entry 3"},
		},
	}

	// WHEN
	run(io, stack, "123")

	lines := bytes.Split(out.Bytes(), []byte("\n"))
	assert.Len(t, lines, 4)
	assert.Equal(t, lines[0], []byte("  abc - entry 1"))
	assert.Equal(t, lines[1], []byte("> 123 - entry 2"))
	assert.Equal(t, lines[2], []byte("  def - entry 3"))
}
