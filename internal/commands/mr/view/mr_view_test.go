//go:build !integration

package view

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/acarl005/stripansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/run"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	mainTest "gitlab.com/gitlab-org/cli/test"
)

var (
	f         cmdutils.Factory
	stdout    *bytes.Buffer
	stderr    *bytes.Buffer
	ioStreams *iostreams.IOStreams
)

func TestMain(m *testing.M) {
	ioStreams, _, stdout, stderr = cmdtest.TestIOStreams(cmdtest.WithTestIOStreamsAsTTY(true))
	client, _ := gitlab.NewClient("")
	f = cmdtest.NewTestFactory(
		ioStreams,
		cmdtest.WithConfig(config.NewFromString(heredoc.Doc(`
			hosts:
			  gitlab.com:
			    username: monalisa
			    token: OTOKEN
		`))),
		cmdtest.WithGitLabClient(client),
	)

	timer, _ := time.Parse(time.RFC3339, "2014-11-12T11:45:26.371Z")
	api.GetMR = func(client *gitlab.Client, projectID any, mrID int64, opts *gitlab.GetMergeRequestsOptions) (*gitlab.MergeRequest, error) {
		if projectID == "" || projectID == "WRONG_REPO" || projectID == "expected_err" {
			return nil, fmt.Errorf("error expected")
		}
		repo, err := f.BaseRepo()
		if err != nil {
			return nil, err
		}
		return &gitlab.MergeRequest{
			BasicMergeRequest: gitlab.BasicMergeRequest{
				ID:          mrID,
				IID:         mrID,
				Title:       "mrTitle",
				Labels:      gitlab.Labels{"test", "bug"},
				State:       "opened",
				Description: "mrBody",
				Author: &gitlab.BasicUser{
					ID:       mrID,
					Name:     "John Dev Wick",
					Username: "jdwick",
				},
				Assignees: []*gitlab.BasicUser{
					{
						Username: "mona",
					},
					{
						Username: "lisa",
					},
				},
				Reviewers: []*gitlab.BasicUser{
					{
						Username: "lisa",
					},
					{
						Username: "mona",
					},
				},
				WebURL:         fmt.Sprintf("https://%s/%s/-/merge_requests/%d", repo.RepoHost(), repo.FullName(), mrID),
				CreatedAt:      &timer,
				UserNotesCount: 2,
				Milestone: &gitlab.Milestone{
					Title: "MilestoneTitle",
				},
			},
		}, nil
	}
	cmdtest.InitTest(m, "mr_view_test")
}

func TestMRView_web_numberArg(t *testing.T) {
	cmd := NewCmdView(f)
	cmdutils.EnableRepoOverride(cmd, f)

	var seenCmd *exec.Cmd
	restoreCmd := run.SetPrepareCmd(func(cmd *exec.Cmd) run.Runnable {
		seenCmd = cmd
		return &mainTest.OutputStub{}
	})
	defer restoreCmd()

	_, err := cmdtest.RunCommand(cmd, "225 -w -R cli-automated-testing/test")
	if err != nil {
		t.Error(err)
		return
	}

	out := stripansi.Strip(stdout.String())
	outErr := stripansi.Strip(stderr.String())
	stdout.Reset()
	stderr.Reset()

	assert.Contains(t, outErr, "Opening gitlab.com/cli-automated-testing/test/-/merge_requests/225 in your browser.")
	assert.Equal(t, out, "")

	if seenCmd == nil {
		t.Log("expected a command to run")
	}
}

func TestMRView(t *testing.T) {
	oldListMrNotes := listMRNotes
	timer, _ := time.Parse(time.RFC3339, "2014-11-12T11:45:26.371Z")
	listMRNotes = func(client *gitlab.Client, projectID any, mrID int64, opts *gitlab.ListMergeRequestNotesOptions) ([]*gitlab.Note, error) {
		if projectID == "PROJECT_MR_WITH_EMPTY_NOTE" {
			return []*gitlab.Note{}, nil
		}
		return []*gitlab.Note{
			{
				ID:    1,
				Body:  "Note Body",
				Title: "Note Title",
				Author: gitlab.NoteAuthor{
					ID:       1,
					Username: "johnwick",
					Name:     "John Wick",
				},
				System:     false,
				CreatedAt:  &timer,
				NoteableID: 0,
			},
			{
				ID:    1,
				Body:  "Marked MR as ready",
				Title: "",
				Author: gitlab.NoteAuthor{
					ID:       1,
					Username: "johnwick",
					Name:     "John Wick",
				},
				System:     true,
				CreatedAt:  &timer,
				NoteableID: 0,
			},
		}, nil
	}

	t.Run("show", func(t *testing.T) {
		cmd := NewCmdView(f)
		cmdutils.EnableRepoOverride(cmd, f)

		cmdOut, err := cmdtest.ExecuteCommand(cmd, "13 -c -s -R cli-automated-testing/test", stdout, stderr)
		require.NoError(t, err)

		out := stripansi.Strip(cmdOut.OutBuf.String())
		outErr := stripansi.Strip(cmdOut.ErrBuf.String())

		require.Contains(t, out, "mrTitle !13")
		require.Equal(t, outErr, "")
		assert.Contains(t, out, "https://gitlab.com/cli-automated-testing/test/-/merge_requests/13")
		assert.Contains(t, out, "johnwick Marked MR as ready")
	})

	t.Run("no_tty", func(t *testing.T) {
		ioStreams.IsaTTY = false
		ioStreams.IsErrTTY = false

		cmd := NewCmdView(f)
		cmdutils.EnableRepoOverride(cmd, f)

		cmdOut, err := cmdtest.ExecuteCommand(cmd, "13 -c -s -R cli-automated-testing/test", stdout, stderr)
		require.NoError(t, err)

		out := stripansi.Strip(cmdOut.OutBuf.String())
		outErr := stripansi.Strip(cmdOut.ErrBuf.String())

		expectedOutputs := []string{
			`title:\tmrTitle`,
			`assignees:\tmona, lisa`,
			`reviewers:\tlisa, mona`,
			`author:\tjdwick`,
			`state:\topen`,
			`comments:\t2`,
			`labels:\ttest, bug`,
			`milestone:\tMilestoneTitle\n`,
			`--`,
			`mrBody`,
		}

		assert.Equal(t, "", outErr)
		t.Helper()
		var r *regexp.Regexp
		for _, l := range expectedOutputs {
			r = regexp.MustCompile(l)
			if !r.MatchString(out) {
				t.Errorf("output did not match regexp /%s/\n> output\n%s\n", r, out)
				return
			}
		}
	})
	listMRNotes = oldListMrNotes
}

func Test_rawMRPreview(t *testing.T) {
	// NOTE: we need to force disable colors, otherwise we'd need ANSI sequences in our test output assertions.
	t.Setenv("NO_COLOR", "true")

	fakeNote1 := &gitlab.Note{}
	fakeNote1.Author.Username = "bob"
	fakeNote2 := &gitlab.Note{}
	fakeNote2.Author.Username = "alice"

	time1, _ := time.Parse(time.RFC3339, "2023-03-09T16:50:20.111Z")
	time2, _ := time.Parse(time.RFC3339, "2023-03-09T16:52:30.222Z")

	mr := &gitlab.MergeRequest{
		BasicMergeRequest: gitlab.BasicMergeRequest{
			IID:            503,
			Title:          "MR title",
			Description:    "MR description",
			State:          "merged",
			Author:         &gitlab.BasicUser{Username: "alice"},
			Labels:         gitlab.Labels{"label1", "label2"},
			Assignees:      []*gitlab.BasicUser{{Username: "alice"}, {Username: "bob"}},
			Reviewers:      []*gitlab.BasicUser{{Username: "john"}, {Username: "paul"}},
			UserNotesCount: 2,
			Milestone:      &gitlab.Milestone{Title: "Some milestone"},
			WebURL:         "https://gitlab.com/OWNER/REPO/-/merge_requests/503",
		},
	}

	notes := []*gitlab.Note{
		{
			System:    true,
			Author:    fakeNote1.Author,
			Body:      "assigned to @alice",
			CreatedAt: &time1,
		},
		{
			System:    false,
			Author:    fakeNote1.Author,
			Body:      "Some comment",
			CreatedAt: &time1,
		},
		{
			System:    false,
			Author:    fakeNote2.Author,
			Body:      "Another comment",
			CreatedAt: &time2,
		},
	}

	ioStreams, _, _, _ = cmdtest.TestIOStreams(cmdtest.WithTestIOStreamsAsTTY(true))

	tests := []struct {
		name  string
		opts  *options
		mr    *gitlab.MergeRequest
		notes []*gitlab.Note
		want  []string
	}{
		{
			"mr_default",
			&options{
				io: ioStreams,
			},
			mr,
			notes,
			[]string{
				"title:\tMR title",
				"state:\tmerged",
				"author:\talice",
				"labels:\tlabel1, label2",
				"assignees:\talice, bob",
				"reviewers:\tjohn, paul",
				"comments:\t2",
				"milestone:\tSome milestone",
				"number:\t503",
				"url:\thttps://gitlab.com/OWNER/REPO/-/merge_requests/503",
				"--",
				"MR description",
			},
		},
		{
			"mr_show_comments_no_comments",
			&options{
				io:             ioStreams,
				showComments:   true,
				showSystemLogs: true,
			},
			mr,
			[]*gitlab.Note{},
			[]string{
				"title:\tMR title",
				"state:\tmerged",
				"author:\talice",
				"labels:\tlabel1, label2",
				"assignees:\talice, bob",
				"reviewers:\tjohn, paul",
				"comments:\t2",
				"milestone:\tSome milestone",
				"number:\t503",
				"url:\thttps://gitlab.com/OWNER/REPO/-/merge_requests/503",
				"--",
				"MR description",
				"\n--\ncomments/notes:\n",
				"There are no comments on this merge request.",
			},
		},
		{
			"mr_with_comments_and_notes",
			&options{
				io:             ioStreams,
				showComments:   true,
				showSystemLogs: true,
			},
			mr,
			notes,
			[]string{
				"title:\tMR title",
				"state:\tmerged",
				"author:\talice",
				"labels:\tlabel1, label2",
				"assignees:\talice, bob",
				"reviewers:\tjohn, paul",
				"comments:\t2",
				"milestone:\tSome milestone",
				"number:\t503",
				"url:\thttps://gitlab.com/OWNER/REPO/-/merge_requests/503",
				"--",
				"MR description",
				"\n--\ncomments/notes:\n",
				fmt.Sprintf("bob assigned to @alice %s", time1),
				"",
				fmt.Sprintf("bob commented %s", time1),
				"Some comment",
				"",
				fmt.Sprintf("alice commented %s", time2),
				"Another comment",
				"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := strings.Join(tt.want, "\n") + "\n"
			got := rawMRPreview(tt.opts, tt.mr, tt.notes)

			require.Equal(t, want, got)
		})
	}
}

func Test_labelsList(t *testing.T) {
	tests := []struct {
		name string
		mr   *gitlab.MergeRequest
		want string
	}{
		{
			"no labels",
			&gitlab.MergeRequest{BasicMergeRequest: gitlab.BasicMergeRequest{
				Labels: gitlab.Labels{},
			}},
			"",
		},
		{
			"one label",
			&gitlab.MergeRequest{BasicMergeRequest: gitlab.BasicMergeRequest{
				Labels: gitlab.Labels{"label1"},
			}},
			"label1",
		},
		{
			"two labels",
			&gitlab.MergeRequest{BasicMergeRequest: gitlab.BasicMergeRequest{
				Labels: gitlab.Labels{"label1", "label2"},
			}},
			"label1, label2",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := labelsList(test.mr)

			if test.want != got {
				t.Errorf(`want "%s"; got "%s"`, test.want, got)
			}
		})
	}
}

func Test_assigneesList(t *testing.T) {
	tests := []struct {
		name string
		mr   *gitlab.MergeRequest
		want string
	}{
		{
			"no assignee",
			&gitlab.MergeRequest{BasicMergeRequest: gitlab.BasicMergeRequest{
				Assignees: []*gitlab.BasicUser{},
			}},
			"",
		},
		{
			"one assignee",
			&gitlab.MergeRequest{BasicMergeRequest: gitlab.BasicMergeRequest{
				Assignees: []*gitlab.BasicUser{{Username: "Alice"}},
			}},
			"Alice",
		},
		{
			"two assignees",
			&gitlab.MergeRequest{BasicMergeRequest: gitlab.BasicMergeRequest{
				Assignees: []*gitlab.BasicUser{{Username: "Alice"}, {Username: "Bob"}},
			}},
			"Alice, Bob",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := assigneesList(test.mr)

			if test.want != got {
				t.Errorf(`want "%s"; got "%s"`, test.want, got)
			}
		})
	}
}

func Test_reviewersList(t *testing.T) {
	tests := []struct {
		name string
		mr   *gitlab.MergeRequest
		want string
	}{
		{
			"no assignee",
			&gitlab.MergeRequest{BasicMergeRequest: gitlab.BasicMergeRequest{
				Reviewers: []*gitlab.BasicUser{},
			}},
			"",
		},
		{
			"one assignee",
			&gitlab.MergeRequest{BasicMergeRequest: gitlab.BasicMergeRequest{
				Reviewers: []*gitlab.BasicUser{{Username: "Alice"}},
			}},
			"Alice",
		},
		{
			"two assignees",
			&gitlab.MergeRequest{BasicMergeRequest: gitlab.BasicMergeRequest{
				Reviewers: []*gitlab.BasicUser{{Username: "Alice"}, {Username: "Bob"}},
			}},
			"Alice, Bob",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := reviewersList(test.mr)

			if test.want != got {
				t.Errorf(`want "%s"; got "%s"`, test.want, got)
			}
		})
	}
}

func TestMrViewJSON(t *testing.T) {
	cmd := NewCmdView(f)
	stdout.Reset()
	stderr.Reset()

	output, err := cmdtest.ExecuteCommand(cmd, "1 -F json", stdout, stderr)
	if err != nil {
		t.Errorf("error running command `mr view 1 -F json`: %v", err)
	}

	assert.True(t, json.Valid([]byte(output.String())))
	assert.Empty(t, output.Stderr())
}

func TestPrintCommentFileContext(t *testing.T) {
	// NOTE: we need to force disable colors, otherwise we'd need ANSI sequences in our test output assertions.
	t.Setenv("NO_COLOR", "true")

	ioStr, _, _, _ := cmdtest.TestIOStreams(cmdtest.WithTestIOStreamsAsTTY(true))
	c := ioStr.Color()

	tests := []struct {
		name     string
		note     *gitlab.Note
		expected string
	}{
		{
			name: "single line comment on new file",
			note: &gitlab.Note{
				Position: &gitlab.NotePosition{
					NewPath: "internal/commands/mr/view/mr_view.go",
					NewLine: 42,
				},
			},
			expected: " on internal/commands/mr/view/mr_view.go:42\n",
		},
		{
			name: "single line comment on old file",
			note: &gitlab.Note{
				Position: &gitlab.NotePosition{
					OldPath: "internal/commands/mr/view/mr_view.go",
					OldLine: 35,
				},
			},
			expected: " on internal/commands/mr/view/mr_view.go:35\n",
		},
		{
			name: "multi-line comment",
			note: &gitlab.Note{
				Position: &gitlab.NotePosition{
					NewPath: "internal/gateway/mcp/tools/get_coin_open_interest.go",
					LineRange: &gitlab.LineRange{
						StartRange: &gitlab.LinePosition{NewLine: 63},
						EndRange:   &gitlab.LinePosition{NewLine: 68},
					},
				},
			},
			expected: " on internal/gateway/mcp/tools/get_coin_open_interest.go:63-68\n",
		},
		{
			name: "single line range (same start and end)",
			note: &gitlab.Note{
				Position: &gitlab.NotePosition{
					NewPath: "main.go",
					LineRange: &gitlab.LineRange{
						StartRange: &gitlab.LinePosition{NewLine: 10},
						EndRange:   &gitlab.LinePosition{NewLine: 10},
					},
				},
			},
			expected: " on main.go:10\n",
		},
		{
			name: "position with no line numbers",
			note: &gitlab.Note{
				Position: &gitlab.NotePosition{
					NewPath: "file.go",
					NewLine: 0,
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			printCommentFileContext(&buf, c, tt.note.Position)
			got := buf.String()
			assert.Equal(t, tt.expected, got)
		})
	}
}

func Test_printTTYMRPreview_closedMRWithNilClosedBy(t *testing.T) {
	// NOTE: we need to force disable colors, otherwise we'd need ANSI sequences in our test output assertions.
	t.Setenv("NO_COLOR", "true")

	ioStreams, _, stdout, _ := cmdtest.TestIOStreams(cmdtest.WithTestIOStreamsAsTTY(true))

	createdTime, _ := time.Parse(time.RFC3339, "2024-01-01T12:00:00Z")

	mr := &gitlab.MergeRequest{
		BasicMergeRequest: gitlab.BasicMergeRequest{
			IID:         505,
			Title:       "Test closed MR",
			Description: "Test description",
			State:       "closed", // Now test closed MR with nil ClosedBy
			Author:      &gitlab.BasicUser{Username: "testuser"},
			WebURL:      "https://gitlab.com/OWNER/REPO/-/merge_requests/505",
			CreatedAt:   &createdTime,
			ClosedBy:    nil,
		},
	}

	opts := &options{
		io:             ioStreams,
		showComments:   false,
		showSystemLogs: false,
	}

	// This should not panic - the bug would cause a nil pointer dereference here
	printTTYMRPreview(opts, mr, nil, []*gitlab.Note{})
	output := stdout.String()

	// Verify that it contains "Closed" but not "Closed by:" since ClosedBy is nil
	assert.Contains(t, output, "Closed")
	assert.NotContains(t, output, "Closed by:")
}
