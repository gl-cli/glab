package sync

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/spf13/cobra"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/mr/create"
	"gitlab.com/gitlab-org/cli/commands/mr/mrutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/git"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/pkg/text"
)

type Options struct {
	stack       git.Stack
	target      glrepo.Interface
	source      glrepo.Interface
	LabClient   *gitlab.Client
	CurrentUser *gitlab.User
	BaseRepo    func() (glrepo.Interface, error)
	Remotes     func() (glrepo.Remotes, error)
	Config      func() (config.Config, error)
	user        gitlab.User
}

var iostream *iostreams.IOStreams

// max string size for MR title is ~255, but we'll add a "..."
const maxMRTitleSize = 252

const (
	BranchIsBehind    = "Your branch is behind"
	BranchHasDiverged = "have diverged"
	NothingToCommit   = "nothing to commit"
	mergedStatus      = "merged"
	closedStatus      = "closed"
)

func NewCmdSyncStack(f *cmdutils.Factory) *cobra.Command {
	opts := &Options{
		Remotes:  f.Remotes,
		Config:   f.Config,
		BaseRepo: f.BaseRepo,
	}

	iostream = f.IO

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
			glab stack sync
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			iostream.StartSpinner("Syncing")

			var gr git.StandardGitCommand

			err := stackSync(f, iostream, opts, gr)
			iostream.StopSpinner("")
			if err != nil {
				return fmt.Errorf("could not run sync: %v", err)
			}

			return nil
		},
	}

	return stackSaveCmd
}

func stackSync(f *cmdutils.Factory, iostream *iostreams.IOStreams, opts *Options, gr git.GitRunner) error {
	client, err := authWithGitlab(f, opts)
	if err != nil {
		return fmt.Errorf("error authorizing with GitLab: %v", err)
	}

	iostream.StopSpinner("")

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

	iostream.StartSpinner("Syncing")

	stack, err := getStack()
	if err != nil {
		return fmt.Errorf("error getting current stack: %v", err)
	}

	user, _, err := client.Users.CurrentUser()
	if err != nil {
		return fmt.Errorf("error getting current user: %v", err)
	}

	opts.stack = stack
	opts.target = repo
	opts.source = source
	opts.user = *user

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
			err = branchBehind(&ref, gr)
			if err != nil {
				return err
			}
		case strings.Contains(status, BranchHasDiverged):
			needsPush, err := branchDiverged(&ref, &stack, gr)
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
			err := populateMR(&ref, opts, client, gr)
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
			err = removeOldMrs(&ref, mr, &stack)
			if err != nil {
				return fmt.Errorf("error removing merged merge request: %v", err)
			}
		}
	}

	if pushAfterSync {
		err := forcePushAllWithLease(&stack, gr)
		if err != nil {
			return fmt.Errorf("error pushing branches to remote: %v", err)
		}
	}

	fmt.Print(progressString("Sync finished!"))
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
	debug("Pulled:", pull)

	return pull, nil
}

func fetchOrigin(gr git.GitRunner) error {
	output, err := gr.Git("fetch", git.DefaultRemote)
	debug("Fetching from remote:", output)

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
	debug("Checked out:", checkout)

	output, err := gr.Git("status", "-uno")
	if err != nil {
		return "", err
	}
	debug("Git status:", output)

	return output, nil
}

func rebaseWithUpdateRefs(ref *git.StackRef, stack *git.Stack, gr git.GitRunner) error {
	lastRef := stack.Last()

	checkout, err := gr.Git("checkout", lastRef.Branch)
	if err != nil {
		return err
	}
	debug("Checked out:", checkout)

	rebase, err := gr.Git("rebase", "--fork-point", "--update-refs", ref.Branch)
	if err != nil {
		return err
	}
	debug("Rebased:", rebase)

	return nil
}

func forcePushAllWithLease(stack *git.Stack, gr git.GitRunner) error {
	fmt.Print(progressString(
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

	fmt.Print(progressString("Push succeeded: " + output))
	return nil
}

func createMR(client *gitlab.Client, opts *Options, ref *git.StackRef, gr git.GitRunner) (*gitlab.MergeRequest, error) {
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
		// Point to the default one
		previousBranch, err = getDefaultBranch(git.DefaultRemote, gr)
		if err != nil {
			return &gitlab.MergeRequest{}, fmt.Errorf("error getting default branch: %v", err)
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

	mr, err := api.CreateMR(client, opts.source.FullName(), l)
	if err != nil {
		return &gitlab.MergeRequest{}, fmt.Errorf("error creating merge request with the API: %v", err)
	}

	return mr, nil
}

func removeOldMrs(ref *git.StackRef, mr *gitlab.MergeRequest, stack *git.Stack) error {
	if mr.State == mergedStatus {
		progress := fmt.Sprintf("Merge request !%v has merged. Removing reference...", mr.IID)
		fmt.Println(progressString(progress))

		err := stack.RemoveRef(*ref)
		if err != nil {
			return err
		}
	} else if mr.State == closedStatus {
		progress := fmt.Sprintf("MR !%v has closed", mr.IID)
		fmt.Println(progressString(progress))
	}
	return nil
}

func authWithGitlab(f *cmdutils.Factory, opts *Options) (*gitlab.Client, error) {
	glConfig, err := f.Config()
	if err != nil {
		return nil, err
	}

	instances, err := glConfig.Hosts()
	if len(instances) == 0 || err != nil {
		return nil, fmt.Errorf("no GitLab instances have been authenticated with glab. Run `%s` to authenticate.", f.IO.Color().Bold("glab auth login"))
	}

	opts.LabClient, err = f.HttpClient()
	if err != nil {
		return nil, fmt.Errorf("error using API client: %v", err)
	}

	return opts.LabClient, nil
}

func errorString(lines ...string) string {
	redCheck := iostream.Color().Red("✘")

	title := lines[0]
	body := strings.Join(lines[1:], "\n  ")

	return fmt.Sprintf("\n%s %s \n  %s", redCheck, title, body)
}

func progressString(lines ...string) string {
	blueDot := iostream.Color().ProgressIcon()
	title := lines[0]

	var body string

	if len(lines) > 1 {
		body = strings.Join(lines[1:], "\n  ")
		return fmt.Sprintf("\n%s %s \n  %s", blueDot, title, body)
	}
	return fmt.Sprintf("\n%s %s\n", blueDot, title)
}

func debug(output ...string) {
	if os.Getenv("DEBUG") != "" {
		log.Print(output)
	}
}

func branchDiverged(ref *git.StackRef, stack *git.Stack, gr git.GitRunner) (bool, error) {
	fmt.Println(progressString(ref.Branch + " has diverged. Rebasing..."))

	err := rebaseWithUpdateRefs(ref, stack, gr)
	if err != nil {
		return false, errors.New(errorString(
			"could not rebase, likely due to a merge conflict.",
			"Fix the issues with Git and run `glab stack sync` again.",
		))
	}

	return true, nil
}

func branchBehind(ref *git.StackRef, gr git.GitRunner) error {
	// possibly someone applied suggestions or someone else added a
	// different commit
	fmt.Println(progressString(ref.Branch + " is behind - pulling updates."))

	_, err := gitPull(gr)
	if err != nil {
		return fmt.Errorf("error checking for a running Git pull: %v", err)
	}

	return nil
}

func populateMR(ref *git.StackRef, opts *Options, client *gitlab.Client, gr git.GitRunner) error {
	// no MR - lets create one!
	fmt.Println(progressString(ref.Branch + " needs a merge request. Creating it now."))

	mr, err := createMR(client, opts, ref, gr)
	if err != nil {
		return fmt.Errorf("error updating stack ref files: %v", err)
	}

	fmt.Println(progressString("Merge request created!"))
	fmt.Println(mrutils.DisplayMR(iostream.Color(), mr, true))

	// update the ref
	ref.MR = mr.WebURL
	err = git.UpdateStackRefFile(opts.stack.Title, *ref)
	if err != nil {
		return fmt.Errorf("error updating stack ref files: %v", err)
	}

	return nil
}

func getDefaultBranch(remote string, gr git.GitRunner) (string, error) {
	defBranchOutput, err := gr.Git("remote", "show", remote)
	if err != nil {
		return "", err
	}

	branch, err := git.ParseDefaultBranch([]byte(defBranchOutput))
	if err != nil {
		return "", err
	}

	return branch, nil
}
