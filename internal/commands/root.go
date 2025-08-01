package commands

import (
	"errors"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	aliasCmd "gitlab.com/gitlab-org/cli/internal/commands/alias"
	apiCmd "gitlab.com/gitlab-org/cli/internal/commands/api"
	authCmd "gitlab.com/gitlab-org/cli/internal/commands/auth"
	changelogCmd "gitlab.com/gitlab-org/cli/internal/commands/changelog"
	pipelineCmd "gitlab.com/gitlab-org/cli/internal/commands/ci"
	clusterCmd "gitlab.com/gitlab-org/cli/internal/commands/cluster"
	completionCmd "gitlab.com/gitlab-org/cli/internal/commands/completion"
	configCmd "gitlab.com/gitlab-org/cli/internal/commands/config"
	deployKeyCmd "gitlab.com/gitlab-org/cli/internal/commands/deploy-key"
	duoCmd "gitlab.com/gitlab-org/cli/internal/commands/duo"
	"gitlab.com/gitlab-org/cli/internal/commands/help"
	incidentCmd "gitlab.com/gitlab-org/cli/internal/commands/incident"
	issueCmd "gitlab.com/gitlab-org/cli/internal/commands/issue"
	iterationCmd "gitlab.com/gitlab-org/cli/internal/commands/iteration"
	jobCmd "gitlab.com/gitlab-org/cli/internal/commands/job"
	labelCmd "gitlab.com/gitlab-org/cli/internal/commands/label"
	mrCmd "gitlab.com/gitlab-org/cli/internal/commands/mr"
	projectCmd "gitlab.com/gitlab-org/cli/internal/commands/project"
	releaseCmd "gitlab.com/gitlab-org/cli/internal/commands/release"
	scheduleCmd "gitlab.com/gitlab-org/cli/internal/commands/schedule"
	securefileCmd "gitlab.com/gitlab-org/cli/internal/commands/securefile"
	snippetCmd "gitlab.com/gitlab-org/cli/internal/commands/snippet"
	sshCmd "gitlab.com/gitlab-org/cli/internal/commands/ssh-key"
	stackCmd "gitlab.com/gitlab-org/cli/internal/commands/stack"
	tokenCmd "gitlab.com/gitlab-org/cli/internal/commands/token"
	updateCmd "gitlab.com/gitlab-org/cli/internal/commands/update"
	userCmd "gitlab.com/gitlab-org/cli/internal/commands/user"
	variableCmd "gitlab.com/gitlab-org/cli/internal/commands/variable"
	versionCmd "gitlab.com/gitlab-org/cli/internal/commands/version"
)

// NewCmdRoot is the main root/parent command
func NewCmdRoot(f cmdutils.Factory) *cobra.Command {
	c := f.IO().Color()
	rootCmd := &cobra.Command{
		Use:           "glab <command> <subcommand> [flags]",
		Short:         "A GitLab CLI tool.",
		Long:          `GLab is an open source GitLab CLI tool that brings GitLab to your command line.`,
		SilenceErrors: true,
		SilenceUsage:  true,
		Annotations: map[string]string{
			"help:environment": heredoc.Doc(`
			BROWSER: The web browser to use for opening links.
			Can be set in the config with 'glab config set browser mybrowser'.

			DEBUG: Set to 1 or true to output more logging information, including underlying Git commands,
			expanded aliases and DNS error details.

			FORCE_HYPERLINKS: Set to 1 to force hyperlinks in output, even when not outputting to a TTY.

			GITLAB_CLIENT_ID: Provide custom 'client_id' generated by GitLab OAuth 2.0 application.
			Defaults to the 'client-id' for GitLab.com.

			GITLAB_HOST or GL_HOST: If GitLab Self-Managed or GitLab Dedicated, specify the URL of the GitLab server.
			(Example: https://gitlab.example.com) Defaults to https://gitlab.com.

			GITLAB_TOKEN: An authentication token for API requests. Set this variable to
			avoid prompts to authenticate. Overrides any previously-stored credentials.
			Can be set in the config with 'glab config set token xxxxxx'.

			GLAB_CHECK_UPDATE: Set to 1 or true to force an update check. By default the cli tool
			checks for updates once a day.

			GLAB_SEND_TELEMETRY: Set to 0 or false to disable telemetry being sent to your GitLab instance.
			Can be set in the config with 'glab config set telemetry false'.
			See https://docs.gitlab.com/administration/settings/usage_statistics/ for more information

			GLAB_CONFIG_DIR: Set to a directory path to override the global configuration location.

			GLAMOUR_STYLE: The environment variable to set your desired Markdown renderer style.
			Available options: dark, light, notty. To set a custom style, read
			https://github.com/charmbracelet/glamour#styles

			NO_COLOR: Set to any value to avoid printing ANSI escape sequences for color output.

			NO_PROMPT: Set to 1 (true) or 0 (false) to disable or enable prompts.

			REMOTE_ALIAS or GIT_REMOTE_URL_VAR: A 'git remote' variable or alias that contains
			the GitLab URL. Can be set in the config with 'glab config set remote_alias origin'.

			VISUAL, EDITOR (in order of precedence): The editor tool to use for authoring text.
			Can be set in the config with 'glab config set editor vim'.
		`),
			"help:feedback": heredoc.Docf(`
			Encountered a bug or want to suggest a feature?
			Open an issue using '%s'
		`, c.Bold(c.Yellow("glab issue create -R gitlab-org/cli"))),
		},
	}

	rootCmd.SetOut(f.IO().StdOut)
	rootCmd.SetErr(f.IO().StdErr)

	rootCmd.PersistentFlags().BoolP("help", "h", false, "Show help for this command.")
	rootCmd.SetHelpFunc(func(command *cobra.Command, args []string) {
		help.RootHelpFunc(f.IO().Color(), command, args)
	})
	rootCmd.SetUsageFunc(help.RootUsageFunc)
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		if errors.Is(err, pflag.ErrHelp) {
			return err
		}
		return &cmdutils.FlagError{Err: err}
	})

	buildInfo := f.BuildInfo()
	formattedVersion := versionCmd.Scheme(buildInfo.Version, buildInfo.Commit)
	rootCmd.SetVersionTemplate(formattedVersion)
	rootCmd.Version = formattedVersion

	// Child commands
	rootCmd.AddCommand(aliasCmd.NewCmdAlias(f))
	rootCmd.AddCommand(configCmd.NewCmdConfig(f))
	rootCmd.AddCommand(completionCmd.NewCmdCompletion(f.IO()))
	rootCmd.AddCommand(versionCmd.NewCmdVersion(f))
	rootCmd.AddCommand(updateCmd.NewCheckUpdateCmd(f))
	rootCmd.AddCommand(authCmd.NewCmdAuth(f))

	rootCmd.AddCommand(changelogCmd.NewCmdChangelog(f))
	rootCmd.AddCommand(clusterCmd.NewCmdCluster(f))
	rootCmd.AddCommand(issueCmd.NewCmdIssue(f))
	rootCmd.AddCommand(iterationCmd.NewCmdIteration(f))
	rootCmd.AddCommand(incidentCmd.NewCmdIncident(f))
	rootCmd.AddCommand(jobCmd.NewCmdJob(f))
	rootCmd.AddCommand(labelCmd.NewCmdLabel(f))
	rootCmd.AddCommand(mrCmd.NewCmdMR(f))
	rootCmd.AddCommand(pipelineCmd.NewCmdCI(f))
	rootCmd.AddCommand(projectCmd.NewCmdRepo(f))
	rootCmd.AddCommand(releaseCmd.NewCmdRelease(f))
	rootCmd.AddCommand(sshCmd.NewCmdSSHKey(f))
	rootCmd.AddCommand(userCmd.NewCmdUser(f))
	rootCmd.AddCommand(variableCmd.NewVariableCmd(f))
	rootCmd.AddCommand(apiCmd.NewCmdApi(f, nil))
	rootCmd.AddCommand(scheduleCmd.NewCmdSchedule(f))
	rootCmd.AddCommand(securefileCmd.NewCmdSecurefile(f))
	rootCmd.AddCommand(snippetCmd.NewCmdSnippet(f))
	rootCmd.AddCommand(duoCmd.NewCmdDuo(f))
	rootCmd.AddCommand(tokenCmd.NewTokenCmd(f))
	rootCmd.AddCommand(stackCmd.NewCmdStack(f))
	rootCmd.AddCommand(deployKeyCmd.NewCmdDeployKey(f))

	// TODO: This can probably be removed by GitLab 18.3
	// See: https://gitlab.com/gitlab-org/cli/-/issues/7885
	// Add global repo override flag but keep it hidden
	cmdutils.AddGlobalRepoOverride(rootCmd, f)

	rootCmd.Flags().BoolP("version", "v", false, "show glab version information")
	return rootCmd
}
