package view

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/ci/ciutils"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/git"
	"gitlab.com/gitlab-org/cli/pkg/utils"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/gdamore/tcell/v2"
	"github.com/lunixbochs/vtclean"
	"github.com/pkg/errors"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type ViewOpts struct {
	RefName string

	ProjectID string
	Commit    *gitlab.Commit
	CommitSHA string
	ApiClient *gitlab.Client
	Output    io.Writer

	OpenInBrowser bool
	PipelineUser  *gitlab.BasicUser
}

type ViewJobKind int64

const (
	Job ViewJobKind = iota
	Bridge
)

type ViewJob struct {
	ID           int        `json:"id"`
	Name         string     `json:"name"`
	StartedAt    *time.Time `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	ErasedAt     *time.Time `json:"erased_at"`
	Duration     float64    `json:"duration"`
	Stage        string     `json:"stage"`
	Status       string     `json:"status"`
	AllowFailure bool       `json:"allow_failure"`

	Kind ViewJobKind

	OriginalJob    *gitlab.Job
	OriginalBridge *gitlab.Bridge
}

func ViewJobFromBridge(bridge *gitlab.Bridge) *ViewJob {
	vj := &ViewJob{}
	vj.ID = bridge.ID
	vj.Name = bridge.Name
	vj.Status = bridge.Status
	vj.Stage = bridge.Stage
	vj.StartedAt = bridge.StartedAt
	vj.FinishedAt = bridge.FinishedAt
	vj.ErasedAt = bridge.ErasedAt
	vj.Duration = bridge.Duration
	vj.AllowFailure = bridge.AllowFailure
	vj.OriginalBridge = bridge
	vj.Kind = Bridge
	return vj
}

func ViewJobFromJob(job *gitlab.Job) *ViewJob {
	vj := &ViewJob{}
	vj.ID = job.ID
	vj.Name = job.Name
	vj.Status = job.Status
	vj.Stage = job.Stage
	vj.StartedAt = job.StartedAt
	vj.FinishedAt = job.FinishedAt
	vj.ErasedAt = job.ErasedAt
	vj.Duration = job.Duration
	vj.AllowFailure = job.AllowFailure
	vj.OriginalJob = job
	vj.Kind = Job
	return vj
}

func NewCmdView(f *cmdutils.Factory) *cobra.Command {
	opts := ViewOpts{}
	pipelineCIView := &cobra.Command{
		Use:   "view [branch/tag]",
		Short: "View, run, trace, log, and cancel CI/CD job's current pipeline.",
		Long: heredoc.Doc(`Supports viewing, running, tracing, and canceling jobs.

		Use arrow keys to navigate jobs and logs.

		- 'Enter' to toggle through a job's logs / traces, or display a child pipeline. Trigger jobs are marked with a '»'.
		- 'Esc' or 'q' to close the logs or trace, or return to the parent pipeline.
		- 'Ctrl+R', 'Ctrl+P' to run, retry, or play a job. Use 'Tab' or arrow keys to navigate the modal, and 'Enter' to confirm.
		- 'Ctrl+D' to cancel a job. If the selected job isn't running or pending, quits the CI/CD view.
		- 'Ctrl+Q' to quit the CI/CD view.
		- 'Ctrl+Space' to suspend application and view the logs. Similar to 'glab pipeline ci trace'.
		Supports vi style bindings and arrow keys for navigating jobs and logs.
	`),
		Example: heredoc.Doc(`
	glab pipeline ci view   # Uses current branch
	glab pipeline ci view master  # Get latest pipeline on master branch
	glab pipeline ci view -b master  # just like the second example
	glab pipeline ci view -b master -R profclems/glab  # Get latest pipeline on master branch of profclems/glab repo
	`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Output = f.IO.StdOut

			var err error
			opts.ApiClient, err = f.HttpClient()
			if err != nil {
				return err
			}

			repo, err := f.BaseRepo()
			if err != nil {
				return err
			}

			opts.ProjectID = repo.FullName()

			if opts.RefName == "" {
				if len(args) == 1 {
					opts.RefName = args[0]
				} else {
					opts.RefName, err = git.CurrentBranch()
					if err != nil {
						return err
					}
				}
			}

			opts.Commit, err = api.GetCommit(opts.ApiClient, opts.ProjectID, opts.RefName)
			if err != nil {
				return err
			}

			opts.CommitSHA = opts.Commit.ID
			if opts.Commit.LastPipeline == nil {
				return fmt.Errorf("Can't find pipeline for commit: %s", opts.CommitSHA)
			}

			cfg, _ := f.Config()

			if opts.OpenInBrowser { // open in browser if --web flag is specified
				webURL := opts.Commit.LastPipeline.WebURL

				if f.IO.IsOutputTTY() {
					fmt.Fprintf(f.IO.StdErr, "Opening %s in your browser.\n", utils.DisplayURL(webURL))
				}

				browser, _ := cfg.Get(repo.RepoHost(), "browser")
				return utils.OpenInBrowser(webURL, browser)
			}

			p, err := api.GetSinglePipeline(opts.ApiClient, opts.Commit.LastPipeline.ID, opts.ProjectID)
			if err != nil {
				return fmt.Errorf("Can't get pipeline #%d info: %s", opts.Commit.LastPipeline.ID, err)
			}
			opts.PipelineUser = p.User
			pipelines = make([]gitlab.PipelineInfo, 0, 10)

			return drawView(opts)
		},
	}

	pipelineCIView.Flags().
		StringVarP(&opts.RefName, "branch", "b", "", "Check pipeline status for a branch or tag. Defaults to the current branch.")
	pipelineCIView.Flags().BoolVarP(&opts.OpenInBrowser, "web", "w", false, "Open pipeline in a browser. Uses default browser, or browser specified in BROWSER variable.")

	return pipelineCIView
}

func drawView(opts ViewOpts) error {
	root := tview.NewPages()
	root.
		SetBackgroundColor(tcell.ColorDefault).
		SetBorderPadding(1, 1, 2, 2).
		SetBorder(true).
		SetTitle(fmt.Sprintf(" Pipeline #%d triggered %s by %s ", opts.Commit.LastPipeline.ID, utils.TimeToPrettyTimeAgo(*opts.Commit.LastPipeline.CreatedAt), opts.PipelineUser.Name))

	boxes = make(map[string]*tview.TextView)
	jobsCh := make(chan []*ViewJob)
	forceUpdateCh := make(chan bool)
	inputCh := make(chan struct{})

	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	app := tview.NewApplication()
	defer recoverPanic(app)

	var navi navigator
	app.SetInputCapture(inputCapture(app, root, navi, inputCh, forceUpdateCh, opts))
	go updateJobs(app, jobsCh, forceUpdateCh, opts)
	go func() {
		defer recoverPanic(app)
		for {
			app.SetFocus(root)
			jobsView(app, jobsCh, inputCh, root, opts)
			app.Draw()
		}
	}()
	if err := app.SetScreen(screen).SetRoot(root, true).SetAfterDrawFunc(linkJobsView(app)).Run(); err != nil {
		return err
	}
	return nil
}

func inputCapture(
	app *tview.Application,
	root *tview.Pages,
	navi navigator,
	inputCh chan struct{},
	forceUpdateCh chan bool,
	opts ViewOpts,
) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' || event.Key() == tcell.KeyEscape {
			switch {
			case modalVisible:
				modalVisible = !modalVisible
				root.HidePage("yesno")
				if inputCh == nil {
					inputCh <- struct{}{}
				}
			case logsVisible:
				logsVisible = !logsVisible
				root.HidePage("logs-" + curJob.Name)
				if inputCh == nil {
					inputCh <- struct{}{}
				}
				app.ForceDraw()
			case len(pipelines) > 0:
				pipelines = pipelines[:len(pipelines)-1]
				curJob = nil
				forceUpdateCh <- true
				app.ForceDraw()
			default:
				app.Stop()
				return nil
			}
		}
		if !modalVisible && !logsVisible && len(jobs) > 0 {
			curJob = navi.Navigate(jobs, event)
			root.SendToFront("jobs-" + curJob.Name)
			if inputCh == nil {
				inputCh <- struct{}{}
			}
		}
		switch event.Key() {
		case tcell.KeyCtrlQ:
			app.Stop()
			return nil
		case tcell.KeyCtrlD:
			if curJob.Kind == Job && (curJob.Status == "pending" || curJob.Status == "running") {
				modalVisible = true
				modal := tview.NewModal().
					SetBackgroundColor(tcell.ColorDefault).
					SetText(fmt.Sprintf("Are you sure you want to cancel %s?", curJob.Name)).
					AddButtons([]string{"✘ No", "✔ Yes"}).
					SetDoneFunc(func(buttonIndex int, buttonLabel string) {
						modalVisible = false
						root.RemovePage("yesno")
						if buttonLabel == "✘ No" {
							app.ForceDraw()
							return
						}
						root.RemovePage("logs-" + curJob.Name)
						app.ForceDraw()
						job, err := api.CancelPipelineJob(opts.ApiClient, opts.ProjectID, curJob.ID)
						if err != nil {
							app.Stop()
							log.Fatal(err)
						}
						if job != nil {
							curJob = ViewJobFromJob(job)
							app.ForceDraw()
						}
					})
				root.AddAndSwitchToPage("yesno", modal, false)
				inputCh <- struct{}{}
				app.ForceDraw()
				return nil
			}
		case tcell.KeyCtrlP, tcell.KeyCtrlR:
			if modalVisible || curJob.Kind != Job {
				break
			}
			modalVisible = true
			modal := tview.NewModal().
				SetBackgroundColor(tcell.ColorDefault).
				SetText(fmt.Sprintf("Are you sure you want to run %s?", curJob.Name)).
				AddButtons([]string{"✘ No", "✔ Yes"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					modalVisible = false
					root.RemovePage("yesno")
					if buttonLabel != "✔ Yes" {
						app.ForceDraw()
						return
					}
					root.RemovePage("logs-" + curJob.Name)
					app.ForceDraw()

					job, err := api.PlayOrRetryJobs(
						opts.ApiClient,
						opts.ProjectID,
						curJob.ID,
						curJob.Status,
					)
					if err != nil {
						app.Stop()
						log.Fatal(err)
					}
					if job != nil {
						curJob = ViewJobFromJob(job)
						app.ForceDraw()
					}
				})
			root.AddAndSwitchToPage("yesno", modal, false)
			inputCh <- struct{}{}
			app.ForceDraw()
			return nil
		case tcell.KeyEnter:
			if !modalVisible {
				if curJob.Kind == Job {
					logsVisible = !logsVisible
					if !logsVisible {
						root.HidePage("logs-" + curJob.Name)
					}
					inputCh <- struct{}{}
					app.ForceDraw()
				} else {
					pipelines = append(pipelines, *curJob.OriginalBridge.DownstreamPipeline)
					curJob = nil
					forceUpdateCh <- true
					app.ForceDraw()
				}
				return nil
			}
		case tcell.KeyCtrlSpace:
			app.Suspend(func() {
				ctx, cancel := context.WithCancel(context.Background())
				go func() {
					err := ciutils.RunTraceSha(
						ctx,
						opts.ApiClient,
						opts.Output,
						opts.ProjectID,
						opts.CommitSHA,
						curJob.Name,
					)
					if err != nil {
						app.Stop()
						log.Fatal(err)
					}
					if ctx.Err() == nil {
						fmt.Println("\nPress <Enter> to resume the ci GUI view.")
					}
				}()
				reader := bufio.NewReader(os.Stdin)
				for {
					r, _, err := reader.ReadRune()
					if err != io.EOF && err != nil {
						app.Stop()
						log.Fatal(err)
					}
					if r == '\n' {
						cancel()
						break
					}
				}
			})
			if inputCh == nil {
				inputCh <- struct{}{}
			}
			return nil
		}
		if inputCh == nil {
			inputCh <- struct{}{}
		}
		return event
	}
}

var (
	logsVisible, modalVisible bool
	curJob                    *ViewJob
	jobs                      []*ViewJob
	pipelines                 []gitlab.PipelineInfo
	boxes                     map[string]*tview.TextView
)

func curPipeline(opts ViewOpts) gitlab.PipelineInfo {
	if len(pipelines) == 0 {
		return *opts.Commit.LastPipeline
	}
	return pipelines[len(pipelines)-1]
}

// navigator manages the internal state for processing tcell.EventKeys
type navigator struct {
	depth, idx int
}

// Navigate uses the ci stages as boundaries and returns the currently focused
// job index after processing a *tcell.EventKey
func (n *navigator) Navigate(jobs []*ViewJob, event *tcell.EventKey) *ViewJob {
	stage := jobs[n.idx].Stage
	prev, next := adjacentStages(jobs, stage)
	switch event.Key() {
	case tcell.KeyLeft:
		stage = prev
	case tcell.KeyRight:
		stage = next
	}
	switch event.Rune() {
	case 'h':
		stage = prev
	case 'l':
		stage = next
	}
	l, u := stageBounds(jobs, stage)

	switch event.Key() {
	case tcell.KeyDown:
		n.depth++
		if n.depth > u-l {
			n.depth = u - l
		}
	case tcell.KeyUp:
		n.depth--
	}
	switch event.Rune() {
	case 'j':
		n.depth++
		if n.depth > u-l {
			n.depth = u - l
		}
	case 'k':
		n.depth--
	case 'g':
		n.depth = 0
	case 'G':
		n.depth = u - l
	}

	if n.depth < 0 {
		n.depth = 0
	}
	n.idx = l + n.depth
	if n.idx > u {
		n.idx = u
	}
	return jobs[n.idx]
}

func stageBounds(jobs []*ViewJob, s string) (l, u int) {
	if len(jobs) <= 1 {
		return 0, 0
	}
	p := jobs[0].Stage
	for i, v := range jobs {
		if v.Stage != s && u != 0 {
			return
		}
		if v.Stage != p {
			l = i
			p = v.Stage
		}
		if v.Stage == s {
			u = i
		}
	}
	return
}

func adjacentStages(jobs []*ViewJob, s string) (p, n string) {
	if len(jobs) == 0 {
		return "", ""
	}
	p = jobs[0].Stage

	for _, v := range jobs {
		if v.Stage != s && n != "" {
			n = v.Stage
			return
		}
		if v.Stage == s {
			n = "cur"
		}
		if n == "" {
			p = v.Stage
		}
	}
	n = jobs[len(jobs)-1].Stage
	return
}

func jobsView(
	app *tview.Application,
	jobsCh chan []*ViewJob,
	inputCh chan struct{},
	root *tview.Pages,
	opts ViewOpts,
) {
	select {
	case jobs = <-jobsCh:
	case <-inputCh:
	case <-time.NewTicker(time.Second * 1).C:
	}
	if jobs == nil {
		jobs = <-jobsCh
	}
	if curJob == nil && len(jobs) > 0 {
		curJob = jobs[0]
	}
	if modalVisible {
		return
	}
	if logsVisible {
		logsKey := "logs-" + curJob.Name
		if !root.SwitchToPage(logsKey).HasPage(logsKey) {
			tv := tview.NewTextView()
			tv.
				SetDynamicColors(true).
				SetBackgroundColor(tcell.ColorDefault).
				SetBorderPadding(0, 0, 1, 1).
				SetBorder(true)

			go func() {
				err := ciutils.RunTraceSha(
					context.Background(),
					opts.ApiClient,
					vtclean.NewWriter(tview.ANSIWriter(tv), true),
					opts.ProjectID,
					opts.CommitSHA,
					curJob.Name,
				)
				if err != nil {
					app.Stop()
					log.Fatal(err)
				}
			}()
			root.AddAndSwitchToPage("logs-"+curJob.Name, tv, true)
		}
		return
	}
	px, _, maxX, maxY := root.GetInnerRect()
	var (
		stages    = 0
		lastStage = ""
	)
	// get the number of stages
	for _, j := range jobs {
		if j.Stage != lastStage {
			lastStage = j.Stage
			stages++
		}
	}
	lastStage = ""
	var (
		rowIdx   int
		stageIdx int
		maxTitle = 20
	)
	boxKeys := make(map[string]bool)
	for _, j := range jobs {
		boxX := px + (maxX / stages * stageIdx)
		if j.Stage != lastStage {
			stageIdx++
			lastStage = j.Stage
			key := "stage-" + j.Stage
			boxKeys[key] = true

			x, y, w, h := boxX, maxY/6-4, maxTitle+2, 3
			b := box(root, key, x, y, w, h)

			caser := cases.Title(language.English)
			b.SetText(caser.String(j.Stage))
			b.SetTextAlign(tview.AlignCenter)
		}
	}
	lastStage = jobs[0].Stage
	rowIdx = 0
	stageIdx = 0
	for _, j := range jobs {
		if j.Stage != lastStage {
			rowIdx = 0
			lastStage = j.Stage
			stageIdx++
		}
		boxX := px + (maxX / stages * stageIdx)

		key := "jobs-" + j.Name
		boxKeys[key] = true
		x, y, w, h := boxX, maxY/6+(rowIdx*5), maxTitle+2, 4
		b := box(root, key, x, y, w, h)
		b.SetTitle(j.Name)
		// The scope of jobs to show, one or array of: created, pending, running,
		// failed, success, canceled, skipped; showing all jobs if none provided
		var statChar rune
		switch j.Status {
		case "success":
			b.SetBorderColor(tcell.ColorGreen)
			statChar = '✔'
		case "failed":
			if j.AllowFailure {
				b.SetBorderColor(tcell.ColorOrange)
				statChar = '!'
			} else {
				b.SetBorderColor(tcell.ColorRed)
				statChar = '✘'
			}
		case "running":
			b.SetBorderColor(tcell.ColorBlue)
			statChar = '●'
		case "pending":
			b.SetBorderColor(tcell.ColorYellow)
			statChar = '●'
		case "manual":
			b.SetBorderColor(tcell.ColorGrey)
			statChar = '■'
		case "canceled":
			statChar = 'Ø'
		case "skipped":
			statChar = '»'
		}
		// retryChar := '⟳'
		title := fmt.Sprintf("%c %s", statChar, j.Name)
		// trim the suffix if it matches the stage, I've seen
		// the pattern in 2 different places to handle
		// different stages for the same service and it tends
		// to make the title spill over the max
		title = strings.TrimSuffix(title, ":"+j.Stage)
		b.SetTitle(title)
		// tview default aligns center, which is nice, but if
		// the title is too long we want to bias towards seeing
		// the beginning of it
		if tview.TaggedStringWidth(title) > maxTitle {
			b.SetTitleAlign(tview.AlignLeft)
		}
		triggerText := ""
		if j.Kind == Bridge {
			triggerText = "»"
		}
		if j.StartedAt != nil {
			end := time.Now()
			if j.FinishedAt != nil {
				end = *j.FinishedAt
			}
			b.SetText(triggerText + "\n" + utils.FmtDuration(end.Sub(*j.StartedAt)))
			b.SetTextAlign(tview.AlignRight)
		} else {
			b.SetText(triggerText)
		}
		b.SetTextAlign(tview.AlignRight)
		rowIdx++

	}
	for k := range boxes {
		if !boxKeys[k] {
			root.RemovePage(k)
		}
	}
	root.SendToFront("jobs-" + curJob.Name)
}

func box(root *tview.Pages, key string, x, y, w, h int) *tview.TextView {
	b, ok := boxes[key]
	if !ok {
		b = tview.NewTextView()
		b.
			SetBackgroundColor(tcell.ColorDefault).
			SetBorder(true)
		boxes[key] = b
	}
	b.SetRect(x, y, w, h)

	root.AddPage(key, b, false, true)
	return b
}

func recoverPanic(app *tview.Application) {
	if r := recover(); r != nil {
		app.Stop()
		log.Fatalf("%s\n%s\n", r, string(debug.Stack()))
	}
}

func updateJobs(
	app *tview.Application,
	jobsCh chan []*ViewJob,
	forceUpdateCh chan bool,
	opts ViewOpts,
) {
	defer recoverPanic(app)
	for {
		if modalVisible {
			time.Sleep(time.Second * 1)
			continue
		}
		var jobs []*gitlab.Job
		var bridges []*gitlab.Bridge
		var err error
		pipeline := curPipeline(opts)
		jobs, bridges, err = api.PipelineJobsWithID(
			opts.ApiClient,
			pipeline.ProjectID,
			pipeline.ID,
		)
		if (len(jobs) == 0 && len(bridges) == 0) || err != nil {
			app.Stop()
			log.Fatal(errors.Wrap(err, "failed to find CI jobs."))
		}
		viewJobs := make([]*ViewJob, 0, len(jobs)+len(bridges))
		for _, j := range jobs {
			viewJobs = append(viewJobs, ViewJobFromJob(j))
		}
		for _, b := range bridges {
			viewJobs = append(viewJobs, ViewJobFromBridge(b))
		}
		jobsCh <- latestJobs(viewJobs)
		select {
		case <-forceUpdateCh:
		case <-time.After(time.Second * 5):
		}

	}
}

func linkJobsView(app *tview.Application) func(screen tcell.Screen) {
	return func(screen tcell.Screen) {
		defer recoverPanic(app)
		err := linkJobs(screen, jobs, boxes)
		if err != nil {
			app.Stop()
			log.Fatal(err)
		}
	}
}

func linkJobs(screen tcell.Screen, jobs []*ViewJob, boxes map[string]*tview.TextView) error {
	if logsVisible || modalVisible {
		return nil
	}
	for i, j := range jobs {
		if _, ok := boxes["jobs-"+j.Name]; !ok {
			return errors.Errorf("jobs-%s not found at index: %d", jobs[i].Name, i)
		}
	}
	var padding int
	// find the amount of space between two jobs is adjacent stages
	for i, k := 0, 1; k < len(jobs); i, k = i+1, k+1 {
		if jobs[i].Stage == jobs[k].Stage {
			continue
		}
		x1, _, w, _ := boxes["jobs-"+jobs[i].Name].GetRect()
		x2, _, _, _ := boxes["jobs-"+jobs[k].Name].GetRect()
		stageWidth := x2 - x1 - w
		switch {
		case stageWidth <= 3:
			padding = 1
		case stageWidth <= 6:
			padding = 2
		case stageWidth > 6:
			padding = 3
		}
	}
	for i, k := 0, 1; k < len(jobs); i, k = i+1, k+1 {
		v1 := boxes["jobs-"+jobs[i].Name]
		v2 := boxes["jobs-"+jobs[k].Name]
		link(screen, v1.Box, v2.Box, padding,
			jobs[i].Stage == jobs[0].Stage,           // is first stage?
			jobs[i].Stage == jobs[len(jobs)-1].Stage) // is last stage?
	}
	return nil
}

func link(
	screen tcell.Screen,
	v1 *tview.Box,
	v2 *tview.Box,
	padding int,
	firstStage, lastStage bool,
) {
	x1, y1, w, h := v1.GetRect()
	x2, y2, _, _ := v2.GetRect()

	dx, dy := x2-x1, y2-y1

	p := padding

	// drawing stages
	if dx != 0 {
		hline(screen, x1+w, y2+h/2, dx-w)
		if dy != 0 {
			// dy != 0 means the last stage had multple jobs
			screen.SetContent(x1+w+p-1, y2+h/2, '╦', nil, tcell.StyleDefault)
		}
		return
	}

	// Drawing a job in the same stage
	// left of view
	if !firstStage {
		if r, _, _, _ := screen.GetContent(x2-p, y1+h/2); r == '╚' {
			screen.SetContent(x2-p, y1+h/2, '╠', nil, tcell.StyleDefault)
		} else {
			screen.SetContent(x2-p, y1+h/2, '╦', nil, tcell.StyleDefault)
		}

		for i := 1; i < p; i++ {
			screen.SetContent(x2-i, y2+h/2, '═', nil, tcell.StyleDefault)
		}
		screen.SetContent(x2-p, y2+h/2, '╚', nil, tcell.StyleDefault)

		vline(screen, x2-p, y1+h-1, dy-1)
	}
	// right of view
	if !lastStage {
		if r, _, _, _ := screen.GetContent(x2+w+p-1, y1+h/2); r == '┛' {
			screen.SetContent(x2+w+p-1, y1+h/2, '╣', nil, tcell.StyleDefault)
		}
		for i := 0; i < p-1; i++ {
			screen.SetContent(x2+w+i, y2+h/2, '═', nil, tcell.StyleDefault)
		}
		screen.SetContent(x2+w+p-1, y2+h/2, '╝', nil, tcell.StyleDefault)

		vline(screen, x2+w+p-1, y1+h-1, dy-1)
	}
}

func hline(screen tcell.Screen, x, y, l int) {
	for i := 0; i < l; i++ {
		screen.SetContent(x+i, y, '═', nil, tcell.StyleDefault)
	}
}

func vline(screen tcell.Screen, x, y, l int) {
	for i := 0; i < l; i++ {
		screen.SetContent(x, y+i, '║', nil, tcell.StyleDefault)
	}
}

// latestJobs returns a list of unique jobs favoring the last stage+name
// version of a job in the provided list
func latestJobs(jobs []*ViewJob) []*ViewJob {
	var (
		lastJob = make(map[string]*ViewJob, len(jobs))
		dupIdx  = -1
	)
	for i, j := range jobs {
		_, ok := lastJob[j.Stage+j.Name]
		if dupIdx == -1 && ok {
			dupIdx = i
		}
		// always want the latest job
		lastJob[j.Stage+j.Name] = j
	}
	if dupIdx == -1 {
		dupIdx = len(jobs)
	}
	// first duplicate marks where retries begin
	outJobs := make([]*ViewJob, dupIdx)
	for i := range outJobs {
		j := jobs[i]
		outJobs[i] = lastJob[j.Stage+j.Name]
	}

	return outJobs
}
