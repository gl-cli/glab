package lint

import (
	"testing"

	"gitlab.com/gitlab-org/cli/commands/cmdtest"
)

func TestMain(m *testing.M) {
	cmdtest.InitTest(m, "ci_lint_test")
}

func Test_pipelineCILint_Integration(t *testing.T) {
	// TODO: This test is temporarily disabled because of
	// https://gitlab.com/gitlab-org/cli/-/issues/1105

	// glTestHost := test.GetHostOrSkip(t)
	//
	// io, _, stdout, stderr := iostreams.Test()
	// fac := cmdtest.StubFactory(glTestHost)
	// fac.IO = io
	// fac.IO.StdErr = stderr
	// fac.IO.StdOut = stdout
	//
	// tests := []struct {
	// 	Name    string
	// 	Args    string
	// 	StdOut  string
	// 	StdErr  string
	// 	WantErr error
	// }{
	// 	{
	// 		Name:   "with no path specified",
	// 		Args:   "",
	// 		StdOut: "✓ CI/CD YAML is valid!\n",
	// 		StdErr: "Getting contents in .gitlab-ci.yml\nValidating...\n",
	// 	},
	// 	{
	// 		Name:   "with path specified as url",
	// 		Args:   glTestHost + "/gitlab-org/cli/-/raw/main/.gitlab-ci.yml",
	// 		StdOut: "✓ CI/CD YAML is valid!\n",
	// 		StdErr: "Getting contents in " + glTestHost + "/gitlab-org/cli/-/raw/main/.gitlab-ci.yml\nValidating...\n",
	// 	},
	// }
	//
	// cmd := NewCmdLint(fac)
	//
	// for _, test := range tests {
	// 	t.Run(test.Name, func(t *testing.T) {
	// 		_, err := cmdtest.RunCommand(cmd, test.Args)
	// 		if err != nil {
	// 			if test.WantErr == nil {
	// 				t.Fatal(err)
	// 			}
	// 			assert.Equal(t, err, test.WantErr)
	// 		}
	// 		assert.Equal(t, test.StdErr, stderr.String())
	// 		assert.Equal(t, test.StdOut, stdout.String())
	// 		stdout.Reset()
	// 		stderr.Reset()
	// 	})
	// }
}
