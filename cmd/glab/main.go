package main

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"runtime"
	"strconv"

	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/utils"

	"github.com/mgutz/ansi"

	surveyCore "github.com/AlecAivazis/survey/v2/core"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands"
	"gitlab.com/gitlab-org/cli/internal/commands/alias/expand"
	"gitlab.com/gitlab-org/cli/internal/commands/help"
	"gitlab.com/gitlab-org/cli/internal/commands/update"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/run"
	"gitlab.com/gitlab-org/cli/internal/tableprinter"

	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/spf13/cobra"
)

var (
	// version is set dynamically at build
	version = "DEV"
	// commit is set dynamically at build
	commit string
	// platform is set dynamically at build
	platform = runtime.GOOS

	// debugMode is set dynamically at build and can be overridden by
	// the configuration file or environment variable
	// sets to "true" or "false" or "1" or "0" as string
	debugMode = "false"
)

// gitLabColorScheme returns a custom color scheme using GitLab product colors
// for semantic meaning and accessibility across different terminal backgrounds
func gitLabColorScheme(lightDarkFunc lipgloss.LightDarkFunc) fang.ColorScheme {
	scheme := fang.DefaultColorScheme(lightDarkFunc)

	// GitLab product semantic colors using lightDarkFunc for accessibility
	// Format: lightDarkFunc(lightTerminalColor, darkTerminalColor)

	// Primary brand elements
	gitlabOrange := lightDarkFunc(lipgloss.Color("#FC6D26"), lipgloss.Color("#FC6D26")) // GitLab brand orange
	gitlabPurple := lightDarkFunc(lipgloss.Color("#7759C2"), lipgloss.Color("#A989F5")) // GitLab brand purple

	// Product semantic colors
	gitlabBlue := lightDarkFunc(lipgloss.Color("#1068BF"), lipgloss.Color("#4285F4")) // Blue for current/active states
	_ = lightDarkFunc(lipgloss.Color("#217645"), lipgloss.Color("#34D058"))           // Green for success (unused for now)
	gitlabRed := lightDarkFunc(lipgloss.Color("#C91C00"), lipgloss.Color("#F97583"))  // Red for errors

	// Neutral colors for text and structure
	gitlabText := lightDarkFunc(lipgloss.Color("#171321"), lipgloss.Color("#FAFAFA"))   // High-contrast text
	gitlabSubtle := lightDarkFunc(lipgloss.Color("#6B6B73"), lipgloss.Color("#B0B0B0")) // Subtle elements

	// Error banner configuration to avoid lightDarkFunc
	whiteText := lipgloss.Color("#FFFFFF")
	redBg := lipgloss.Color("#C91C00")

	// Apply GitLab product color semantics
	scheme.Title = gitlabText            // Main command titles
	scheme.Command = gitlabPurple        // Subcommands
	scheme.Flag = gitlabPurple           // Flags - purple for brand consistency
	scheme.FlagDefault = gitlabSubtle    // Default flag values - subtle
	scheme.Description = gitlabText      // Command descriptions - high contrast
	scheme.Program = gitlabOrange        // Program name (glab) - brand consistency
	scheme.Argument = gitlabBlue         // Command arguments - blue for interactive elements
	scheme.DimmedArgument = gitlabSubtle // Optional/dimmed arguments - reduced prominence
	scheme.QuotedString = gitlabText     // Quoted strings - purple accent
	scheme.Comment = gitlabSubtle        // Comments - reduced visual weight
	scheme.Help = gitlabText             // Help text - maximum readability
	scheme.Dash = gitlabSubtle           // Dashes and separators - subtle structure
	scheme.ErrorHeader = [2]color.Color{whiteText, redBg}
	scheme.ErrorDetails = gitlabRed // Error details in GitLab red

	return scheme
}

func main() {
	// Initialize configuration
	cfg, err := config.Init()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read configuration:  %s\n", err)
		os.Exit(2)
	}

	// Set Debug mode from config if not previously set by debugMode
	debug := debugMode == "true" || debugMode == "1"
	if !debug {
		debugModeCfg, _ := cfg.Get("", "debug")
		debug = debugModeCfg == "true" || debugModeCfg == "1"
	}

	// Initialize factory and iostreams
	cmdFactory := cmdutils.NewFactory(
		iostreams.New(
			iostreams.WithStdin(os.Stdin, iostreams.IsTerminal(os.Stdin)),
			iostreams.WithStdout(iostreams.NewColorable(os.Stdout), iostreams.IsTerminal(os.Stdout)),
			iostreams.WithStderr(iostreams.NewColorable(os.Stderr), iostreams.IsTerminal(os.Stderr)),
			iostreams.WithPagerCommand(iostreams.PagerCommandFromEnv()),

			// overwrite pager from env if set via config
			func(i *iostreams.IOStreams) {
				if pager, _ := cfg.Get("", "glab_pager"); pager != "" {
					i.SetPager(pager)
				}
			},

			// configure hyperlink display
			func(i *iostreams.IOStreams) {
				if enabled, found := utils.IsEnvVarEnabled("FORCE_HYPERLINKS"); found {
					if enabled {
						i.SetDisplayHyperlinks("always")
					}

					return
				}

				if enabled, found := utils.IsEnvVarEnabled("GLAB_FORCE_HYPERLINKS"); found {
					if enabled {
						i.SetDisplayHyperlinks("always")
					}

					return
				}

				if displayHyperlinks, _ := cfg.Get("", "display_hyperlinks"); displayHyperlinks == "true" {
					i.SetDisplayHyperlinks("auto")
				}
			},

			// configure prompt
			func(i *iostreams.IOStreams) {
				if value, found := utils.IsEnvVarEnabled("NO_PROMPT"); found {
					i.SetPrompt(strconv.FormatBool(value))
					return
				}

				if value, found := utils.IsEnvVarEnabled("GLAB_NO_PROMPT"); found {
					i.SetPrompt(strconv.FormatBool(value))
					return
				}

				if promptDisabled, _ := cfg.Get("", "no_prompt"); promptDisabled != "" {
					i.SetPrompt(promptDisabled)
				}
			},
		),
		true,
		cfg,
		api.BuildInfo{Version: version, Commit: commit, Platform: platform, Architecture: runtime.GOARCH},
	)

	setupSurveyCore(cmdFactory.IO())

	// Setup command
	var expandedArgs []string
	if len(os.Args) > 0 {
		expandedArgs = os.Args[1:]
	}
	rootCmd := commands.NewCmdRoot(cmdFactory)
	cmd, _, err := rootCmd.Traverse(expandedArgs)

	setupTelemetryHook(cfg, cmdFactory, cmd)

	if err != nil || cmd == rootCmd {
		originalArgs := expandedArgs
		isShell := false
		expandedArgs, isShell, err = expand.ExpandAlias(cfg, os.Args, nil)
		if err != nil {
			cmdFactory.IO().Logf("Failed to process alias: %s\n", err)
			os.Exit(2)
		}

		if debug {
			fmt.Printf("%v -> %v\n", originalArgs, expandedArgs)
		}

		if isShell {
			externalCmd := exec.Command(expandedArgs[0], expandedArgs[1:]...)
			externalCmd.Stderr = os.Stderr
			externalCmd.Stdout = os.Stdout
			externalCmd.Stdin = os.Stdin
			preparedCmd := run.PrepareCmd(externalCmd)

			err = preparedCmd.Run()
			if err != nil {
				if ee, ok := err.(*exec.ExitError); ok {
					os.Exit(ee.ExitCode())
				}

				cmdFactory.IO().Logf("failed to run external command: %s\n", err)
				os.Exit(3)
			}

			os.Exit(0)
		}
	}

	// Override the default column separator of tableprinter to double spaces
	tableprinter.SetTTYSeparator("  ")
	// Override the default terminal width of tableprinter
	tableprinter.SetTerminalWidth(cmdFactory.IO().TerminalWidth())
	// set whether terminal is a TTY or non-TTY
	tableprinter.SetIsTTY(cmdFactory.IO().IsOutputTTY())

	rootCmd.SetArgs(expandedArgs)

	if err := fang.Execute(context.Background(), rootCmd,
		fang.WithoutCompletions(),
		fang.WithoutManpage(),
		fang.WithColorSchemeFunc(gitLabColorScheme),
		fang.WithErrorHandler(cmdutils.GitLabErrorHandler),
	); err != nil {
		var exitError *cmdutils.ExitError
		if errors.As(err, &exitError) {
			os.Exit(exitError.Code)
		} else {
			os.Exit(1)
		}
	}

	if help.HasFailed() {
		os.Exit(1)
	}

	var argCommand string
	if expandedArgs != nil {
		argCommand = expandedArgs[0]
	}
	if !update.ShouldSkipUpdate(argCommand) {
		checkForUpdate(cmdFactory, rootCmd, debug)
	}
}

func setupSurveyCore(io *iostreams.IOStreams) {
	if !io.ColorEnabled() {
		surveyCore.DisableColor = true
	} else {
		// Override survey's choice of color for default values
		// For default values for e.g. `Input` prompts, Survey uses the literal "white" color,
		// which makes no sense on dark terminals and is literally invisible on light backgrounds.
		// This overrides Survey to output a gray color for 256-color terminals and "default" for basic terminals.
		surveyCore.TemplateFuncsWithColor["color"] = func(style string) string {
			switch style {
			case "white":
				if io.Is256ColorSupported() {
					return fmt.Sprintf("\x1b[%d;5;%dm", 38, 242)
				}
				return ansi.ColorCode("default")
			default:
				return ansi.ColorCode(style)
			}
		}
	}
}

func setupTelemetryHook(cfg config.Config, f cmdutils.Factory, cmd *cobra.Command) {
	if isTelemetryEnabled(cfg) {
		cobra.OnFinalize(addTelemetryHook(f, cmd))
	}
}

func checkForUpdate(f cmdutils.Factory, rootCmd *cobra.Command, debug bool) {
	if !isUpdateCheckEnabled(f) {
		return
	}

	if err := update.CheckUpdate(f, true); err != nil {
		update.PrintUpdateError(f.IO(), err, rootCmd, debug)
	}
}

func isUpdateCheckEnabled(f cmdutils.Factory) bool {
	if enabled, found := utils.IsEnvVarEnabled("GLAB_CHECK_UPDATE"); found {
		return enabled
	}

	val, err := f.Config().Get("", "check_update")
	// WARN: I return true here since I think we should always check for updates
	// and an error likely indicates that the value wasn't found in the config.
	if err != nil {
		return true
	}

	checkUpdate, err := strconv.ParseBool(val)
	if err != nil {
		f.IO().Logf("ERROR: Could not parse config value %q: %s", "check_update", err)
	}

	return checkUpdate
}
