package completion

import (
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/spf13/cobra"
)

func NewCmdCompletion(io *iostreams.IOStreams) *cobra.Command {
	var (
		shellType string

		// description will not be added if true
		excludeDesc = false
	)

	completionCmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate shell completion scripts",
		Long: heredoc.Docf(`
		The output of this command will be computer code and is meant to be saved 
		to a file or immediately evaluated by an interactive shell.
		
		For example, for bash you could add this to your %[1]s~/.bash_profile%[1]s:
		
		%[2]splaintext
		eval "$(glab completion -s bash)"
		%[2]s

		Generate a %[1]s_glab%[1]s completion script and put it somewhere in your %[1]s$fpath%[1]s:

		%[2]splaintext
		glab completion -s zsh > /usr/local/share/zsh/site-functions/_glab
		%[2]s

		Ensure that the following is present in your %[1]s~/.zshrc%[1]s:

		- %[1]sautoload -U compinit%[1]s
		- %[1]scompinit -i%[1]s

		Zsh version 5.7 or later is recommended.

		When installing glab through a package manager, however, it's possible that
		no additional shell configuration is necessary to gain completion support. 
		For Homebrew, see <https://docs.brew.sh/Shell-Completion>
		`, "`", "```"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := io.StdOut
			rootCmd := cmd.Parent()

			switch shellType {
			case "bash":
				return rootCmd.GenBashCompletionV2(out, !excludeDesc)
			case "zsh":
				if excludeDesc {
					return rootCmd.GenZshCompletionNoDesc(out)
				}
				return rootCmd.GenZshCompletion(out)
			case "powershell":
				if excludeDesc {
					return rootCmd.GenPowerShellCompletion(out)
				}
				return rootCmd.GenPowerShellCompletionWithDesc(out)
			case "fish":
				return rootCmd.GenFishCompletion(out, !excludeDesc)
			default:
				return fmt.Errorf("unsupported shell type %q", shellType)
			}
		},
	}

	completionCmd.Flags().StringVarP(&shellType, "shell", "s", "bash", "Shell type: {bash|zsh|fish|powershell}")
	completionCmd.Flags().BoolVarP(&excludeDesc, "no-desc", "", false, "Do not include shell completion description")
	return completionCmd
}
