package set

import (
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/cli/internal/iostreams"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/google/shlex"
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/config"
)

type options struct {
	io     *iostreams.IOStreams
	config func() config.Config

	name      string
	expansion string
	isShell   bool
	rootCmd   *cobra.Command
}

func NewCmdSet(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		config: f.Config,
		io:     f.IO(),
	}

	aliasSetCmd := &cobra.Command{
		Use:   "set <alias name> '<command>' [flags]",
		Short: `Set an alias for a longer command.`,
		Long: heredoc.Doc(`
		Declare a word as an alias for a longer command.

		Your expansion might include arguments and flags. If your expansion
		includes positional placeholders such as '$1' or '$2', any extra
		arguments that follow the invocation of an alias are inserted
		appropriately.

		Specify '--shell' in your alias to run it through 'sh', a shell
		converter. Shell conversion enables you to compose commands with "|"
		or redirect with ">", with these caveats:

		- Any extra arguments following the alias are not passed to the
		  expanded expression arguments by default.
		- You must explicitly accept the arguments using '$1', '$2', and so on.
		- Use '$@' to accept all arguments.

		For Windows users only:

		- On Windows, shell aliases are executed with 'sh' as installed by
		  Git For Windows. If you installed Git in some other way in Windows,
		  shell aliases might not work for you.
		- Always use quotation marks when defining a command, as in the examples.
		`),
		Example: heredoc.Doc(`
		$ glab alias set mrv 'mr view'
		$ glab mrv -w 123
		> glab mr view -w 123

		$ glab alias set createissue 'glab create issue --title "$1"'
		$ glab createissue "My Issue" --description "Something is broken."
		> glab create issue --title "My Issue" --description "Something is broken."

		$ glab alias set --shell igrep 'glab issue list --assignee="$1" | grep $2'
		$ glab igrep user foo
		> glab issue list --assignee="user" | grep "foo"
	`),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.complete(cmd, args)

			return opts.run()
		},
	}
	aliasSetCmd.Flags().BoolVarP(&opts.isShell, "shell", "s", false, "Declare an alias to be passed through a shell interpreter.")
	return aliasSetCmd
}

func (o *options) complete(cmd *cobra.Command, args []string) {
	o.rootCmd = cmd.Root()
	o.name = args[0]
	o.expansion = args[1]
}

func (o *options) run() error {
	c := o.io.Color()
	cfg := o.config()

	aliasCfg, err := cfg.Aliases()
	if err != nil {
		return err
	}

	if o.io.IsaTTY && o.io.IsErrTTY {
		fmt.Fprintf(o.io.StdErr, "- Adding alias for %s: %s.\n", c.Bold(o.name), c.Bold(o.expansion))
	}

	expansion := o.expansion
	isShell := o.isShell
	if isShell && !strings.HasPrefix(expansion, "!") {
		expansion = "!" + expansion
	}
	isShell = strings.HasPrefix(expansion, "!")

	if validCommand(o.rootCmd, o.name) {
		return fmt.Errorf("could not create alias: %q is already a glab command.", o.name)
	}

	if !isShell && !validCommand(o.rootCmd, expansion) {
		return fmt.Errorf("could not create alias: %s does not correspond to a glab command.", expansion)
	}

	successMsg := fmt.Sprintf("%s Added alias.", c.Green("✓"))
	if oldExpansion, ok := aliasCfg.Get(o.name); ok {
		successMsg = fmt.Sprintf("%s Changed alias %s from %s to %s.",
			c.Green("✓"),
			c.Bold(o.name),
			c.Bold(oldExpansion),
			c.Bold(expansion),
		)
	}

	err = aliasCfg.Set(o.name, expansion)
	if err != nil {
		return fmt.Errorf("could not create alias: %s", err)
	}

	fmt.Fprintln(o.io.StdErr, successMsg)
	return nil
}

func validCommand(rootCmd *cobra.Command, expansion string) bool {
	split, err := shlex.Split(expansion)
	if err != nil {
		return false
	}

	cmd, _, err := rootCmd.Traverse(split)
	return err == nil && cmd != rootCmd
}
