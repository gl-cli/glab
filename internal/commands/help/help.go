package help

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/utils"
)

// renderWithGlamour renders markdown text using Glamour
func renderWithGlamour(text string) string {
	if text == "" {
		return ""
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(85),
	)
	if err != nil {
		return text
	}

	renderedText, err := renderer.Render(text)
	if err != nil {
		return text
	}

	return strings.TrimSpace(renderedText)
}

// This code forked from the GitHub CLI at http://github.com/cli - cli/internal/text/indent.go
// https://github.com/cli/cli/blob/929e082c13909044e2585af292ae952c9ca6f25c/pkg/text/indent.go

// MIT License
//
// Copyright (c) 2019 GitHub Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

func RootUsageFunc(command *cobra.Command) error {
	command.Printf("Usage:  %s", command.UseLine())

	subcommands := command.Commands()
	if len(subcommands) > 0 {
		command.Print("\n\nAvailable commands:\n")
		for _, c := range subcommands {
			if c.Hidden {
				continue
			}
			command.Printf("  %s\n", c.Name())
		}
		return nil
	}

	flagUsages := command.LocalFlags().FlagUsages()
	if flagUsages != "" {
		command.Println("\n\nFlags:")
		command.Print(utils.Indent(dedent(flagUsages), "  "))
	}
	return nil
}

var hasFailed bool

// HasFailed signals that the main process should exit with non-zero status
func HasFailed() bool {
	return hasFailed
}

// Display helpful error message in case subcommand name was mistyped.
// This matches Cobra's behavior for root command, which Cobra
// confusingly doesn't apply to nested commands.
func nestedSuggestFunc(command *cobra.Command, arg string) {
	command.Printf("unknown command %q for %q\n", arg, command.CommandPath())

	var candidates []string
	if arg == "help" {
		candidates = []string{"--help"}
	} else {
		if command.SuggestionsMinimumDistance <= 0 {
			command.SuggestionsMinimumDistance = 2
		}
		candidates = command.SuggestionsFor(arg)
	}

	if len(candidates) > 0 {
		command.Print("\nDid you mean this?\n")
		for _, c := range candidates {
			command.Printf("\t%s\n", c)
		}
	}

	command.Print("\n")
	_ = RootUsageFunc(command)
}

func isRootCmd(command *cobra.Command) bool {
	return command != nil && !command.HasParent()
}

func RootHelpFunc(c *iostreams.ColorPalette, command *cobra.Command, args []string) {
	originalLong := command.Long
	originalExample := command.Example

	if command.Long != "" {
		command.Long = renderWithGlamour(command.Long)
	}
	if command.Example != "" {
		command.Example = renderWithGlamour(command.Example)
	}

	if isRootCmd(command.Parent()) && len(args) >= 2 && args[1] != "--help" && args[1] != "-h" {
		nestedSuggestFunc(command, args[1])
		hasFailed = true

		command.Long = originalLong
		command.Example = originalExample
		return
	}

	var coreCommands []string
	var additionalCommands []string
	for _, c := range command.Commands() {
		if c.Short == "" {
			continue
		}
		if c.Hidden {
			continue
		}

		s := rpad(c.Name()+":", c.NamePadding()) + c.Short
		if _, ok := c.Annotations["IsCore"]; ok {
			coreCommands = append(coreCommands, s)
		} else {
			additionalCommands = append(additionalCommands, s)
		}
	}

	// If there are no core commands, assume everything is a core command
	if len(coreCommands) == 0 {
		coreCommands = additionalCommands
		additionalCommands = []string{}
	}

	type helpEntry struct {
		Title string
		Body  string
	}

	var helpEntries []helpEntry
	if command.Long != "" {
		helpEntries = append(helpEntries, helpEntry{"", command.Long})
	} else if command.Short != "" {
		helpEntries = append(helpEntries, helpEntry{"", command.Short})
	}
	helpEntries = append(helpEntries, helpEntry{"USAGE", command.UseLine()})
	if len(command.Aliases) > 0 {
		helpEntries = append(helpEntries, helpEntry{"ALIASES", dedent(strings.Join(command.Aliases, ", "))})
	}
	if len(coreCommands) > 0 {
		helpEntries = append(helpEntries, helpEntry{"CORE COMMANDS", strings.Join(coreCommands, "\n")})
	}
	if len(additionalCommands) > 0 {
		helpEntries = append(helpEntries, helpEntry{"ADDITIONAL COMMANDS", strings.Join(additionalCommands, "\n")})
	}

	flagUsages := command.LocalFlags().FlagUsages()
	if flagUsages != "" {
		helpEntries = append(helpEntries, helpEntry{"FLAGS", dedent(flagUsages)})
	}
	inheritedFlagUsages := command.InheritedFlags().FlagUsages()
	if inheritedFlagUsages != "" {
		helpEntries = append(helpEntries, helpEntry{"INHERITED FLAGS", dedent(inheritedFlagUsages)})
	}
	if _, ok := command.Annotations["help:arguments"]; ok {
		helpEntries = append(helpEntries, helpEntry{"ARGUMENTS", command.Annotations["help:arguments"]})
	}
	if command.Example != "" {
		helpEntries = append(helpEntries, helpEntry{"EXAMPLES", command.Example})
	}
	if _, ok := command.Annotations["help:environment"]; ok {
		helpEntries = append(helpEntries, helpEntry{"ENVIRONMENT VARIABLES", command.Annotations["help:environment"]})
	}
	helpEntries = append(helpEntries, helpEntry{"LEARN MORE", `
Use 'glab <command> <subcommand> --help' for more information about a command.`})
	if _, ok := command.Annotations["help:feedback"]; ok {
		helpEntries = append(helpEntries, helpEntry{"FEEDBACK", command.Annotations["help:feedback"]})
	}

	out := command.OutOrStdout()
	for _, e := range helpEntries {
		if e.Title != "" {
			// If there is a title, add indentation to each line in the body
			fmt.Fprintln(out, c.Bold(e.Title))
			fmt.Fprintln(out, utils.Indent(strings.Trim(e.Body, "\r\n"), "  "))
		} else {
			// If there is no title print the body as is
			fmt.Fprintln(out, e.Body)
		}
		fmt.Fprintln(out)
	}

	command.Long = originalLong
	command.Example = originalExample
}

// rpad adds padding to the right of a string.
func rpad(s string, padding int) string {
	template := fmt.Sprintf("%%-%ds ", padding)
	return fmt.Sprintf(template, s)
}

func dedent(s string) string {
	lines := strings.Split(s, "\n")
	minIndent := -1

	for _, l := range lines {
		if l == "" {
			continue
		}

		indent := len(l) - len(strings.TrimLeft(l, " "))
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}

	if minIndent <= 0 {
		return s
	}

	var buf bytes.Buffer
	for _, l := range lines {
		fmt.Fprintln(&buf, strings.TrimPrefix(l, strings.Repeat(" ", minIndent)))
	}
	return strings.TrimSuffix(buf.String(), "\n")
}
