package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/mgutz/ansi"

	surveyCore "github.com/AlecAivazis/survey/v2/core"
	"gitlab.com/gitlab-org/cli/commands"
	"gitlab.com/gitlab-org/cli/commands/alias/expand"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/help"
	"gitlab.com/gitlab-org/cli/commands/hooks"
	"gitlab.com/gitlab-org/cli/commands/update"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/run"
	"gitlab.com/gitlab-org/cli/pkg/glinstance"
	"gitlab.com/gitlab-org/cli/pkg/tableprinter"
	"gitlab.com/gitlab-org/cli/pkg/utils"

	"github.com/spf13/cobra"
)

var (
	// version is set dynamically at build
	version = "DEV"
	// commit is set dynamically at build
	commit string
	// platform is set dynamically at build
	platform = runtime.GOOS
)

// debug is set dynamically at build and can be overridden by
// the configuration file or environment variable
// sets to "true" or "false" or "1" or "0" as string
var debugMode = "false"

// debug is parsed boolean of debugMode
var debug bool

func main() {
	debug = debugMode == "true" || debugMode == "1"

	cmdFactory := cmdutils.NewFactory(iostreams.Init(), true)

	cfg, err := cmdFactory.Config()
	if err != nil {
		cmdFactory.IO().Logf("failed to read configuration:  %s\n", err)
		os.Exit(2)
	}

	api.SetUserAgent(version, platform, runtime.GOARCH)
	maybeOverrideDefaultHost(cmdFactory, cfg)

	if !cmdFactory.IO().ColorEnabled() {
		surveyCore.DisableColor = true
	} else {
		// Override survey's choice of color for default values
		// For default values for e.g. `Input` prompts, Survey uses the literal "white" color,
		// which makes no sense on dark terminals and is literally invisible on light backgrounds.
		// This overrides Survey to output a gray color for 256-color terminals and "default" for basic terminals.
		surveyCore.TemplateFuncsWithColor["color"] = func(style string) string {
			switch style {
			case "white":
				if cmdFactory.IO().Is256ColorSupported() {
					return fmt.Sprintf("\x1b[%d;5;%dm", 38, 242)
				}
				return ansi.ColorCode("default")
			default:
				return ansi.ColorCode(style)
			}
		}
	}

	rootCmd := commands.NewCmdRoot(cmdFactory, version, commit)

	// Set Debug mode from config if not previously set by debugMode
	if !debug {
		debugModeCfg, _ := cfg.Get("", "debug")
		debug = debugModeCfg == "true" || debugModeCfg == "1"
	}

	if pager, _ := cfg.Get("", "glab_pager"); pager != "" {
		cmdFactory.IO().SetPager(pager)
	}

	if promptDisabled, _ := cfg.Get("", "no_prompt"); promptDisabled != "" {
		cmdFactory.IO().SetPrompt(promptDisabled)
	}

	if forceHyperlinks := os.Getenv("FORCE_HYPERLINKS"); forceHyperlinks != "" && forceHyperlinks != "0" {
		cmdFactory.IO().SetDisplayHyperlinks("always")
	} else if displayHyperlinks, _ := cfg.Get("", "display_hyperlinks"); displayHyperlinks == "true" {
		cmdFactory.IO().SetDisplayHyperlinks("auto")
	}

	var expandedArgs []string
	if len(os.Args) > 0 {
		expandedArgs = os.Args[1:]
	}

	cmd, _, err := rootCmd.Traverse(expandedArgs)

	checkForTelemetryHook(cfg, cmdFactory, cmd)

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

	if cmd, err := rootCmd.ExecuteC(); err != nil {
		if !errors.Is(err, cmdutils.SilentError) {
			printError(cmdFactory.IO(), err, cmd, debug)
		}

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
	} else {
		argCommand = ""
	}

	shouldCheck := false

	// GLAB_CHECK_UPDATE has higher priority than the check_update configuration value
	if envVal, ok := os.LookupEnv("GLAB_CHECK_UPDATE"); ok {
		if checkUpdate, err := strconv.ParseBool(envVal); err == nil {
			shouldCheck = checkUpdate
		}
	} else {
		// Fall back to config value if env var not set
		if checkUpdate, _ := cfg.Get("", "check_update"); checkUpdate != "" {
			if parsed, err := strconv.ParseBool(checkUpdate); err == nil {
				shouldCheck = parsed
			}
		}
	}

	if shouldCheck {
		if err := update.CheckUpdate(cmdFactory, version, true, argCommand); err != nil {
			printError(cmdFactory.IO(), err, rootCmd, debug)
		}
	}
}

func printError(streams *iostreams.IOStreams, err error, cmd *cobra.Command, debug bool) {
	color := streams.Color()

	var dnsError *net.DNSError
	if errors.As(err, &dnsError) {
		streams.Logf("%s error connecting to %s\n", color.FailedIcon(), dnsError.Name)
		if debug {
			streams.Log(color.FailedIcon(), dnsError)
		}
		streams.Logf("%s Check your internet connection and status.gitlab.com. If on GitLab Self-Managed, run 'sudo gitlab-ctl status' on your server.\n", color.DotWarnIcon())
	} else {
		var exitError *cmdutils.ExitError
		if errors.As(err, &exitError) {
			streams.Logf("%s %s %s=%s\n", color.FailedIcon(), color.Bold(exitError.Details), color.Red("error"), exitError.Err)
		} else {
			streams.Log("ERROR:", err)

			var flagError *cmdutils.FlagError
			if errors.As(err, &flagError) || strings.HasPrefix(err.Error(), "unknown command ") {
				streams.Logf("Try '%s --help' for more information.", cmd.CommandPath())
			}

		}
	}

	if cmd != nil {
		cmd.Print("\n")
	}
}

func maybeOverrideDefaultHost(f cmdutils.Factory, cfg config.Config) {
	baseRepo, err := f.BaseRepo()
	if err == nil {
		glinstance.OverrideDefault(baseRepo.RepoHost())
	}

	// Fetch the custom host config from env vars, then local config.yml, then global config,yml.
	customGLHost, _ := cfg.Get("", "host")
	if customGLHost != "" {
		if utils.IsValidURL(customGLHost) {
			var protocol string
			customGLHost, protocol = glinstance.StripHostProtocol(customGLHost)
			glinstance.OverrideDefaultProtocol(protocol)
		}
		glinstance.OverrideDefault(customGLHost)
	}
}

func checkForTelemetryHook(cfg config.Config, f cmdutils.Factory, cmd *cobra.Command) {
	if hooks.IsTelemetryEnabled(cfg) {
		cobra.OnFinalize(hooks.AddTelemetryHook(f, cmd))
	}
}
