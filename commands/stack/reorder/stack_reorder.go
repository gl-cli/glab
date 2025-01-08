package reorder

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/mr/mrutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/git"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/pkg/text"
)

type Options struct {
	LabClient   *gitlab.Client
	CurrentUser *gitlab.User
	BaseRepo    func() (glrepo.Interface, error)
	Remotes     func() (glrepo.Remotes, error)
	Config      func() (config.Config, error)
}

func NewCmdReorderStack(f *cmdutils.Factory, getText cmdutils.GetTextUsingEditor) *cobra.Command {
	opts := &Options{
		Remotes:  f.Remotes,
		Config:   f.Config,
		BaseRepo: f.BaseRepo,
	}

	stackSaveCmd := &cobra.Command{
		Use:   "reorder",
		Short: `Reorder a stack of merge requests. (EXPERIMENTAL.)`,
		Long: `Reorder how the current stack's merge requests are merged.
` + text.ExperimentalString,
		Example: heredoc.Doc(`glab stack reorder`),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := reorderFunc(f, getText, f.IO, opts)
			if err != nil {
				return fmt.Errorf("could not run stack reorder: %v", err)
			}

			return nil
		},
	}
	return stackSaveCmd
}

func reorderFunc(f *cmdutils.Factory, getText cmdutils.GetTextUsingEditor, iostream *iostreams.IOStreams, opts *Options) error {
	iostream.StartSpinner("Reordering\n")
	defer iostream.StopSpinner("")

	title, err := git.GetCurrentStackTitle()
	if err != nil {
		return fmt.Errorf("error retrieving current stack title: %v", err)
	}

	ref, err := git.CurrentStackRefFromCurrentBranch(title)
	if err != nil {
		return fmt.Errorf("error checking for stack: %v", err)
	}

	stack, err := git.GatherStackRefs(title)
	if err != nil {
		return fmt.Errorf("error getting refs from file system: %v", err)
	}

	iostream.StopSpinner("")
	// pausing the spinner in case it's a terminal based editor

	branches, err := promptForOrder(f, getText, stack, ref.Branch)
	if err != nil {
		return fmt.Errorf("error getting new branch order: %v", err)
	}

	// resuming spinner
	iostream.StartSpinner("Reordering\n")

	updatedStack, err := matchBranchesToStack(stack, branches)
	if err != nil {
		return fmt.Errorf("error matching branches to stack: %v", err)
	}

	if reflect.DeepEqual(stack, updatedStack) {
		return fmt.Errorf("no updates needed")
	}

	_, err = authWithGitlab(f, opts)
	if err != nil {
		return fmt.Errorf("error authorizing with GitLab: %v", err)
	}

	err = updateMRs(f, updatedStack, stack)
	if err != nil {
		return fmt.Errorf("error updating merge requests: %v", err)
	}

	iostream.StopSpinner("%s Reordering complete\n", f.IO.Color().GreenCheck())

	return nil
}

func matchBranchesToStack(stack git.Stack, branches []string) (git.Stack, error) {
	stackBranches := stack.Branches()

	// need to clone the refs here so we don't modify the original stack
	newStack := git.Stack{Title: stack.Title, Refs: make(map[string]git.StackRef)}

	for index, branch := range branches {
		// let's find a ref from the branch
		ref, err := stack.RefFromBranch(branch)
		if err != nil {
			return git.Stack{}, fmt.Errorf("could not match branch to stack ref: %v", err)
		}

		var next string
		var prev string

		if index == 0 {
			// first branch in the stack
			prev = ""
		} else {
			// otherwise, get the previous ref
			prevRef, err := stack.RefFromBranch(branches[index-1])
			if err != nil {
				return git.Stack{}, err
			}

			prev = prevRef.SHA
		}

		if index == len(branches)-1 {
			// last branch in the stack
			next = ""
		} else {
			// otherwise, get the next ref
			nextRef, err := stack.RefFromBranch(branches[index+1])
			if err != nil {
				return git.Stack{}, err
			}

			next = nextRef.SHA
		}

		newRef := ref
		newRef.Next = next
		newRef.Prev = prev

		newStack.Refs[newRef.SHA] = newRef

		// update the stack file
		err = git.UpdateStackRefFile(newStack.Title, newRef)
		if err != nil {
			return git.Stack{}, err
		}

		// and remove the branch from our list
		stackBranches = slices.DeleteFunc(stackBranches,
			func(branch string) bool {
				return branch == ref.Branch
			})
	}

	if len(stackBranches) > 0 {
		return git.Stack{},
			errors.New("missing one or more refs from the reordered list: " +
				strings.Join(stackBranches, ", "))
	}

	return newStack, nil
}

func updateMRs(f *cmdutils.Factory, newStack git.Stack, oldStack git.Stack) error {
	for _, ref := range newStack.Iter2() {
		// if there is already an MR and the order has been adjusted
		if ref.MR != "" &&
			(ref.Next != oldStack.Refs[ref.SHA].Next ||
				ref.Prev != oldStack.Refs[ref.SHA].Prev) {

			client, err := f.HttpClient()
			if err != nil {
				return fmt.Errorf("error connecting to GitLab: %v", err)
			}

			mr, _, err := mrutils.MRFromArgsWithOpts(f, []string{ref.Branch}, nil, "opened")
			if err != nil {
				return fmt.Errorf("error getting merge request from GitLab: %v", err)
			}

			var previousBranch string

			if ref.Prev == "" {
				previousBranch, err = git.GetDefaultBranch(git.DefaultRemote)
				if err != nil {
					return fmt.Errorf("error getting default branch: %v", err)
				}
			} else {
				previousBranch = newStack.Refs[ref.Prev].Branch
			}

			opts := gitlab.UpdateMergeRequestOptions{TargetBranch: &previousBranch}

			_, err = api.UpdateMR(client, mr.ProjectID, mr.IID, &opts)
			if err != nil {
				return fmt.Errorf("error updating merge request on GitLab: %v", err)
			}
		}
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
		return nil, fmt.Errorf(
			"no GitLab instances have been authenticated with glab. Run `%s` to authenticate",
			f.IO.Color().Bold("glab auth login"),
		)
	}

	opts.LabClient, err = f.HttpClient()
	if err != nil {
		return nil, fmt.Errorf("error using API client: %v", err)
	}

	return opts.LabClient, nil
}
