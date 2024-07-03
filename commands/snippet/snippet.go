package snippet

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/snippet/create"
)

func NewCmdSnippet(f *cmdutils.Factory) *cobra.Command {
	snippetCmd := &cobra.Command{
		Use:   "snippet <command> [flags]",
		Short: `Create, view and manage snippets.`,
		Long:  ``,
		Example: heredoc.Doc(`
			glab snippet create --title "Title of the snippet" --filename "main.go"
		`),
		Annotations: map[string]string{
			"help:arguments": heredoc.Doc(`
			A snippet can be supplied as argument in the following format:
			- by number, e.g. "123"
			`),
		},
	}

	cmdutils.EnableRepoOverride(snippetCmd, f)

	snippetCmd.AddCommand(create.NewCmdCreate(f))
	return snippetCmd
}
