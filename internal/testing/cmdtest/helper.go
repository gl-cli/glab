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
	"runtime"
	"strconv"
	"strings"
	"testing"

	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/git"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/iostreams"

	"github.com/google/shlex"
	"github.com/otiai10/copy"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
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

// WithTestIOStreamsAsTTY sets stdin, stdout and stderr as TTY
// By default they are not treated as TTYs. This will overwrite the behavior
// for the three of them. If you only want to set a specific one,
// use iostreams.WithStdin, iostreams.WithStdout or iostreams.WithStderr.
func WithTestIOStreamsAsTTY(asTTY bool) iostreams.IOStreamsOption {
	return func(i *iostreams.IOStreams) {
		i.IsInTTY = asTTY
		i.IsaTTY = asTTY
		i.IsErrTTY = asTTY
	}
}

func TestIOStreams(options ...iostreams.IOStreamsOption) (*iostreams.IOStreams, *bytes.Buffer, *bytes.Buffer, *bytes.Buffer) {
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	opts := []iostreams.IOStreamsOption{
		iostreams.WithStdin(io.NopCloser(in), false),
		iostreams.WithStdout(out, false),
		iostreams.WithStderr(errOut, false),
	}
	opts = append(opts, options...)

	ios := iostreams.New(opts...)

	return ios, in, out, errOut
}

type Factory struct {
	ApiClientStub  func(repoHost string, cfg config.Config) (*api.Client, error)
	HttpClientStub func() (*gitlab.Client, error)
	BaseRepoStub   func() (glrepo.Interface, error)
	RemotesStub    func() (glrepo.Remotes, error)
	ConfigStub     func() config.Config
	BranchStub     func() (string, error)
	IOStub         *iostreams.IOStreams
	BuildInfoStub  api.BuildInfo

	repoOverride string
}

func (f *Factory) RepoOverride(repo string) error {
	f.repoOverride = repo
	return nil
}

func (f *Factory) ApiClient(repoHost string, cfg config.Config) (*api.Client, error) {
	return f.ApiClientStub(repoHost, cfg)
}

func (f *Factory) HttpClient() (*gitlab.Client, error) {
	return f.HttpClientStub()
}

func (f *Factory) BaseRepo() (glrepo.Interface, error) {
	if f.repoOverride != "" {
		return glrepo.FromFullName(f.repoOverride, glinstance.DefaultHostname)
	}
	return f.BaseRepoStub()
}

func (f *Factory) Remotes() (glrepo.Remotes, error) {
	return f.RemotesStub()
}

func (f *Factory) Config() config.Config {
	return f.ConfigStub()
}

func (f *Factory) Branch() (string, error) {
	return f.BranchStub()
}

func (f *Factory) IO() *iostreams.IOStreams {
	return f.IOStub
}

func (f *Factory) DefaultHostname() string {
	return glinstance.DefaultHostname
}

func (f *Factory) BuildInfo() api.BuildInfo {
	return f.BuildInfoStub
}

type CmdExecFunc func(cli string) (*test.CmdOut, error)

type CmdFunc func(cmdutils.Factory) *cobra.Command

// FactoryOption is a function that configures a Factory
type FactoryOption func(f *Factory)

// WithApiClient configures the Factory with a specific API client
func WithApiClient(client *api.Client) FactoryOption {
	return func(f *Factory) {
		f.ApiClientStub = func(repoHost string, cfg config.Config) (*api.Client, error) {
			return client, nil
		}
	}
}

// WithGitLabClient configures the Factory with a specific GitLab client
func WithGitLabClient(client *gitlab.Client) FactoryOption {
	return func(f *Factory) {
		f.HttpClientStub = func() (*gitlab.Client, error) {
			return client, nil
		}
	}
}

// WithConfig configures the Factory with a specific config
func WithConfig(cfg config.Config) FactoryOption {
	return func(f *Factory) {
		f.ConfigStub = func() config.Config {
			return cfg
		}
	}
}

// WithHttpClientError configures the Factory to return an error when creating HTTP client
func WithHttpClientError(err error) FactoryOption {
	return func(f *Factory) {
		f.HttpClientStub = func() (*gitlab.Client, error) {
			return nil, err
		}
	}
}

// WithBaseRepoError configures the Factory to return an error when getting base repo
func WithBaseRepoError(err error) FactoryOption {
	return func(f *Factory) {
		f.BaseRepoStub = func() (glrepo.Interface, error) {
			return nil, err
		}
	}
}

// WithBranchError configures the Factory to return an error when getting branch
func WithBranchError(err error) FactoryOption {
	return func(f *Factory) {
		f.BranchStub = func() (string, error) {
			return "", err
		}
	}
}

// WithBaseRepo configures the Factory with a specific base repository
func WithBaseRepo(owner, repo string) FactoryOption {
	return func(f *Factory) {
		f.BaseRepoStub = func() (glrepo.Interface, error) {
			return glrepo.New(owner, repo, glinstance.DefaultHostname), nil
		}
	}
}

// WithBranch configures the Factory with a specific branch
func WithBranch(branch string) FactoryOption {
	return func(f *Factory) {
		f.BranchStub = func() (string, error) {
			return branch, nil
		}
	}
}

// WithBuildInfo configures the Factory build information
func WithBuildInfo(buildInfo api.BuildInfo) FactoryOption {
	return func(f *Factory) {
		f.BuildInfoStub = buildInfo
	}
}

// NewTestFactory creates a Factory configured for testing with the given options
func NewTestFactory(ios *iostreams.IOStreams, opts ...FactoryOption) *Factory {
	f := &Factory{
		IOStub: ios,
		ApiClientStub: func(repoHost string, cfg config.Config) (*api.Client, error) {
			return &api.Client{}, nil
		},
		HttpClientStub: func() (*gitlab.Client, error) {
			return &gitlab.Client{}, nil
		},
		ConfigStub: func() config.Config {
			return config.NewBlankConfig()
		},
		BaseRepoStub: func() (glrepo.Interface, error) {
			return glrepo.New("OWNER", "REPO", glinstance.DefaultHostname), nil
		},
		BranchStub: func() (string, error) {
			return "main", nil
		},
		BuildInfoStub: api.BuildInfo{Version: "test", Commit: "test", Platform: runtime.GOOS, Architecture: runtime.GOARCH},
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

	ios, _, stdout, stderr := TestIOStreams(WithTestIOStreamsAsTTY(true))

	f := NewTestFactory(ios, opts...)
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

func NewTestApiClient(t *testing.T, httpClient *http.Client, token, host string, options ...api.ClientOption) *api.Client {
	t.Helper()

	opts := []api.ClientOption{
		api.WithUserAgent("glab test client"),
		api.WithBaseURL(glinstance.APIEndpoint(host, glinstance.DefaultProtocol, "")),
		api.WithInsecureSkipVerify(true),
		api.WithHTTPClient(httpClient),
	}
	opts = append(opts, options...)
	testClient, err := api.NewClient(
		func(*http.Client) (gitlab.AuthSource, error) { return gitlab.AccessTokenAuthSource{Token: token}, nil },
		opts...,
	)
	require.NoError(t, err)
	return testClient
}
