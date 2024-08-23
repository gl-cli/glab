package view

import (
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"

	"gitlab.com/gitlab-org/cli/pkg/httpmock"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/run"
	"gitlab.com/gitlab-org/cli/test"
)

func assertScreen(t *testing.T, screen tcell.Screen, expected []string) {
	sx, sy := screen.Size()
	assert.Equal(t, len(expected), sy)
	assert.Equal(t, len([]rune(expected[0])), sx)
	actual := make([]string, sy)
	for y, str := range expected {
		runes := make([]rune, len(str))
		row := []rune(str)
		for x, expectedRune := range row {
			r, _, _, _ := screen.GetContent(x, y)
			runes[x] = r
			_ = expectedRune
			// assert.Equal(t, expectedRune, r, "%s != %s at (%d,%d)",
			//	strconv.QuoteRune(expectedRune), strconv.QuoteRune(r), x, y)
		}

		actual[y] = strings.TrimRight(string(runes), string('\x00'))
		assert.Equal(t, str, actual[y])
	}
	t.Logf("Expected w: %d l: %d", len([]rune(expected[0])), len(expected))
	for _, str := range expected {
		t.Log(str)
	}
	t.Logf("Actual w: %d l: %d", sx, sy)
	for _, str := range actual {
		t.Log(str)
	}
}

func Test_line(t *testing.T) {
	tests := []struct {
		desc     string
		lineF    func(screen tcell.Screen, x, y, l int)
		x, y, l  int
		expected []string
	}{
		{
			"hline",
			hline,
			2, 2, 5,
			[]string{
				"          ",
				"          ",
				"  ═════   ",
				"          ",
				"          ",
				"          ",
				"          ",
				"          ",
				"          ",
				"          ",
			},
		},
		{
			"hline overflow",
			hline,
			2, 2, 10,
			[]string{
				"          ",
				"          ",
				"  ════════",
				"          ",
				"          ",
				"          ",
				"          ",
				"          ",
				"          ",
				"          ",
			},
		},
		{
			"vline",
			vline,
			2, 2, 5,
			[]string{
				"          ",
				"          ",
				"  ║       ",
				"  ║       ",
				"  ║       ",
				"  ║       ",
				"  ║       ",
				"          ",
				"          ",
				"          ",
			},
		},
		{
			"vline overflow",
			vline,
			2, 2, 10,
			[]string{
				"          ",
				"          ",
				"  ║       ",
				"  ║       ",
				"  ║       ",
				"  ║       ",
				"  ║       ",
				"  ║       ",
				"  ║       ",
				"  ║       ",
			},
		},
	}

	for _, test := range tests {
		screen := tcell.NewSimulationScreen("UTF-8")
		err := screen.Init()
		if err != nil {
			t.Fatal(err)
		}
		// Set screen to matrix size
		screen.SetSize(len(test.expected), len(test.expected[0]))

		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			test.lineF(screen, test.x, test.y, test.l)
			screen.Show()
			assertScreen(t, screen, test.expected)
		})
	}
}

func testbox(x, y, w, h int) *tview.TextView {
	b := tview.NewTextView()
	b.
		SetBackgroundColor(tcell.ColorDefault).
		SetBorder(true).
		SetRect(x, y, w, h)
	return b
}

func Test_Link(t *testing.T) {
	tests := []struct {
		desc        string
		b1, b2      *tview.Box
		first, last bool
		expected    []string
	}{
		{
			"first stage",
			testbox(2, 1, 3, 3).Box, testbox(2, 5, 3, 3).Box,
			true, false,
			[]string{
				"          ",
				"  ┌─┐     ",
				"  │ │     ",
				"  └─┘ ║   ",
				"      ║   ",
				"  ┌─┐ ║   ",
				"  │ │═╝   ",
				"  └─┘     ",
				"          ",
				"          ",
			},
		},
		{
			"last stage",
			testbox(5, 1, 3, 3).Box, testbox(5, 5, 3, 3).Box,
			false, true,
			[]string{
				"          ",
				"     ┌─┐  ",
				"   ╦ │ │  ",
				"   ║ └─┘  ",
				"   ║      ",
				"   ║ ┌─┐  ",
				"   ╚═│ │  ",
				"     └─┘  ",
				"          ",
				"          ",
			},
		},
		{
			"cross stage",
			testbox(1, 1, 3, 3).Box, testbox(7, 1, 3, 3).Box,
			false, false,
			[]string{
				"          ",
				" ┌─┐   ┌─┐",
				" │ │═══│ │",
				" └─┘   └─┘",
				"          ",
				"          ",
				"          ",
				"          ",
				"          ",
				"          ",
			},
		},
	}

	for _, test := range tests {
		screen := tcell.NewSimulationScreen("UTF-8")
		err := screen.Init()
		if err != nil {
			t.Fatal(err)
		}
		// Set screen to matrix size
		screen.SetSize(len(test.expected), len(test.expected[0]))

		test.b1.Draw(screen)
		test.b2.Draw(screen)

		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			link(screen, test.b1, test.b2, 2, test.first, test.last)
			screen.Show()
			assertScreen(t, screen, test.expected)
		})
	}
}

func Test_LinkJobs(t *testing.T) {
	expected := []string{
		"                 ",
		" ┌─┐   ┌─┐   ┌─┐ ",
		" │ │╦═╦│ │╦═╦│ │ ",
		" └─┘║ ║└─┘║ ║└─┘ ",
		"    ║ ║   ║ ║    ",
		" ┌─┐║ ║┌─┐║ ║┌─┐ ",
		" │ │╝ ╠│ │╝ ╚│ │ ",
		" └─┘║ ║└─┘║  └─┘ ",
		"    ║ ║   ║      ",
		" ┌─┐║ ║┌─┐║      ",
		" │ │╝ ╚│ │╝      ",
		" └─┘║  └─┘       ",
		"    ║            ",
		" ┌─┐║            ",
		" │ │╝            ",
		" └─┘             ",
		"                 ",
	}
	jobs := []*ViewJob{
		{
			Name:  "stage1-job1",
			Stage: "stage1",
			Kind:  Job,
		},
		{
			Name:  "stage1-job2",
			Stage: "stage1",
			Kind:  Job,
		},
		{
			Name:  "stage1-job3",
			Stage: "stage1",
			Kind:  Job,
		},
		{
			Name:  "stage1-job4",
			Stage: "stage1",
			Kind:  Job,
		},
		{
			Name:  "stage2-job1",
			Stage: "stage2",
			Kind:  Bridge,
		},
		{
			Name:  "stage2-job2",
			Stage: "stage2",
			Kind:  Job,
		},
		{
			Name:  "stage2-job3",
			Stage: "stage2",
			Kind:  Job,
		},
		{
			Name:  "stage3-job1",
			Stage: "stage3",
			Kind:  Job,
		},
		{
			Name:  "stage3-job2",
			Stage: "stage3",
			Kind:  Job,
		},
	}
	boxes := map[string]*tview.TextView{
		"jobs-stage1-job1": testbox(1, 1, 3, 3),
		"jobs-stage1-job2": testbox(1, 5, 3, 3),
		"jobs-stage1-job3": testbox(1, 9, 3, 3),
		"jobs-stage1-job4": testbox(1, 13, 3, 3),

		"jobs-stage2-job1": testbox(7, 1, 3, 3),
		"jobs-stage2-job2": testbox(7, 5, 3, 3),
		"jobs-stage2-job3": testbox(7, 9, 3, 3),

		"jobs-stage3-job1": testbox(13, 1, 3, 3),
		"jobs-stage3-job2": testbox(13, 5, 3, 3),
	}

	screen := tcell.NewSimulationScreen("UTF-8")
	err := screen.Init()
	if err != nil {
		t.Fatal(err)
	}
	// Set screen to matrix size
	screen.SetSize(len(expected), len(expected[0]))

	for _, b := range boxes {
		b.Draw(screen)
	}

	err = linkJobs(screen, jobs, boxes)
	if err != nil {
		t.Fatal(err)
	}

	screen.Show()
	assertScreen(t, screen, expected)
}

func Test_LinkJobsNegative(t *testing.T) {
	tests := []struct {
		desc  string
		jobs  []*ViewJob
		boxes map[string]*tview.TextView
	}{
		{
			"determinePadding -- first job missing",
			[]*ViewJob{
				{
					Name:  "stage1-job1",
					Stage: "stage1",
					Kind:  Job,
				},
			},
			map[string]*tview.TextView{
				"jobs-stage2-job1": testbox(1, 5, 3, 3),
				"jobs-stage2-job2": testbox(1, 9, 3, 3),
			},
		},
		{
			"determinePadding -- second job missing",
			[]*ViewJob{
				{
					Name:  "stage1-job1",
					Stage: "stage1",
					Kind:  Job,
				},
				{
					Name:  "stage2-job1",
					Stage: "stage2",
					Kind:  Job,
				},
				{
					Name:  "stage2-job2",
					Stage: "stage2",
					Kind:  Job,
				},
			},
			map[string]*tview.TextView{
				"jobs-stage1-job1": testbox(1, 1, 3, 3),
				"jobs-stage2-job2": testbox(1, 9, 3, 3),
			},
		},
		{
			"Link -- third job missing",
			[]*ViewJob{
				{
					Name:  "stage1-job1",
					Stage: "stage1",
					Kind:  Job,
				},
				{
					Name:  "stage2-job1",
					Stage: "stage2",
					Kind:  Job,
				},
				{
					Name:  "stage2-job2",
					Stage: "stage2",
					Kind:  Job,
				},
			},
			map[string]*tview.TextView{
				"jobs-stage1-job1": testbox(1, 1, 3, 3),
				"jobs-stage2-job1": testbox(1, 5, 3, 3),
			},
		},
		{
			"Link -- third job missing",
			[]*ViewJob{
				{
					Name:  "stage1-job1",
					Stage: "stage1",
					Kind:  Job,
				},
				{
					Name:  "stage2-job1",
					Stage: "stage2",
					Kind:  Job,
				},
				{
					Name:  "stage2-job2",
					Stage: "stage2",
					Kind:  Job,
				},
			},
			map[string]*tview.TextView{
				"jobs-stage1-job1": testbox(1, 1, 3, 3),
				"jobs-stage2-job1": testbox(1, 5, 3, 3),
			},
		},
	}
	for _, test := range tests {
		screen := tcell.NewSimulationScreen("UTF-8")
		err := screen.Init()
		if err != nil {
			t.Fatal(err)
		}
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			assert.Error(t, linkJobs(screen, test.jobs, test.boxes))
		})
	}
}

func Test_jobsView(t *testing.T) {
	expected := []string{
		"  ┌────────────────────┐      ┌────────────────────┐      ┌────────────────────┐        ",
		"  │       Stage1       │      │       Stage2       │      │       Stage3       │        ",
		"  └────────────────────┘      └────────────────────┘      └────────────────────┘        ",
		"                                                                                        ",
		"  ╔✔ stage1-job1-reall…╗      ┌───● stage2-job1────┐      ┌───■ stage3-job1────┐        ",
		"  ║                    ║      │                    │      │                    │        ",
		"  ║             01m 01s║═╦══╦═│                    │═╦══╦═│                    │        ",
		"  ╚════════════════════╝ ║  ║ └────────────────────┘ ║  ║ └────────────────────┘        ",
		"                         ║  ║                        ║  ║                               ",
		"  ┌───✔ stage1-job2────┐ ║  ║ ┌───● stage2-job2────┐ ║  ║ ┌───■ stage3-job2────┐        ",
		"  │                    │ ║  ║ │                    │ ║  ║ │                   »│        ",
		"  │                    │═╝  ╠═│                    │═╝  ╚═│                    │        ",
		"  └────────────────────┘ ║  ║ └────────────────────┘ ║    └────────────────────┘        ",
		"                         ║  ║                        ║                                  ",
		"  ┌───✔ stage1-job3────┐ ║  ║ ┌───● stage2-job3────┐ ║                                  ",
		"  │                    │ ║  ║ │                    │ ║                                  ",
		"  │                    │═╝  ╚═│                    │═╝                                  ",
		"  └────────────────────┘ ║    └────────────────────┘                                    ",
		"                         ║                                                              ",
		"  ┌───✘ stage1-job4────┐ ║                                                              ",
		"  │                    │ ║                                                              ",
		"  │                    │═╝                                                              ",
		"  └────────────────────┘                                                                ",
		"                                                                                        ",
		"                                                                                        ",
		"                                                                                        ",
	}
	now := time.Now()
	past := now.Add(time.Second * -61)
	jobs := []*ViewJob{
		{
			Name:       "stage1-job1-really-long",
			Stage:      "stage1",
			Status:     "success",
			StartedAt:  &past, // relies on test running in <1s we'll see how it goes
			FinishedAt: &now,
		},
		{
			Name:   "stage1-job2",
			Stage:  "stage1",
			Status: "success",
		},
		{
			Name:   "stage1-job3",
			Stage:  "stage1",
			Status: "success",
		},
		{
			Name:   "stage1-job4",
			Stage:  "stage1",
			Status: "failed",
		},
		{
			Name:   "stage2-job1",
			Stage:  "stage2",
			Status: "running",
		},
		{
			Name:   "stage2-job2",
			Stage:  "stage2",
			Status: "running",
		},
		{
			Name:   "stage2-job3",
			Stage:  "stage2",
			Status: "pending",
		},
		{
			Name:   "stage3-job1",
			Stage:  "stage3",
			Status: "manual",
		},
		{
			Name:   "stage3-job2",
			Stage:  "stage3",
			Status: "manual",
			Kind:   Bridge,
		},
	}

	boxes = make(map[string]*tview.TextView)
	jobsCh := make(chan []*ViewJob)
	inputCh := make(chan struct{})
	root := tview.NewPages()
	root.
		SetBackgroundColor(tcell.ColorDefault).
		SetBorderPadding(1, 1, 2, 2)

	screen := tcell.NewSimulationScreen("UTF-8")
	err := screen.Init()
	if err != nil {
		t.Fatal(err)
	}
	// Set screen to matrix size
	screen.SetSize(len([]rune(expected[0])), len(expected))
	w, h := screen.Size()
	root.SetRect(0, 0, w, h)

	go func() {
		jobsCh <- jobs
	}()
	root.Box.Focus(nil)
	jobsView(nil, jobsCh, inputCh, root, ViewOpts{})
	root.Focus(func(p tview.Primitive) { p.Focus(nil) })
	root.Draw(screen)
	linkJobsView(nil)(screen)
	screen.Sync()
	assertScreen(t, screen, expected)
}

func Test_latestJobs(t *testing.T) {
	tests := []struct {
		desc     string
		jobs     []*ViewJob
		expected []*ViewJob
	}{
		{
			desc: "no newer jobs",
			jobs: []*ViewJob{
				{
					ID:    1,
					Name:  "stage1-job1",
					Stage: "stage1",
				},
				{
					ID:    2,
					Name:  "stage1-job2",
					Stage: "stage1",
				},
				{
					ID:    3,
					Name:  "stage1-job3",
					Stage: "stage1",
				},
			},
			expected: []*ViewJob{
				{
					ID:    1,
					Name:  "stage1-job1",
					Stage: "stage1",
				},
				{
					ID:    2,
					Name:  "stage1-job2",
					Stage: "stage1",
				},
				{
					ID:    3,
					Name:  "stage1-job3",
					Stage: "stage1",
				},
			},
		},
		{
			desc: "1 newer",
			jobs: []*ViewJob{
				{
					ID:    1,
					Name:  "stage1-job1",
					Stage: "stage1",
				},
				{
					ID:    2,
					Name:  "stage1-job2",
					Stage: "stage1",
				},
				{
					ID:    3,
					Name:  "stage1-job3",
					Stage: "stage1",
				},
				{
					ID:    4,
					Name:  "stage1-job1",
					Stage: "stage1",
				},
			},
			expected: []*ViewJob{
				{
					ID:    4,
					Name:  "stage1-job1",
					Stage: "stage1",
				},
				{
					ID:    2,
					Name:  "stage1-job2",
					Stage: "stage1",
				},
				{
					ID:    3,
					Name:  "stage1-job3",
					Stage: "stage1",
				},
			},
		},
		{
			desc: "2 newer",
			jobs: []*ViewJob{
				{
					ID:    1,
					Name:  "stage1-job1",
					Stage: "stage1",
				},
				{
					ID:    2,
					Name:  "stage1-job2",
					Stage: "stage1",
				},
				{
					ID:    3,
					Name:  "stage1-job3",
					Stage: "stage1",
				},
				{
					ID:    4,
					Name:  "stage1-job3",
					Stage: "stage1",
				},
				{
					ID:    5,
					Name:  "stage1-job1",
					Stage: "stage1",
				},
			},
			expected: []*ViewJob{
				{
					ID:    5,
					Name:  "stage1-job1",
					Stage: "stage1",
				},
				{
					ID:    2,
					Name:  "stage1-job2",
					Stage: "stage1",
				},
				{
					ID:    4,
					Name:  "stage1-job3",
					Stage: "stage1",
				},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			jobs := latestJobs(test.jobs)
			assert.Equal(t, test.expected, jobs)
		})
	}
}

func Test_adjacentStages(t *testing.T) {
	tests := []struct {
		desc                       string
		stage                      string
		jobs                       []*ViewJob
		expectedPrev, expectedNext string
	}{
		{
			"no jobs",
			"1",
			[]*ViewJob{},
			"", "",
		},
		{
			"first stage",
			"1",
			[]*ViewJob{
				{
					Stage: "1",
				},
				{
					Stage: "1",
				},
				{
					Stage: "1",
				},
				{
					Stage: "2",
				},
			},
			"1", "2",
		},
		{
			"mid stage",
			"2",
			[]*ViewJob{
				{
					Stage: "1",
				},
				{
					Stage: "1",
				},
				{
					Stage: "1",
				},
				{
					Stage: "2",
				},
				{
					Stage: "2",
				},
				{
					Stage: "3",
				},
			},
			"1", "3",
		},
		{
			"last stage",
			"3",
			[]*ViewJob{
				{
					Stage: "1",
				},
				{
					Stage: "1",
				},
				{
					Stage: "1",
				},
				{
					Stage: "2",
				},
				{
					Stage: "2",
				},
				{
					Stage: "3",
				},
			},
			"2", "3",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			prev, next := adjacentStages(test.jobs, test.stage)
			assert.Equal(t, test.expectedPrev, prev)
			assert.Equal(t, test.expectedNext, next)
		})
	}
}

func Test_stageBounds(t *testing.T) {
	tests := []struct {
		desc                         string
		stage                        string
		jobs                         []*ViewJob
		expectedLower, expectedUpper int
	}{
		{
			"no jobs",
			"1",
			[]*ViewJob{},
			0, 0,
		},
		{
			"first stage",
			"1",
			[]*ViewJob{
				{
					Stage: "1",
				},
				{
					Stage: "1",
				},
				{
					Stage: "1",
				},
				{
					Stage: "2",
				},
			},
			0, 2,
		},
		{
			"mid stage",
			"2",
			[]*ViewJob{
				{
					Stage: "1",
				},
				{
					Stage: "1",
				},
				{
					Stage: "1",
				},
				{
					Stage: "2",
				},
				{
					Stage: "2",
				},
				{
					Stage: "3",
				},
			},
			3, 4,
		},
		{
			"last stage",
			"3",
			[]*ViewJob{
				{
					Stage: "1",
				},
				{
					Stage: "1",
				},
				{
					Stage: "1",
				},
				{
					Stage: "2",
				},
				{
					Stage: "2",
				},
				{
					Stage: "3",
				},
			},
			5, 5,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			lower, upper := stageBounds(test.jobs, test.stage)
			assert.Equal(t, test.expectedLower, lower)
			assert.Equal(t, test.expectedUpper, upper)
		})
	}
}

func Test_handleNavigation(t *testing.T) {
	jobs := []*ViewJob{
		{
			Name:   "stage1-job1-really-long",
			Stage:  "stage1",
			Status: "success",
		},
		{
			Name:   "stage1-job2",
			Stage:  "stage1",
			Status: "success",
		},
		{
			Name:   "stage1-job3",
			Stage:  "stage1",
			Status: "success",
		},
		{
			Name:   "stage1-job4",
			Stage:  "stage1",
			Status: "failed",
		},
		{
			Name:   "stage2-job1",
			Stage:  "stage2",
			Status: "running",
		},
		{
			Name:   "stage2-job2",
			Stage:  "stage2",
			Status: "running",
		},
		{
			Name:   "stage2-job3",
			Stage:  "stage2",
			Status: "pending",
		},
		{
			Name:   "stage3-job1",
			Stage:  "stage3",
			Status: "manual",
		},
		{
			Name:   "stage3-job2",
			Stage:  "stage3",
			Status: "manual",
		},
	}
	tests := []struct {
		desc     string
		input    []*tcell.EventKey
		expected int
	}{
		{
			"do nothing",
			[]*tcell.EventKey{},
			0,
		},
		{
			"arrows down",
			[]*tcell.EventKey{
				tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
			},
			3,
		},
		{
			"arrows down bottom boundary",
			[]*tcell.EventKey{
				tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
			},
			3,
		},
		{
			"arrows down bottom middle boundary",
			[]*tcell.EventKey{
				tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone),
			},
			6,
		},
		{
			"arrows down last (3rd) column bottom boundary",
			[]*tcell.EventKey{
				tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone),
			},
			8,
		},
		{
			"arrows down persistent depth bottom boundary",
			[]*tcell.EventKey{
				tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone),
			},
			3,
		},
		{
			"arrows left boundary",
			[]*tcell.EventKey{
				tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone),
			},
			0,
		},
		{
			"arrows up boundary",
			[]*tcell.EventKey{
				tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone),
			},
			0,
		},
		{
			"arrows right boundary",
			[]*tcell.EventKey{
				tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone),
			},
			7,
		},
		{
			"hjkl down",
			[]*tcell.EventKey{
				tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone),
			},
			3,
		},
		{
			"hjkl down bottom boundary",
			[]*tcell.EventKey{
				tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone),
			},
			3,
		},
		{
			"hjkl down bottom middle boundary",
			[]*tcell.EventKey{
				tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'l', tcell.ModNone),
			},
			6,
		},
		{
			"hjkl down last (3rd) column bottom boundary",
			[]*tcell.EventKey{
				tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'l', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'l', tcell.ModNone),
			},
			8,
		},
		{
			"hjkl down persistent depth bottom boundary",
			[]*tcell.EventKey{
				tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'l', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'l', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'h', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'h', tcell.ModNone),
			},
			3,
		},
		{
			"hjkl left boundary",
			[]*tcell.EventKey{
				tcell.NewEventKey(tcell.KeyRune, 'h', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'h', tcell.ModNone),
			},
			0,
		},
		{
			"hjkl up boundary",
			[]*tcell.EventKey{
				tcell.NewEventKey(tcell.KeyRune, 'k', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'k', tcell.ModNone),
			},
			0,
		},
		{
			"hjkl right boundary",
			[]*tcell.EventKey{
				tcell.NewEventKey(tcell.KeyRune, 'l', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'l', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'l', tcell.ModNone),
			},
			7,
		},
		{
			"G boundary",
			[]*tcell.EventKey{
				tcell.NewEventKey(tcell.KeyRune, 'G', tcell.ModNone),
			},
			3,
		},
		{
			"Gg boundary",
			[]*tcell.EventKey{
				tcell.NewEventKey(tcell.KeyRune, 'G', tcell.ModNone),
				tcell.NewEventKey(tcell.KeyRune, 'g', tcell.ModNone),
			},
			0,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			var navi navigator
			for _, e := range test.input {
				navi.Navigate(jobs, e)
			}
			assert.Equal(t, test.expected, navi.idx)
		})
	}
}

func runCommand(rt http.RoundTripper, cli string) (*test.CmdOut, error, func()) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(true, "")

	factory := cmdtest.InitFactory(ios, rt)

	_, _ = factory.HttpClient()

	cmd := NewCmdView(factory)

	restoreCmd := run.SetPrepareCmd(func(cmd *exec.Cmd) run.Runnable {
		return &test.OutputStub{}
	})

	cmdOut, err := cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)

	return cmdOut, err, restoreCmd
}

func TestCIView(t *testing.T) {
	type httpMock struct {
		method string
		path   string
		status int
		body   string
	}

	tests := []struct {
		name      string
		cli       string
		httpMocks []httpMock

		expectedOutput string
	}{
		{
			name: "view ci pipeline on web for a given branch",
			cli:  "--web --branch foo",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"https://gitlab.com/api/v4/projects/OWNER%2FREPO/repository/commits/foo",
					http.StatusOK,
					`{
						"id": "6104942438c14ec7bd21c6cd5bd995272b3faff6",
						"last_pipeline": {
							"id": 8,
							"ref": "main",
							"sha": "2dc6aa325a317eda67812f05600bdf0fcdc70ab0",
							"status": "created",
							"web_url": "https://gitlab.com/OWNER/REPO/-/pipelines/225"
						},
						"status": "running"
					}`,
				},
			},
			expectedOutput: "Opening gitlab.com/OWNER/REPO/-/pipelines/225 in your browser.\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.FullURL,
			}
			defer fakeHTTP.Verify(t)

			for _, mock := range tc.httpMocks {
				fakeHTTP.RegisterResponder(mock.method, mock.path, httpmock.NewStringResponse(mock.status, mock.body))
			}

			output, err, restoreCmd := runCommand(fakeHTTP, tc.cli)
			defer restoreCmd()

			if assert.NoErrorf(t, err, "error running command `ci view %s`: %v", tc.cli, err) {
				assert.Empty(t, output.String())
				assert.Equal(t, tc.expectedOutput, output.Stderr())

			}
		})
	}
}
