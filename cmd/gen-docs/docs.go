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
	// Generate docs for all top-level commands
	for _, cmd := range glabCli.Commands() {
		if err := genCommandDocs(cmd, basePath, []string{}); err != nil {
			return err
		}
	}
	return nil
}

// genCommandDocs recursively generates documentation for a command and all its subcommands
// Top-level commands always get their own folder with index.md
// Non-top-level commands with subcommands get their own folder with index.md
// Non-top-level commands without subcommands get a .md file in their parent's directory
func genCommandDocs(cmd *cobra.Command, basePath string, parentPath []string) error {
	// Skip help commands
	if cmd.Name() == "help" {
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
	// - Top-level commands always get folder/index.md (even without subcommands)
	// - Non-top-level commands with subcommands get folder/index.md
	// - Non-top-level commands without subcommands get parent/command.md
	var filename string
	if cmd.HasAvailableSubCommands() || len(parentPath) == 0 {
		// Commands with subcommands OR top-level commands always get folder/index.md
		filename = filepath.Join(fullPath, "index.md")
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
		if subCmd.Name() != "help" {
			if err := genCommandDocs(subCmd, basePath, currentPath); err != nil {
				return err
			}
		}
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
		if cmdC.Name() != "help" {
			if cmdC.HasAvailableSubCommands() {
				subcommands += fmt.Sprintf("- [`%s`](%s/index.md)\n", cmdC.Name(), cmdC.Name())
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
	buf.WriteString("stage: Create" + "\n")
	buf.WriteString("group: Code Review" + "\n")
	buf.WriteString("info: To determine the technical writer assigned to the Stage/Group associated with this page, see https://about.gitlab.com/handbook/product/ux/technical-writing/#assignments" + "\n")
	buf.WriteString("---" + "\n\n")

	// Generated by a script
	buf.WriteString("<!--" + "\n")
	buf.WriteString("This documentation is auto generated by a script." + "\n")
	buf.WriteString("Please do not edit this file directly. Run `make gen-docs` instead." + "\n")
	buf.WriteString("-->" + "\n\n")

	buf.WriteString("# `" + name + "`\n\n")
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
