package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/pflag"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/commands"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
)

func main() {
	var flagErr pflag.ErrorHandling
	docsCmd := pflag.NewFlagSet("", flagErr)
	manpage := docsCmd.BoolP("manpage", "m", false, "Generate manual pages instead of web docs.")
	path := docsCmd.StringP("path", "p", "./docs/source/", "Path where you want the generated docs saved.")
	help := docsCmd.BoolP("help", "h", false, "Help about any command.")

	if err := docsCmd.Parse(os.Args); err != nil {
		fatal(err)
	}
	if *help {
		_, err := fmt.Fprintf(os.Stderr, "Usage of %s:\n\n%s", os.Args[0], docsCmd.FlagUsages())
		if err != nil {
			fatal(err)
		}
		os.Exit(1)
	}
	err := os.MkdirAll(*path, 0o750)
	if err != nil {
		fatal(err)
	}

	glabCli := commands.NewCmdRoot(&factory{io: iostreams.New()})
	glabCli.DisableAutoGenTag = true
	if *manpage {
		if err := genManPage(glabCli, *path); err != nil {
			fatal(err)
		}
	} else {
		if err := genWebDocs(glabCli, *path); err != nil {
			fatal(err)
		}
	}
}

func genManPage(glabCli *cobra.Command, path string) error {
	header := &doc.GenManHeader{
		Title:   "glab",
		Section: "1",
		Source:  "",
		Manual:  "",
	}
	err := doc.GenManTree(glabCli, header, path)
	if err != nil {
		return err
	}
	return nil
}

func genWebDocs(glabCli *cobra.Command, basePath string) error {
	// Generate docs for root command with special handling
	if err := genRootCommandDocs(glabCli, basePath); err != nil {
		return err
	}

	// Generate docs for all top-level commands
	for _, cmd := range glabCli.Commands() {
		if err := genCommandDocs(cmd, basePath, []string{}); err != nil {
			return err
		}
	}
	return nil
}

// genCommandDocs recursively generates documentation for a command and all its subcommands
// Top-level commands always get their own folder with _index.md
// Non-top-level commands with subcommands get their own folder with _index.md
// Non-top-level commands without subcommands get a .md file in their parent's directory
func genCommandDocs(cmd *cobra.Command, basePath string, parentPath []string) error {
	// Skip help commands and unavailable commands (hidden/deprecated)
	if cmd.Name() == "help" || !cmd.IsAvailableCommand() {
		return nil
	}

	// Build the current command path
	currentPath := append(parentPath, cmd.Name())
	fullPath := filepath.Join(append([]string{basePath}, currentPath...)...)

	fmt.Println("Generating docs for " + strings.Join(currentPath, " "))

	// Create directory for this command
	if err := os.MkdirAll(fullPath, 0o750); err != nil {
		return err
	}

	// Generate the command documentation
	out := new(bytes.Buffer)
	if err := GenMarkdownCustom(cmd, out); err != nil {
		return err
	}

	// Write the documentation file
	// - Top-level commands always get folder/_index.md (even without subcommands)
	// - Non-top-level commands with subcommands get folder/_index.md
	// - Non-top-level commands without subcommands get parent/command.md
	var filename string
	if cmd.HasAvailableSubCommands() || len(parentPath) == 0 {
		// Commands with subcommands OR top-level commands always get folder/_index.md
		filename = filepath.Join(fullPath, "_index.md")
	} else {
		// Non-top-level leaf commands (no subcommands) write as a .md file in the parent directory
		parentDir := filepath.Join(append([]string{basePath}, parentPath...)...)
		filename = filepath.Join(parentDir, cmd.Name()+".md")
	}

	if err := config.WriteFile(filename, out.Bytes(), 0o644); err != nil {
		return err
	}

	// Recursively generate docs for all subcommands
	for _, subCmd := range cmd.Commands() {
		if subCmd.Name() != "help" && subCmd.IsAvailableCommand() {
			if err := genCommandDocs(subCmd, basePath, currentPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// genRootCommandDocs generates documentation for the root command with custom content preservation
func genRootCommandDocs(cmd *cobra.Command, basePath string) error {
	fmt.Println("Generating docs for root command")

	// Generate the root command documentation with custom intro content
	out := new(bytes.Buffer)
	if err := GenRootMarkdownCustom(cmd, out); err != nil {
		return err
	}

	// Write to _index.md in the base path
	filename := filepath.Join(basePath, "_index.md")
	if err := config.WriteFile(filename, out.Bytes(), 0o644); err != nil {
		return err
	}

	return nil
}

func printSubcommands(cmd *cobra.Command, buf *bytes.Buffer) {
	if len(cmd.Commands()) < 1 {
		return
	}

	var subcommands string
	// Generate children commands
	for _, cmdC := range cmd.Commands() {
		if cmdC.Name() != "help" && cmdC.IsAvailableCommand() {
			if cmdC.HasAvailableSubCommands() {
				subcommands += fmt.Sprintf("- [`%s`](%s/_index.md)\n", cmdC.Name(), cmdC.Name())
			} else {
				subcommands += fmt.Sprintf("- [`%s`](%s.md)\n", cmdC.Name(), cmdC.Name())
			}
		}
	}

	if subcommands != "" {
		buf.WriteString("\n## Subcommands\n\n")
		buf.WriteString(subcommands)
	}
}

func printRootSubcommands(cmd *cobra.Command, buf *bytes.Buffer) {
	if len(cmd.Commands()) < 1 {
		return
	}

	var subcommands string
	// Generate children commands for root
	// All top-level commands get directories and _index.md files based on the generation logic
	for _, cmdC := range cmd.Commands() {
		if cmdC.Name() != "help" && cmdC.IsAvailableCommand() {
			subcommands += fmt.Sprintf("- [`glab %s`](%s/_index.md)\n", cmdC.Name(), cmdC.Name())
		}
	}

	if subcommands != "" {
		buf.WriteString("\n## Commands\n\n")
		buf.WriteString(subcommands)
	}
}

// printRootOptions is like printOptions but without the leading newline for root command
func printRootOptions(buf *bytes.Buffer, cmd *cobra.Command) error {
	flags := cmd.NonInheritedFlags()
	flags.SetOutput(buf)
	if flags.HasAvailableFlags() {
		buf.WriteString("## Options\n\n```plaintext\n")
		flags.PrintDefaults()
		buf.WriteString("```\n")
	}

	parentFlags := cmd.InheritedFlags()
	parentFlags.SetOutput(buf)
	if parentFlags.HasAvailableFlags() {
		buf.WriteString("\n## Options inherited from parent commands\n\n```plaintext\n")
		parentFlags.PrintDefaults()
		buf.WriteString("```\n")
	}
	return nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

// adapted from: github.com/spf13/cobra/blob/main/doc/md_docs.go
// GenMarkdownTreeCustom is the the same as GenMarkdownTree, but
// with custom filePrepender and linkHandler.
func GenMarkdownTreeCustom(cmd *cobra.Command, dir string) error {
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		if err := GenMarkdownTreeCustom(c, dir); err != nil {
			return err
		}
	}

	basename := strings.ReplaceAll(cmd.Name(), " ", "_") + ".md"
	filename := filepath.Join(dir, basename)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := GenMarkdownCustom(cmd, f); err != nil {
		return err
	}
	return nil
}

// GenMarkdownCustom creates custom Markdown output. github.com/spf13/cobra/blob/main/doc/md_docs.go
func GenMarkdownCustom(cmd *cobra.Command, w io.Writer) error {
	cmd.InitDefaultHelpCmd()
	cmd.InitDefaultHelpFlag()

	buf := new(bytes.Buffer)
	name := cmd.CommandPath()
	// GitLab Specific Docs Metadata
	buf.WriteString("---" + "\n")
	buf.WriteString("title: " + name + "\n")
	buf.WriteString("stage: Create" + "\n")
	buf.WriteString("group: Code Review" + "\n")
	buf.WriteString("info: To determine the technical writer assigned to the Stage/Group associated with this page, see https://about.gitlab.com/handbook/product/ux/technical-writing/#assignments" + "\n")
	buf.WriteString("---" + "\n\n")

	// Generated by a script
	buf.WriteString("<!--" + "\n")
	buf.WriteString("This documentation is auto generated by a script." + "\n")
	buf.WriteString("Please do not edit this file directly. Run `make gen-docs` instead." + "\n")
	buf.WriteString("-->" + "\n\n")

	buf.WriteString(cmd.Short + "\n")
	if len(cmd.Long) > 0 {
		// Skipping `help` commands until Long description can be revised
		if cmd.Name() != "help" {
			buf.WriteString("\n## Synopsis\n\n")
			buf.WriteString(cmd.Long)
		}
	}

	if cmd.Runnable() {
		fmt.Fprintf(buf, "\n```plaintext\n%s\n```\n", cmd.UseLine())
	}

	if len(cmd.Aliases) > 0 {
		buf.WriteString("\n## Aliases\n\n")
		fmt.Fprintf(buf, "```plaintext\n%s\n```\n", strings.Join(cmd.Aliases, "\n"))
	}

	if len(cmd.Example) > 0 {
		buf.WriteString("\n## Examples\n\n")
		fmt.Fprintf(buf, "```console\n%s\n```\n", cmd.Example)
	}

	if err := printOptions(buf, cmd); err != nil {
		return err
	}

	printSubcommands(cmd, buf)

	_, err := buf.WriteTo(w)
	return err
}

// adapted from: github.com/spf13/cobra/blob/main/doc/md_docs.go
func printOptions(buf *bytes.Buffer, cmd *cobra.Command) error {
	flags := cmd.NonInheritedFlags()
	flags.SetOutput(buf)
	if flags.HasAvailableFlags() {
		buf.WriteString("\n## Options\n\n```plaintext\n")
		flags.PrintDefaults()
		buf.WriteString("```\n")
	}

	parentFlags := cmd.InheritedFlags()
	parentFlags.SetOutput(buf)
	if parentFlags.HasAvailableFlags() {
		buf.WriteString("\n## Options inherited from parent commands\n\n```plaintext\n")
		parentFlags.PrintDefaults()
		buf.WriteString("```\n")
	}
	return nil
}

// GenRootMarkdownCustom creates custom markdown for the root command with preserved intro content
func GenRootMarkdownCustom(cmd *cobra.Command, w io.Writer) error {
	buf := new(bytes.Buffer)
	// GitLab Specific Docs Metadata
	buf.WriteString("---" + "\n")
	buf.WriteString("title: GitLab CLI (glab)\n")
	buf.WriteString("stage: Create" + "\n")
	buf.WriteString("group: Code Review" + "\n")
	buf.WriteString("info: To determine the technical writer assigned to the Stage/Group associated with this page, see https://about.gitlab.com/handbook/product/ux/technical-writing/#assignments" + "\n")
	buf.WriteString("---" + "\n\n")

	// Generated by a script comment
	buf.WriteString("<!--" + "\n")
	buf.WriteString("This documentation is auto generated by a script." + "\n")
	buf.WriteString("Please do not edit this file directly. Run `make gen-docs` instead." + "\n")
	buf.WriteString("-->" + "\n\n")

	// Add details shortcode
	buf.WriteString("{{< details >}}\n\n")
	buf.WriteString("- Tier: Free, Premium, Ultimate\n")
	buf.WriteString("- Offering: GitLab.com, GitLab Self-Managed, GitLab Dedicated\n\n")
	buf.WriteString("{{< /details >}}\n\n")

	// Custom title and intro content (from README)
	buf.WriteString("GLab is an open source GitLab CLI tool. It brings GitLab to your terminal, next to where you are already working with `git` and your code, without switching between windows and browser tabs. While it's powerful for issues and merge requests, `glab` does even more:\n\n")
	buf.WriteString("- View, manage, and retry CI/CD pipelines directly from your CLI.\n")
	buf.WriteString("- Create changelogs.\n")
	buf.WriteString("- Create and manage releases.\n")
	buf.WriteString("- Ask GitLab Duo Chat questions about Git.\n")
	buf.WriteString("- Manage GitLab agents for Kubernetes.\n\n")
	buf.WriteString("`glab` is available for repositories hosted on GitLab.com, GitLab Dedicated, and GitLab Self-Managed. It supports multiple authenticated GitLab instances, and automatically detects the authenticated hostname from the remotes available in your working Git directory.\n\n")
	buf.WriteString("![command example](img/glabgettingstarted.gif)\n\n")

	// Install and authenticate sections (moved up, with README links)
	buf.WriteString("## Install the CLI\n\n")
	buf.WriteString("Installation instructions are available in the GLab\n")
	buf.WriteString("[`README`](https://gitlab.com/gitlab-org/cli/#installation).\n\n")

	buf.WriteString("## Authenticate with GitLab\n\n")
	buf.WriteString("GLab supports multiple authentication methods including OAuth and personal access tokens.\n")
	buf.WriteString("To get started, run `glab auth login` and follow the interactive setup.\n\n")
	buf.WriteString("For detailed authentication instructions, see the [Authentication section](https://gitlab.com/gitlab-org/cli#authentication)\n")
	buf.WriteString("in the main README.\n\n")

	if cmd.Runnable() {
		fmt.Fprintf(buf, "```plaintext\n%s\n```\n\n", cmd.UseLine())
	}

	// Generate environment variables section from annotations with table formatting
	if envHelp, ok := cmd.Annotations["help:environment"]; ok {
		buf.WriteString("## Environment Variables\n\n")
		buf.WriteString("<!-- markdownlint-disable MD044 MD034 -->\n")
		buf.WriteString("| Variable | Description |\n")
		buf.WriteString("|----------|-------------|\n")

		// Parse the environment variables from the annotation
		envLines := strings.Split(strings.TrimSpace(envHelp), "\n\n")
		for _, envBlock := range envLines {
			lines := strings.Split(envBlock, "\n")
			if len(lines) > 0 {
				firstLine := strings.TrimSpace(lines[0])
				if strings.Contains(firstLine, ":") {
					parts := strings.SplitN(firstLine, ":", 2)
					if len(parts) == 2 {
						varName := strings.TrimSpace(parts[0])
						description := strings.TrimSpace(parts[1])

						// Add additional lines to description
						for i := 1; i < len(lines); i++ {
							additionalLine := strings.TrimSpace(lines[i])
							if additionalLine != "" {
								description += " " + additionalLine
							}
						}

						buf.WriteString(fmt.Sprintf("| `%s` | %s |\n", varName, description))
					}
				}
			}
		}
		buf.WriteString("<!-- markdownlint-enable MD044 MD034 -->\n")
		buf.WriteString("\n")
	}
	if err := printRootOptions(buf, cmd); err != nil {
		return err
	}

	// Generate subcommands (replaces manual "Core commands")
	printRootSubcommands(cmd, buf)

	// Add report issues section at the end
	buf.WriteString("\n## Report issues\n\n")
	buf.WriteString("Open an issue in the [`gitlab-org/cli` repository](https://gitlab.com/gitlab-org/cli/issues/new)\n")
	buf.WriteString("to send us feedback.\n")

	_, err := buf.WriteTo(w)
	return err
}

type factory struct {
	io *iostreams.IOStreams
}

func (f *factory) RepoOverride(repo string) error {
	return nil
}

func (f *factory) ApiClient(repoHost string) (*api.Client, error) {
	return nil, errors.New("not implemented")
}

func (f *factory) GitLabClient() (*gitlab.Client, error) {
	return nil, errors.New("not implemented")
}

func (f *factory) BaseRepo() (glrepo.Interface, error) {
	return nil, errors.New("not implemented")
}

func (f *factory) Remotes() (glrepo.Remotes, error) {
	return nil, errors.New("not implemented")
}

func (f *factory) Config() config.Config {
	return nil
}

func (f *factory) Branch() (string, error) {
	return "", errors.New("not implemented")
}

func (f *factory) IO() *iostreams.IOStreams {
	return f.io
}

func (f *factory) DefaultHostname() string {
	return glinstance.DefaultHostname
}

func (f *factory) BuildInfo() api.BuildInfo {
	return api.BuildInfo{}
}
