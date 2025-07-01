package sync

import (
	"errors"
	"fmt"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/spf13/cobra"

	"gitlab.com/gitlab-org/cli/internal/auth"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/mr/create"
	"gitlab.com/gitlab-org/cli/internal/commands/mr/mrutils"
	"gitlab.com/gitlab-org/cli/internal/dbg"
	"gitlab.com/gitlab-org/cli/internal/git"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/text"
)

type options struct {
	io        *iostreams.IOStreams
	stack     git.Stack
	target    glrepo.Interface
	source    glrepo.Interface
	labClient *gitlab.Client
	baseRepo  func() (glrepo.Interface, error)
	remotes   func() (glrepo.Remotes, error)
	user      gitlab.User
}

// max string size for MR title is ~255, but we'll add a "..."
const maxMRTitleSize = 252

const (
	BranchIsBehind    = "Your branch is behind"
	BranchHasDiverged = "have diverged"
	NothingToCommit   = "nothing to commit"
	mergedStatus      = "merged"
	closedStatus      = "closed"
)

func NewCmdSyncStack(f cmdutils.Factory, gr git.GitRunner) *cobra.Command {
	opts := &options{
		io:       f.IO(),
		remotes:  f.Remotes,
		baseRepo: f.BaseRepo,
	}

	stackSaveCmd := &cobra.Command{
		Use:   "sync",
		Short: `Sync and submit progress on a stacked diff. (EXPERIMENTAL.)`,
		Long: heredoc.Doc(`Sync and submit progress on a stacked diff. This command runs these steps:

1. Optional. If working in a fork, select whether to push to the fork,
   or the upstream repository.
1. Pushes any amended changes to their merge requests.
1. Rebases any changes that happened previously in the stack.
1. Removes any branches that were already merged, or with a closed merge request.
` + text.ExperimentalString),
		Example: heredoc.Doc(`
			$ glab stack sync
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.io.StartSpinner("Syncing")
			defer opts.io.StopSpinner("")

			var gr git.StandardGitCommand

			return opts.run(f, gr)
		},
	}

	return stackSaveCmd
}

func (o *options) run(f cmdutils.Factory, gr git.GitRunner) error {
	client, err := auth.GetAuthenticatedClient(f.Config(), f.HttpClient, f.IO())
	if err != nil {
		return fmt.Errorf("error authorizing with GitLab: %v", err)
	}
	o.labClient = client

	o.io.StopSpinner("")

	repo, err := f.BaseRepo()
	if err != nil {
		return fmt.Errorf("error determining base repo: %v", err)
	}

	// This prompts the user for the head repo if they're in a fork,
	// allowing them to choose between their fork and the original repository
	source, err := create.ResolvedHeadRepo(f)()
	if err != nil {
		return fmt.Errorf("error determining head repo: %v", err)
	}

	o.io.StartSpinner("Syncing")

	stack, err := getStack()
	if err != nil {
		return fmt.Errorf("error getting current stack: %v", err)
	}

	user, _, err := client.Users.CurrentUser()
	if err != nil {
		return fmt.Errorf("error getting current user: %v", err)
	}

	o.stack = stack
	o.target = repo
	o.source = source
	o.user = *user

	err = fetchOrigin(gr)
	if err != nil {
		return err
	}

	pushAfterSync := false

	for ref := range stack.Iter() {
		status, err := branchStatus(&ref, gr)
		if err != nil {
			return fmt.Errorf("error getting branch status: %v", err)
		}

		switch {
		case strings.Contains(status, BranchIsBehind):
			err = branchBehind(o.io, &ref, gr)
			if err != nil {
				return err
			}
		case strings.Contains(status, BranchHasDiverged):
			needsPush, err := branchDiverged(o.io, &ref, &stack, gr)
			if err != nil {
				return err
			}

			if needsPush {
				pushAfterSync = true
			}
		case strings.Contains(status, NothingToCommit):
			// this is fine. we can just move on.
		default:
			return fmt.Errorf("your Git branch is ahead, but it shouldn't be. You might need to squash your commits.")
		}

		if ref.MR == "" {
			err := populateMR(o.io, &ref, o, client, gr)
			if err != nil {
				return err
			}
		} else {
			// we found an MR. let's get the status:
			mr, _, err := mrutils.MRFromArgsWithOpts(f, []string{ref.Branch}, nil, "any")
			if err != nil {
				return fmt.Errorf("error getting merge request from branch: %v. Does it still exist?", err)
			}

			// remove the MR from the stack if it's merged
			// do not remove the MR from the stack if it is closed,
			// but alert the user
			err = removeOldMrs(o.io, &ref, mr, &stack, gr)
			if err != nil {
				return fmt.Errorf("error removing merged merge request: %v", err)
			}
		}
	}

	if pushAfterSync {
		err := forcePushAllWithLease(o.io, &stack, gr)
		if err != nil {
			return fmt.Errorf("error pushing branches to remote: %v", err)
		}
	}

	fmt.Print(progressString(o.io, "Sync finished!"))
	return nil
}

func getStack() (git.Stack, error) {
	title, err := git.GetCurrentStackTitle()
	if err != nil {
		return git.Stack{}, fmt.Errorf("error getting current stack: %v", err)
	}

	stack, err := git.GatherStackRefs(title)
	if err != nil {
		return git.Stack{}, fmt.Errorf("error getting current stack references: %v", err)
	}
	return stack, nil
}

func gitPull(gr git.GitRunner) (string, error) {
	pull, err := gr.Git("pull")
	if err != nil {
		return "", err
	}
	dbg.Debug("Pulled:", pull)

	return pull, nil
}

func fetchOrigin(gr git.GitRunner) error {
	output, err := gr.Git("fetch", git.DefaultRemote)
	dbg.Debug("Fetching from remote:", output)

	if err != nil {
		return err
	}

	return nil
}

func branchStatus(ref *git.StackRef, gr git.GitRunner) (string, error) {
	checkout, err := gr.Git("checkout", ref.Branch)
	if err != nil {
		return "", err
	}
	dbg.Debug("Checked out:", checkout)

	output, err := gr.Git("status", "-uno")
	if err != nil {
		return "", err
	}
	dbg.Debug("Git status:", output)

	return output, nil
}

func rebaseWithUpdateRefs(ref *git.StackRef, stack *git.Stack, gr git.GitRunner) error {
	lastRef := stack.Last()

	checkout, err := gr.Git("checkout", lastRef.Branch)
	if err != nil {
		return err
	}
	dbg.Debug("Checked out:", checkout)

	rebase, err := gr.Git("rebase", "--fork-point", "--update-refs", ref.Branch)
	if err != nil {
		return err
	}
	dbg.Debug("Rebased:", rebase)

	return nil
}

func forcePushAllWithLease(io *iostreams.IOStreams, stack *git.Stack, gr git.GitRunner) error {
	fmt.Print(progressString(
		io,
		"Updating branches:",
		strings.Join(stack.Branches(), ", "),
	))

	output, err := gr.Git(append(
		[]string{"push", git.DefaultRemote, "--force-with-lease"},
		stack.Branches()...,
	)...)
	if err != nil {
		return err
	}

	fmt.Print(progressString(io, "Push succeeded: "+output))
	return nil
}

func createMR(client *gitlab.Client, opts *options, ref *git.StackRef, gr git.GitRunner) (*gitlab.MergeRequest, error) {
	targetProject, err := opts.target.Project(client)
	if err != nil {
		return &gitlab.MergeRequest{}, fmt.Errorf("error getting target project: %v", err)
	}

	_, err = gr.Git("push", "--set-upstream", git.DefaultRemote, ref.Branch)
	if err != nil {
		return &gitlab.MergeRequest{}, fmt.Errorf("error pushing branch: %v", err)
	}

	var previousBranch string
	if ref.IsFirst() {
		// Point to the base branch
		previousBranch, err = opts.stack.BaseBranch(gr)
		if err != nil {
			return &gitlab.MergeRequest{}, fmt.Errorf("error getting base branch: %w", err)
		}

		if !git.RemoteBranchExists(previousBranch, gr) {
			return &gitlab.MergeRequest{}, fmt.Errorf("branch %q does not exist on remote %q. Please push the branch to the remote before syncing",
				previousBranch,
				git.DefaultRemote)
		}

	} else {
		// if we have a previous branch, let's point to that
		previousBranch = opts.stack.Refs[ref.Prev].Branch
	}

	var description string
	if len(ref.Description) > maxMRTitleSize {
		description = ref.Description[0:68] + "..."
	} else {
		description = ref.Description
	}

	l := &gitlab.CreateMergeRequestOptions{
		Title:              gitlab.Ptr(description),
		SourceBranch:       gitlab.Ptr(ref.Branch),
		TargetBranch:       gitlab.Ptr(previousBranch),
		AssigneeID:         gitlab.Ptr(opts.user.ID),
		RemoveSourceBranch: gitlab.Ptr(true),
		TargetProjectID:    gitlab.Ptr(targetProject.ID),
	}

	mr, _, err := client.MergeRequests.CreateMergeRequest(opts.source.FullName(), l)
	if err != nil {
		return &gitlab.MergeRequest{}, fmt.Errorf("error creating merge request with the API: %v", err)
	}

	return mr, nil
}

func removeOldMrs(io *iostreams.IOStreams, ref *git.StackRef, mr *gitlab.MergeRequest, stack *git.Stack, gr git.GitRunner) error {
	if mr.State == mergedStatus {
		progress := fmt.Sprintf("Merge request !%v has merged. Removing reference...", mr.IID)
		fmt.Println(progressString(io, progress))

		err := stack.RemoveRef(*ref, gr)
		if err != nil {
			return err
		}
	} else if mr.State == closedStatus {
		progress := fmt.Sprintf("MR !%v has closed", mr.IID)
		fmt.Println(progressString(io, progress))
	}
	return nil
}

func errorString(io *iostreams.IOStreams, lines ...string) string {
	redCheck := io.Color().Red("✘")

	title := lines[0]
	body := strings.Join(lines[1:], "\n  ")

	return fmt.Sprintf("\n%s %s \n  %s", redCheck, title, body)
}

func progressString(io *iostreams.IOStreams, lines ...string) string {
	blueDot := io.Color().ProgressIcon()
	title := lines[0]

	var body string

	if len(lines) > 1 {
		body = strings.Join(lines[1:], "\n  ")
		return fmt.Sprintf("\n%s %s \n  %s", blueDot, title, body)
	}
	return fmt.Sprintf("\n%s %s\n", blueDot, title)
}

func branchDiverged(io *iostreams.IOStreams, ref *git.StackRef, stack *git.Stack, gr git.GitRunner) (bool, error) {
	fmt.Println(progressString(io, ref.Branch+" has diverged. Rebasing..."))

	err := rebaseWithUpdateRefs(ref, stack, gr)
	if err != nil {
		return false, errors.New(errorString(
			io,
			"could not rebase, likely due to a merge conflict.",
			"Fix the issues with Git and run `glab stack sync` again.",
		))
	}

	return true, nil
}

func branchBehind(io *iostreams.IOStreams, ref *git.StackRef, gr git.GitRunner) error {
	// possibly someone applied suggestions or someone else added a
	// different commit
	fmt.Println(progressString(io, ref.Branch+" is behind - pulling updates."))

	_, err := gitPull(gr)
	if err != nil {
		return fmt.Errorf("error checking for a running Git pull: %v", err)
	}

	return nil
}

func populateMR(io *iostreams.IOStreams, ref *git.StackRef, opts *options, client *gitlab.Client, gr git.GitRunner) error {
	// no MR - lets create one!
	fmt.Println(progressString(io, ref.Branch+" needs a merge request. Creating it now."))

	mr, err := createMR(client, opts, ref, gr)
	if err != nil {
		return fmt.Errorf("error updating stack ref files: %v", err)
	}

	fmt.Println(progressString(io, "Merge request created!"))
	fmt.Println(mrutils.DisplayMR(io.Color(), &mr.BasicMergeRequest, true))

	// update the ref
	ref.MR = mr.WebURL
	err = git.UpdateStackRefFile(opts.stack.Title, *ref)
	if err != nil {
		return fmt.Errorf("error updating stack ref files: %v", err)
	}

	return nil
}
