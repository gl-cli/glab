package cmdtest

import (
	"bytes"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/pkg/git"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/google/shlex"
	"github.com/otiai10/copy"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/test"
)

var (
	ProjectPath    string
	GlabBinaryPath string
)

type fatalLogger interface {
	Fatal(...any)
}

func init() {
	path := &bytes.Buffer{}
	// get root dir via git
	gitCmd := git.GitCommand("rev-parse", "--show-toplevel")
	gitCmd.Stdout = path
	err := gitCmd.Run()
	if err != nil {
		log.Fatalln("Failed to get root directory: ", err)
	}
	ProjectPath = strings.TrimSuffix(path.String(), "\n")
	if !strings.HasSuffix(ProjectPath, "/") {
		ProjectPath += "/"
	}
}

func InitTest(m *testing.M, suffix string) {
	// Build a glab binary with test symbols. If the parent test binary was run
	// with coverage enabled, enable coverage on the child binary, too.
	var err error
	GlabBinaryPath, err = filepath.Abs(os.ExpandEnv(ProjectPath + "testdata/glab.test"))
	if err != nil {
		log.Fatal(err)
	}
	testCmd := []string{"test", "-c", "-o", GlabBinaryPath, ProjectPath + "cmd/glab"}
	if coverMode := testing.CoverMode(); coverMode != "" {
		testCmd = append(testCmd, "-covermode", coverMode, "-coverpkg", "./...")
	}
	if out, err := exec.Command("go", testCmd...).CombinedOutput(); err != nil {
		log.Fatalf("Error building glab test binary: %s (%s)", string(out), err)
	}

	originalWd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	var repo string = CopyTestRepo(log.New(os.Stderr, "", log.LstdFlags), suffix)

	if err := os.Chdir(repo); err != nil {
		log.Fatalf("Error chdir to test/testdata: %s", err)
	}
	code := m.Run()

	if err := os.Chdir(originalWd); err != nil {
		log.Fatalf("Error chdir to original working dir: %s", err)
	}

	testdirs, err := filepath.Glob(os.ExpandEnv(repo))
	if err != nil {
		log.Printf("Error listing glob test/testdata-*: %s", err)
	}
	for _, dir := range testdirs {
		err := os.RemoveAll(dir)
		if err != nil {
			log.Printf("Error removing dir %s: %s", dir, err)
		}
	}

	os.Exit(code)
}

func RunCommand(cmd *cobra.Command, cli string, stds ...*bytes.Buffer) (*test.CmdOut, error) {
	// var stdin *bytes.Buffer
	var stderr *bytes.Buffer
	var stdout *bytes.Buffer

	//for i, std := range stds {
	//	if std != nil {
	//		if i == 0 {
	//			stdin = std
	//		}
	//		if i == 1 {
	//			stdout = std
	//		}
	//		if i == 2 {
	//			stderr = std
	//		}
	//	}
	//}
	//cmd.SetIn(stdin)
	//cmd.SetOut(stdout)
	//cmd.SetErr(stderr)

	argv, err := shlex.Split(cli)
	if err != nil {
		return nil, err
	}
	cmd.SetArgs(argv)
	_, err = cmd.ExecuteC()

	return &test.CmdOut{
		OutBuf: stdout,
		ErrBuf: stderr,
	}, err
}

func InitIOStreams(isTTY bool, doHyperlinks string) (*iostreams.IOStreams, *bytes.Buffer, *bytes.Buffer, *bytes.Buffer) {
	ios, stdin, stdout, stderr := iostreams.Test()
	ios.IsaTTY = isTTY
	ios.IsInTTY = isTTY
	ios.IsErrTTY = isTTY

	if doHyperlinks != "" {
		ios.SetDisplayHyperlinks(doHyperlinks)
	}

	return ios, stdin, stdout, stderr
}

func InitFactory(ios *iostreams.IOStreams, rt http.RoundTripper) *cmdutils.Factory {
	return &cmdutils.Factory{
		IO: ios,
		HttpClient: func() (*gitlab.Client, error) {
			a, err := api.TestClient(&http.Client{Transport: rt}, "", "", false)
			if err != nil {
				return nil, err
			}
			return a.Lab(), err
		},
		Config: func() (config.Config, error) {
			return config.NewBlankConfig(), nil
		},
		BaseRepo: func() (glrepo.Interface, error) {
			return glrepo.New("OWNER", "REPO"), nil
		},
		Branch: func() (string, error) {
			return "main", nil
		},
	}
}

type CmdExecFunc func(cli string) (*test.CmdOut, error)

type CmdFunc func(f *cmdutils.Factory) *cobra.Command

// FactoryOption is a function that configures a Factory
type FactoryOption func(f *cmdutils.Factory)

// WithGitLabClient configures the Factory with a specific GitLab client
func WithGitLabClient(client *gitlab.Client) FactoryOption {
	return func(f *cmdutils.Factory) {
		f.HttpClient = func() (*gitlab.Client, error) {
			return client, nil
		}
	}
}

// WithConfig configures the Factory with a specific config
func WithConfig(cfg config.Config) FactoryOption {
	return func(f *cmdutils.Factory) {
		f.Config = func() (config.Config, error) {
			return cfg, nil
		}
	}
}

func WithConfigError(err error) FactoryOption {
	return func(f *cmdutils.Factory) {
		f.Config = func() (config.Config, error) {
			return nil, err
		}
	}
}

// WithHttpClientError configures the Factory to return an error when creating HTTP client
func WithHttpClientError(err error) FactoryOption {
	return func(f *cmdutils.Factory) {
		f.HttpClient = func() (*gitlab.Client, error) {
			return nil, err
		}
	}
}

// WithBaseRepoError configures the Factory to return an error when getting base repo
func WithBaseRepoError(err error) FactoryOption {
	return func(f *cmdutils.Factory) {
		f.BaseRepo = func() (glrepo.Interface, error) {
			return nil, err
		}
	}
}

// WithBranchError configures the Factory to return an error when getting branch
func WithBranchError(err error) FactoryOption {
	return func(f *cmdutils.Factory) {
		f.Branch = func() (string, error) {
			return "", err
		}
	}
}

// WithBaseRepo configures the Factory with a specific base repository
func WithBaseRepo(owner, repo string) FactoryOption {
	return func(f *cmdutils.Factory) {
		f.BaseRepo = func() (glrepo.Interface, error) {
			return glrepo.New(owner, repo), nil
		}
	}
}

// WithBranch configures the Factory with a specific branch
func WithBranch(branch string) FactoryOption {
	return func(f *cmdutils.Factory) {
		f.Branch = func() (string, error) {
			return branch, nil
		}
	}
}

// NewTestFactory creates a Factory configured for testing with the given options
func NewTestFactory(t *testing.T, ios *iostreams.IOStreams, opts ...FactoryOption) *cmdutils.Factory {
	t.Helper()

	// Create a default factory
	f := &cmdutils.Factory{
		IO: ios,
		HttpClient: func() (*gitlab.Client, error) {
			return &gitlab.Client{}, nil
		},
		Config: func() (config.Config, error) {
			return config.NewBlankConfig(), nil
		},
		BaseRepo: func() (glrepo.Interface, error) {
			return glrepo.New("OWNER", "REPO"), nil
		},
		Branch: func() (string, error) {
			return "main", nil
		},
	}

	// Apply all options
	for _, opt := range opts {
		opt(f)
	}

	return f
}

// SetupCmdForTest creates a test environment with a configured Factory
func SetupCmdForTest(t *testing.T, cmdFunc CmdFunc, opts ...FactoryOption) CmdExecFunc {
	t.Helper()

	ios, _, stdout, stderr := InitIOStreams(true, "")

	// Create a default factory
	f := &cmdutils.Factory{
		IO: ios,
		HttpClient: func() (*gitlab.Client, error) {
			t.Errorf("You must configure a GitLab Test client in your tests. Use the WithGitLabClient option function")
			return nil, nil
		},
		Config: func() (config.Config, error) {
			return config.NewBlankConfig(), nil
		},
		BaseRepo: func() (glrepo.Interface, error) {
			return glrepo.New("OWNER", "REPO"), nil
		},
		Branch: func() (string, error) {
			return "main", nil
		},
	}

	// Apply all options
	for _, opt := range opts {
		opt(f)
	}

	return func(cli string) (*test.CmdOut, error) {
		return ExecuteCommand(cmdFunc(f), cli, stdout, stderr)
	}
}

func ExecuteCommand(cmd *cobra.Command, cli string, stdout *bytes.Buffer, stderr *bytes.Buffer) (*test.CmdOut, error) {
	argv, err := shlex.Split(cli)
	if err != nil {
		return nil, err
	}

	cmd.SetArgs(argv)
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err = cmd.ExecuteC()
	return &test.CmdOut{
		OutBuf: stdout,
		ErrBuf: stderr,
	}, err
}

func CopyTestRepo(log fatalLogger, name string) string {
	if name == "" {
		name = strconv.Itoa(int(rand.Uint64()))
	}
	dest, err := filepath.Abs(os.ExpandEnv(ProjectPath + "test/testdata-" + name))
	if err != nil {
		log.Fatal(err)
	}
	src, err := filepath.Abs(os.ExpandEnv(ProjectPath + "test/testdata"))
	if err != nil {
		log.Fatal(err)
	}
	if err := copy.Copy(src, dest); err != nil {
		log.Fatal(err)
	}
	// Move the test.git dir into the expected path at .git
	if !config.CheckPathExists(dest + "/.git") {
		if err := os.Rename(dest+"/test.git", dest+"/.git"); err != nil {
			log.Fatal(err)
		}
	}
	// Move the test.glab-cli dir into the expected path at .glab-cli
	if !config.CheckPathExists(dest + "/.glab-cli") {
		if err := os.Rename(dest+"/test.glab-cli", dest+"/.glab-cli"); err != nil {
			log.Fatal(err)
		}
	}
	return dest
}

func FirstLine(output []byte) string {
	if i := bytes.IndexAny(output, "\n"); i >= 0 {
		return strings.ReplaceAll(string(output)[0:i], "PASS", "")
	}
	return string(output)
}

func Eq(t *testing.T, got any, expected any) {
	t.Helper()
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func StubFactory(repo string) *cmdutils.Factory {
	cmdutils.CachedConfig = config.NewBlankConfig()

	f := cmdutils.NewFactory()
	if repo != "" {
		f.RepoOverride(repo)
	}

	return f
}

func StubFactoryWithConfig(repo string) (*cmdutils.Factory, error) {
	cmdutils.CachedConfig, cmdutils.ConfigError = config.ParseConfig("config.yml")
	if cmdutils.ConfigError != nil {
		return nil, cmdutils.ConfigError
	}
	f := cmdutils.NewFactory()
	if repo != "" {
		f.RepoOverride(repo)
	}

	return f, nil
}
