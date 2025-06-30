package bootstrap

import (
	"fmt"
	"io"
	"os/exec"
)

type Cmd interface {
	RunWithOutput(name string, arg ...string) ([]byte, error)
	Run(name string, arg ...string) error
}

type cmdWrapper struct {
	stdout, stderr io.Writer
	env            []string
}

type errorWithOutput struct {
	output []byte
	err    error
}

func (e errorWithOutput) Error() string {
	return fmt.Sprintf("command failed with %q and output:\n%s", e.err, e.output)
}

func (e errorWithOutput) Unwrap() error {
	return e.err
}

func NewCmd(stdout, stderr io.Writer, env []string) Cmd {
	return &cmdWrapper{
		stdout: stdout,
		stderr: stderr,
		env:    env,
	}
}

func (c *cmdWrapper) RunWithOutput(name string, arg ...string) ([]byte, error) {
	command := exec.Command(name, arg...)
	command.Env = c.env
	output, err := command.CombinedOutput()
	if err != nil {
		return output, &errorWithOutput{output: output, err: err}
	}
	return output, nil
}

func (c *cmdWrapper) Run(name string, arg ...string) error {
	command := exec.Command(name, arg...)
	command.Stdout = c.stdout
	command.Stderr = c.stderr
	command.Env = c.env
	return command.Run()
}
