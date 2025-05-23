package git_mock

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"gitlab.com/gitlab-org/cli/internal/run"
	"gitlab.com/gitlab-org/cli/pkg/utils"
)

//go:generate go run go.uber.org/mock/mockgen@v0.4.0 -typed -destination=./git_interface_mock_for_test.go -package=git_mock gitlab.com/gitlab-org/cli/pkg/git_mock GitInterface

const DefaultRemote = "origin"

// ErrNotOnAnyBranch indicates that the user is in detached HEAD state
var ErrNotOnAnyBranch = errors.New("you're not on any Git branch (a 'detached HEAD' state).")

type (
	GitInterface interface {
		CheckoutBranch(branch string) error
		CurrentBranch() (string, error)
		DeleteLocalBranch(branch string) error
		GetDefaultBranch(remote string) (string, error)
		RemoteBranchExists(branch string) (bool, error)
	}

	StandardGitRunner struct {
		gitBinary string
	}
)

var _ GitInterface = (*StandardGitRunner)(nil)

func NewStandardGitRunner(gitBinary string) *StandardGitRunner {
	if gitBinary == "" {
		gitBinary = "git" // default to using "git" from PATH
	}
	return &StandardGitRunner{gitBinary: gitBinary}
}

// CurrentBranch reads the checked-out branch for the git repository
func (g *StandardGitRunner) CurrentBranch() (string, error) {
	stdout, stderr, err := g.runGitCommand("symbolic-ref", "--quiet", "--short", "HEAD")
	if err == nil {
		return utils.FirstLine(stdout), nil
	}
	var cmdErr *run.CmdError
	if errors.As(err, &cmdErr) {
		if cmdErr.Stderr.Len() == 0 {
			return "", ErrNotOnAnyBranch // Detached HEAD error
		}
	}
	return "", fmt.Errorf("unknown error getting current branch: %v - %s", err, stderr)
}

// GetDefaultBranch finds and returns the remote's default branch
func (g *StandardGitRunner) GetDefaultBranch(remote string) (string, error) {
	stdout, stderr, err := g.runGitCommand("remote", "show", remote)
	if err != nil {
		return "", fmt.Errorf("could not find default branch: %v - %s", err, stderr)
	}
	return ParseDefaultBranch(stdout)
}

// RemoteBranchExists returns a boolean if the remote branch exists
func (g *StandardGitRunner) RemoteBranchExists(branch string) (bool, error) {
	_, stderr, err := g.runGitCommand("ls-remote", "--exit-code", "--heads", DefaultRemote, branch)
	if err != nil {
		return false, fmt.Errorf("could not find remote branch: %v - %s", err, stderr)
	}
	return true, nil
}

// DeleteLocalBranch deletes a local git branch
func (g *StandardGitRunner) DeleteLocalBranch(branch string) error {
	_, stderr, err := g.runGitCommand("branch", "-D", branch)
	if err != nil {
		return fmt.Errorf("could not delete local branch: %v - %s", err, stderr)
	}
	return nil
}

// CheckoutBranch checks out a branch in the current git repository
func (g *StandardGitRunner) CheckoutBranch(branch string) error {
	_, stderr, err := g.runGitCommand("checkout", branch)
	if err != nil {
		return fmt.Errorf("could not checkout branch: %v - %s", err, stderr)
	}
	return nil
}

// runGitCommand executes a git command with proper environment setup and returns stdout/stderr
func (g *StandardGitRunner) runGitCommand(args ...string) ([]byte, string, error) {
	cmd := exec.Command(g.gitBinary, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// ensure output from git is in English for string matching
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "LC_ALL=C")

	err := cmd.Run()
	return stdout.Bytes(), stderr.String(), err
}

func ParseDefaultBranch(output []byte) (string, error) {
	var headBranch string

	for _, o := range strings.Split(string(output), "\n") {
		o = strings.TrimSpace(o)
		r, err := regexp.Compile(`(HEAD branch:)\s+`)
		if err != nil {
			return "master", err
		}
		if r.MatchString(o) {
			headBranch = strings.TrimPrefix(o, "HEAD branch: ")
			break
		}
	}
	return headBranch, nil
}
