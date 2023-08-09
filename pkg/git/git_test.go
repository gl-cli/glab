package git

import (
	"errors"
	"os"
	"os/exec"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/gitlab-org/cli/internal/run"
	"gitlab.com/gitlab-org/cli/test"

	"github.com/stretchr/testify/require"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func Test_isFilesystemPath(t *testing.T) {
	type args struct {
		p string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Filesystem",
			args: args{"./.git"},
			want: true,
		},
		{
			name: "Filesystem",
			args: args{".git"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isFilesystemPath(tt.args.p); got != tt.want {
				t.Errorf("isFilesystemPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_UncommittedChangeCount(t *testing.T) {
	type c struct {
		Label    string
		Expected int
		Output   string
	}
	cases := []c{
		{Label: "no changes", Expected: 0, Output: ""},
		{Label: "one change", Expected: 1, Output: " M poem.txt"},
		{Label: "untracked file", Expected: 2, Output: " M poem.txt\n?? new.txt"},
	}

	teardown := run.SetPrepareCmd(func(*exec.Cmd) run.Runnable {
		return &test.OutputStub{}
	})
	defer teardown()

	for _, v := range cases {
		_ = run.SetPrepareCmd(func(*exec.Cmd) run.Runnable {
			return &test.OutputStub{Out: []byte(v.Output)}
		})
		ucc, _ := UncommittedChangeCount()

		if ucc != v.Expected {
			t.Errorf("got unexpected ucc value: %d for case %s", ucc, v.Label)
		}
	}
}

func Test_CurrentBranch(t *testing.T) {
	cs, teardown := test.InitCmdStubber()
	defer teardown()

	expected := "branch-name"

	cs.Stub(expected)

	result, err := CurrentBranch()
	if err != nil {
		t.Errorf("got unexpected error: %v", err)
	}
	if len(cs.Calls) != 1 {
		t.Errorf("expected 1 git call, saw %d", len(cs.Calls))
	}
	if result != expected {
		t.Errorf("unexpected branch name: %s instead of %s", result, expected)
	}
}

func Test_CurrentBranch_detached_head(t *testing.T) {
	cs, teardown := test.InitCmdStubber()
	defer teardown()

	cs.StubError("")

	_, err := CurrentBranch()
	if err == nil {
		t.Errorf("expected an error")
	}
	if err != ErrNotOnAnyBranch {
		t.Errorf("got unexpected error: %s instead of %s", err, ErrNotOnAnyBranch)
	}
	if len(cs.Calls) != 1 {
		t.Errorf("expected 1 git call, saw %d", len(cs.Calls))
	}
}

func Test_CurrentBranch_unexpected_error(t *testing.T) {
	cs, teardown := test.InitCmdStubber()
	defer teardown()

	cs.StubError("lol")

	expectedError := "lol\nstub: lol"

	_, err := CurrentBranch()
	if err == nil {
		t.Errorf("expected an error")
	}
	if err.Error() != expectedError {
		t.Errorf("got unexpected error: %s instead of %s", err.Error(), expectedError)
	}
	if len(cs.Calls) != 1 {
		t.Errorf("expected 1 git call, saw %d", len(cs.Calls))
	}
}

func TestParseExtraCloneArgs(t *testing.T) {
	type Wanted struct {
		args []string
		dir  string
	}
	tests := []struct {
		name string
		args []string
		want Wanted
	}{
		{
			name: "args and target",
			args: []string{"target_directory", "-o", "upstream", "--depth", "1"},
			want: Wanted{
				args: []string{"-o", "upstream", "--depth", "1"},
				dir:  "target_directory",
			},
		},
		{
			name: "only args",
			args: []string{"-o", "upstream", "--depth", "1"},
			want: Wanted{
				args: []string{"-o", "upstream", "--depth", "1"},
				dir:  "",
			},
		},
		{
			name: "only target",
			args: []string{"target_directory"},
			want: Wanted{
				args: []string{},
				dir:  "target_directory",
			},
		},
		{
			name: "no args",
			args: []string{},
			want: Wanted{
				args: []string{},
				dir:  "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, dir := parseCloneArgs(tt.args)
			got := Wanted{
				args: args,
				dir:  dir,
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v want %#v", got, tt.want)
			}
		})
	}
}

func TestReadBranchConfig(t *testing.T) {
	cs, teardown := test.InitCmdStubber()
	defer teardown()

	cs.Stub(`branch.branch-name.remote origin
branch.branch.remote git@gitlab.com:glab-test/test.git
branch.branch.merge refs/heads/branch-name`)

	u, err := ParseURL("git@gitlab.com:glab-test/test.git")
	assert.Nil(t, err)
	wantCfg := BranchConfig{
		"origin",
		u,
		"refs/heads/branch-name",
	}

	t.Run("", func(t *testing.T) {
		if gotCfg := ReadBranchConfig("branch-name"); !reflect.DeepEqual(gotCfg, wantCfg) {
			t.Errorf("ReadBranchConfig() = %v, want %v", gotCfg, wantCfg)
		}
	})
}

func Test_parseRemotes(t *testing.T) {
	remoteList := []string{
		"mona\tgit@gitlab.com:monalisa/myfork.git (fetch)",
		"origin\thttps://gitlab.com/monalisa/octo-cat.git (fetch)",
		"origin\thttps://gitlab.com/monalisa/octo-cat-push.git (push)",
		"upstream\thttps://example.com/nowhere.git (fetch)",
		"upstream\thttps://gitlab.com/hubot/tools (push)",
		"zardoz\thttps://example.com/zed.git (push)",
	}
	r := parseRemotes(remoteList)
	eq(t, len(r), 4)

	eq(t, r[0].Name, "mona")
	eq(t, r[0].FetchURL.String(), "ssh://git@gitlab.com/monalisa/myfork.git")
	if r[0].PushURL != nil {
		t.Errorf("expected no PushURL, got %q", r[0].PushURL)
	}
	eq(t, r[1].Name, "origin")
	eq(t, r[1].FetchURL.Path, "/monalisa/octo-cat.git")
	eq(t, r[1].PushURL.Path, "/monalisa/octo-cat-push.git")

	eq(t, r[2].Name, "upstream")
	eq(t, r[2].FetchURL.Host, "example.com")
	eq(t, r[2].PushURL.Host, "gitlab.com")

	eq(t, r[3].Name, "zardoz")
}

func TestGetDefaultBranch(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		want    string
		wantErr bool
	}{
		{
			name: "No Params",
			want: "master",
		},
		{
			name: "Different Remote",
			want: "master",
			args: "profclems/test",
		},
		{
			name:    "Invalid repo",
			want:    "master",
			args:    "testssz",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetDefaultBranch(tt.args)
			if (err != nil) != tt.wantErr {
				t.Logf("GetDefaultBranch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetDefaultBranch() got = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("get branch from HEAD", func(t *testing.T) {
		cs, teardown := test.InitCmdStubber()
		defer teardown()

		cs.Stub(`* remote origin
Fetch URL: https://gitlab.com/gitlab-community/cli.git
Push  URL: https://gitlab.com/gitlab-community/cli.git
HEAD branch: main`)

		got, err := GetDefaultBranch("origin")
		assert.Nil(t, err)
		assert.Equal(t, "main", got)
	})
}

func TestGetRemoteURL(t *testing.T) {
	tests := []struct {
		name        string
		remoteAlias string
		want        string
		wantErr     bool
	}{
		{
			name:        "isInvalid",
			remoteAlias: "someorigin",
			wantErr:     true,
		},
		{
			name:        "isInvalid",
			remoteAlias: "origin",
			want:        getEnv("CI_PROJECT_PATH", "gitlab-org/cli"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetRemoteURL(tt.remoteAlias)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRemoteURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Contains(t, got, tt.want)
		})
	}
}

func TestDescribeByTags(t *testing.T) {
	cases := map[string]struct {
		expected   string
		output     string
		errorValue error
	}{
		"invalid repository": {
			expected:   "",
			output:     "",
			errorValue: errors.New("fatal: not a git repository (or any of the parent directories): .git"),
		},
		"commit is tag": {
			expected:   "1.0.0",
			output:     "1.0.0",
			errorValue: nil,
		},
		"commit is not tag": {
			expected:   "1.0.0-1-g4aa1b8",
			output:     "1.0.0-1-g4aa1b8",
			errorValue: nil,
		},
	}

	t.Cleanup(func() {
		teardown := run.SetPrepareCmd(func(*exec.Cmd) run.Runnable {
			return &test.OutputStub{}
		})
		teardown()
	})

	for name, v := range cases {
		t.Run(name, func(t *testing.T) {
			_ = run.SetPrepareCmd(func(*exec.Cmd) run.Runnable {
				return &test.OutputStub{Out: []byte(v.output), Error: v.errorValue}
			})

			version, err := DescribeByTags()
			require.Equal(t, v.errorValue, errors.Unwrap(err))
			require.Equal(t, v.expected, version, "unexpected version value for case %s", name)
		})
	}
}

func Test_assertValidConfig(t *testing.T) {
	t.Run("config key is valid", func(t *testing.T) {
		err := assertValidConfigKey("remote.this.testsuite")
		require.NoError(t, err)
	})
	t.Run("config key is valid", func(t *testing.T) {
		err := assertValidConfigKey("this.testsuite")
		require.NoError(t, err)
	})

	t.Run("panic modes", func(t *testing.T) {
		err := assertValidConfigKey("this")
		require.Error(t, err)
		require.Errorf(t, err, "incorrect git config key")
	})
}

func Test_configValueExists(t *testing.T) {
	// TODO(gitlab-org/cli#3778): To ensure that the commands
	// work against a real repository, and drop all stubbing,
	// we'll need to implement some test setup code that inits
	// a git repository for a test.
	//
	// See https://gitlab.com/gitlab-org/cli/-/issues/3778
	cs, teardown := test.InitCmdStubber()
	defer teardown()

	t.Run("config value does not exist", func(t *testing.T) {
		cs.Stub("does not match")
		v, err := configValueExists("remote.this.testsuite", "rocks")
		require.NoError(t, err)
		require.Equal(t, false, v)
	})

	t.Run("config value exists", func(t *testing.T) {
		cs.Stub("rocks")
		v, err := configValueExists("remote.this.testsuite", "rocks")
		require.NoError(t, err)
		require.Equal(t, true, v)
	})
}

func TestSetConfig(t *testing.T) {
	// TODO(gitlab-org/cli#3778): To ensure that the commands
	// work against a real repository, and drop all stubbing,
	// we'll need to implement some test setup code that inits
	// a git repository for a test.
	//
	// See https://gitlab.com/gitlab-org/cli/-/issues/3778
	cs, teardown := test.InitCmdStubber()
	defer teardown()

	t.Run("config value does not exist", func(t *testing.T) {
		cs.Stub("")
		cs.Stub("")
		err := SetConfig("this.testsuite", "rocks")
		require.NoError(t, err)
	})

	t.Run("config value exists", func(t *testing.T) {
		cs.Stub("rocks")
		err := SetConfig("this.testsuite", "rocks")
		require.NoError(t, err)
	})

	t.Run("unknown error occurred", func(t *testing.T) {
		cs.StubError("unknown error occurred")
		err := SetConfig("this.testsuite", "rocks")
		require.Error(t, err)
	})
}

func TestListTags(t *testing.T) {
	cases := map[string]struct {
		expected []string
		output   string
		errorVal error
	}{
		"invalid repository": {
			expected: nil,
			output:   "",
			errorVal: errors.New("fatal: not a git repository (or any of the parent directories): .git"),
		},
		"no tags": {
			expected: nil,
			output:   "",
			errorVal: nil,
		},
		"no tags w/ extra newline": {
			expected: []string{},
			output:   "\n",
			errorVal: nil,
		},
		"single semver tag": {
			expected: []string{"1.0.0"},
			output:   "1.0.0",
			errorVal: nil,
		},
		"multiple semver tags": {
			expected: []string{"1.0.0", "2.0.0", "3.0.0"},
			output:   "1.0.0\n2.0.0\n3.0.0",
			errorVal: nil,
		},
		"multiple semver tags with extra newlines": {
			expected: []string{"1.0.0", "2.0.0", "3.0.0"},
			output:   "1.0.0\n2.0.0\n3.0.0\n\n",
			errorVal: nil,
		},
		"single non-semver tag": {
			expected: []string{"a"},
			output:   "a",
			errorVal: nil,
		},
		"multiple non-semver tag": {
			expected: []string{"a", "b"},
			output:   "a\nb",
			errorVal: nil,
		},
	}

	t.Cleanup(func() {
		teardown := run.SetPrepareCmd(func(*exec.Cmd) run.Runnable {
			return &test.OutputStub{}
		})
		teardown()
	})

	for name, v := range cases {
		t.Run(name, func(t *testing.T) {
			_ = run.SetPrepareCmd(func(*exec.Cmd) run.Runnable {
				return &test.OutputStub{Out: []byte(v.output), Error: v.errorVal}
			})

			tags, err := ListTags()

			require.Equal(t, v.errorVal, errors.Unwrap(err))
			require.Equal(t, v.expected, tags)
		})
	}
}
