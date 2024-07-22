package navigate

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"

	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/git"
	"gitlab.com/gitlab-org/cli/pkg/text"
)

func baseCommand() (git.Stack, error) {
	title, err := git.GetCurrentStackTitle()
	if err != nil {
		return git.Stack{}, err
	}

	stack, err := git.GatherStackRefs(title)
	if err != nil {
		return git.Stack{}, err
	}

	return stack, nil
}

func NewCmdStackFirst(f *cmdutils.Factory) *cobra.Command {
	return &cobra.Command{
		Use:     "first",
		Short:   "Moves to the first diff in the stack. (EXPERIMENTAL.)",
		Long:    "Moves to the first diff in the stack, and checks out that branch.\n" + text.ExperimentalString,
		Example: "glab stack first",
		RunE: func(cmd *cobra.Command, args []string) error {
			stack, err := baseCommand()
			if err != nil {
				return err
			}

			ref, err := stack.First()
			if err != nil {
				return err
			}

			err = git.CheckoutBranch(ref.Branch)
			if err != nil {
				return err
			}

			switchMessage(f, &ref)

			return nil
		},
	}
}

func NewCmdStackNext(f *cmdutils.Factory) *cobra.Command {
	return &cobra.Command{
		Use:     "next",
		Short:   "Moves to the next diff in the stack. (EXPERIMENTAL.)",
		Long:    "Moves to the next diff in the stack, and checks out that branch.\n" + text.ExperimentalString,
		Example: "glab stack next",
		RunE: func(cmd *cobra.Command, args []string) error {
			stack, err := baseCommand()
			if err != nil {
				return err
			}

			ref, err := git.CurrentStackRefFromBranch(stack.Title)
			if err != nil {
				return err
			}

			if ref.Next == "" {
				return fmt.Errorf("you are already at the last diff. Use `glab stack list` to see the complete list.")
			}

			err = git.CheckoutBranch(stack.Refs[ref.Next].Branch)
			if err != nil {
				return err
			}

			next := stack.Refs[ref.Next]
			switchMessage(f, &next)

			return nil
		},
	}
}

func NewCmdStackPrev(f *cmdutils.Factory) *cobra.Command {
	return &cobra.Command{
		Use:     "prev",
		Short:   "Moves to the previous diff in the stack. (EXPERIMENTAL.)",
		Long:    "Moves to the previous diff in the stack, and checks out that branch.\n" + text.ExperimentalString,
		Example: "glab stack prev",
		RunE: func(cmd *cobra.Command, args []string) error {
			stack, err := baseCommand()
			if err != nil {
				return err
			}

			ref, err := git.CurrentStackRefFromBranch(stack.Title)
			if err != nil {
				return err
			}

			if ref.Prev == "" {
				return fmt.Errorf("you are already at the first diff. Use `glab stack list` to see the complete list.")
			}

			err = git.CheckoutBranch(stack.Refs[ref.Prev].Branch)
			if err != nil {
				return err
			}

			prev := stack.Refs[ref.Prev]
			switchMessage(f, &prev)

			return nil
		},
	}
}

func NewCmdStackLast(f *cmdutils.Factory) *cobra.Command {
	return &cobra.Command{
		Use:     "last",
		Short:   "Moves to the last diff in the stack. (EXPERIMENTAL.)",
		Long:    "Moves to the last diff in the stack, and checks out that branch.\n" + text.ExperimentalString,
		Example: "glab stack last",
		RunE: func(cmd *cobra.Command, args []string) error {
			stack, err := baseCommand()
			if err != nil {
				return err
			}

			ref, err := stack.Last()
			if err != nil {
				return err
			}

			err = git.CheckoutBranch(ref.Branch)
			if err != nil {
				return err
			}

			switchMessage(f, &ref)

			return nil
		},
	}
}

func NewCmdStackMove(f *cmdutils.Factory) *cobra.Command {
	return &cobra.Command{
		Use:     "move",
		Short:   "Moves to any selected entry in the stack. (EXPERIMENTAL.)",
		Long:    "Shows a menu with a fuzzy finder to select a stack.\n" + text.ExperimentalString,
		Example: "glab stack move",
		RunE: func(cmd *cobra.Command, args []string) error {
			stack, err := baseCommand()
			if err != nil {
				return err
			}

			var branches []string
			var descriptions []string

			firstRef, err := stack.First()
			if err != nil {
				return err
			}

			i := 1
			ref := firstRef
			for {
				branches = append(branches, ref.Branch)
				message := fmt.Sprintf("%v: %v", i, ref.Description)
				descriptions = append(descriptions, message)

				i++

				if ref.Next == "" {
					break
				}

				ref = stack.Refs[ref.Next]
			}

			var branch string
			prompt := &survey.Select{
				Message: "Choose a diff to be checked out:",
				Options: branches,
				Description: func(value string, index int) string {
					return descriptions[index]
				},
			}
			err = survey.AskOne(prompt, &branch)
			if err != nil {
				return err
			}

			err = git.CheckoutBranch(branch)
			if err != nil {
				return err
			}

			return nil
		},
	}
}

func switchMessage(f *cmdutils.Factory, ref *git.StackRef) {
	color := f.IO.Color()
	fmt.Printf(
		"%v Switched to branch: %v - %v\n",
		color.ProgressIcon(),
		color.Blue(ref.Branch),
		color.Bold(ref.Description),
	)
}
