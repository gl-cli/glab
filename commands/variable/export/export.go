package export

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/flag"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
)

type ExportOpts struct {
	HTTPClient func() (*gitlab.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (glrepo.Interface, error)

	ValueSet     bool
	Group        string
	OutputFormat string
	Scope        string

	Page    int
	PerPage int
}

func marshalJson(variables interface{}) ([]byte, error) {
	res, err := json.MarshalIndent(variables, "", "  ")
	if err != nil {
		return nil, err
	}

	return res, nil
}

func NewCmdExport(f *cmdutils.Factory, runE func(opts *ExportOpts) error) *cobra.Command {
	opts := &ExportOpts{
		IO: f.IO,
	}

	cmd := &cobra.Command{
		Use:     "export",
		Short:   "Export variables from a project or group.",
		Aliases: []string{"ex"},
		Args:    cobra.ExactArgs(0),
		Example: heredoc.Doc(`
                        glab variable export
                        glab variable export --per-page 1000 --page 1
                        glab variable export --group gitlab-org
                        glab variable export --group gitlab-org --per-page 1000 --page 1
                `),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			// Supports repo override
			opts.HTTPClient = f.HttpClient
			opts.BaseRepo = f.BaseRepo

			group, err := flag.GroupOverride(cmd)
			if err != nil {
				return err
			}
			opts.Group = group

			if runE != nil {
				err = runE(opts)
				return
			}
			err = exportRun(opts)
			return
		},
	}

	cmdutils.EnableRepoOverride(cmd, f)
	cmd.PersistentFlags().StringP("group", "g", "", "Select a group or subgroup. Ignored if a repository argument is set.")
	cmd.Flags().IntVarP(&opts.Page, "page", "p", 1, "Page number.")
	cmd.Flags().IntVarP(&opts.PerPage, "per-page", "P", 100, "Number of items to list per page.")
	cmd.Flags().StringVarP(&opts.OutputFormat, "format", "F", "json", "Format of output: json, export, env.")
	cmd.Flags().StringVarP(&opts.Scope, "scope", "s", "*", "The environment_scope of the variables. Values: '*' (default), or specific environments.")

	return cmd
}

func exportRun(opts *ExportOpts) error {
	var out io.Writer = os.Stdout
	if opts.IO != nil && opts.IO.StdOut != nil {
		out = opts.IO.StdOut
	}

	httpClient, err := opts.HTTPClient()
	if err != nil {
		return err
	}

	repo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	if opts.Group != "" {
		createVarOpts := &gitlab.ListGroupVariablesOptions{Page: opts.Page, PerPage: opts.PerPage}
		groupVariables, err := api.ListGroupVariables(httpClient, opts.Group, createVarOpts)
		if err != nil {
			return err
		}

		opts.IO.Logf("Exporting variables from the %s group:\n", opts.Group)

		if len(groupVariables) == 0 {
			return nil
		}

		return printGroupVariables(groupVariables, opts, out)

	} else {
		createVarOpts := &gitlab.ListProjectVariablesOptions{Page: opts.Page, PerPage: opts.PerPage}
		projectVariables, err := api.ListProjectVariables(httpClient, repo.FullName(), createVarOpts)
		if err != nil {
			return err
		}

		opts.IO.Logf("Exporting variables from the %s project:\n", repo.FullName())

		if len(projectVariables) == 0 {
			return nil
		}

		return printProjectVariables(projectVariables, opts, out)
	}
}

func matchesScope(varScope, optScope string) bool {
	if varScope == "*" || optScope == "*" {
		return true
	}
	if varScope == optScope {
		return true
	}
	if strings.Contains(varScope, "*") {
		varPattern := "^" + regexp.QuoteMeta(varScope) + "$"
		optPattern := "^" + regexp.QuoteMeta(optScope) + "$"

		varPattern = strings.ReplaceAll(varPattern, `\*`, ".*")
		optPattern = strings.ReplaceAll(optPattern, `\*`, ".*")

		matchesVar, _ := regexp.MatchString(varPattern, optScope)
		matchesOpt, _ := regexp.MatchString(optPattern, varScope)

		return matchesVar || matchesOpt
	}
	return false
}

func isValidEnvironmentScope(optScope string) bool {
	pattern := `^[a-zA-Z0-9\s\-_/${}\x20]+$`
	re, _ := regexp.Compile(pattern)
	matched := re.MatchString(optScope)
	return matched || optScope == "*"
}

func printGroupVariables(variables []*gitlab.GroupVariable, opts *ExportOpts, out io.Writer) error {
	if !isValidEnvironmentScope((opts.Scope)) {
		return fmt.Errorf("invalid environment scope: %s", opts.Scope)
	}

	writtenKeys := make([]string, 0)
	switch opts.OutputFormat {
	case "env":
		for _, variable := range variables {
			if matchesScope(variable.EnvironmentScope, opts.Scope) {
				if !strings.Contains(variable.EnvironmentScope, "*") {
					fmt.Fprintf(out, "%s=%s\n", variable.Key, variable.Value)
					writtenKeys = append(writtenKeys, variable.Key)
				}
			}
		}
		keysMap := CreateWrittenKeysMap(writtenKeys)
		for _, variable := range variables {
			if matchesScope(variable.EnvironmentScope, opts.Scope) {
				if !(keysMap[variable.Key]) && (strings.Contains(variable.EnvironmentScope, "*")) {
					fmt.Fprintf(out, "%s=%s\n", variable.Key, variable.Value)
				}
			}
		}
	case "export":
		for _, variable := range variables {
			if matchesScope(variable.EnvironmentScope, opts.Scope) {
				if !strings.Contains(variable.EnvironmentScope, "*") {
					fmt.Fprintf(out, "export %s=%s\n", variable.Key, variable.Value)
					writtenKeys = append(writtenKeys, variable.Key)
				}
			}
		}
		keysMap := CreateWrittenKeysMap(writtenKeys)
		for _, variable := range variables {
			if matchesScope(variable.EnvironmentScope, opts.Scope) {
				if !(keysMap[variable.Key]) && (strings.Contains(variable.EnvironmentScope, "*")) {
					fmt.Fprintf(out, "export %s=%s\n", variable.Key, variable.Value)
				}
			}
		}
	case "json":
		filteredVariables := make([]*gitlab.GroupVariable, 0)
		for _, variable := range variables {
			if matchesScope(variable.EnvironmentScope, opts.Scope) {
				filteredVariables = append(filteredVariables, variable)
			}
		}
		res, err := marshalJson(filteredVariables)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(res))
	default:
		return fmt.Errorf("unsupported output format: %s", opts.OutputFormat)
	}

	return nil
}

func printProjectVariables(variables []*gitlab.ProjectVariable, opts *ExportOpts, out io.Writer) error {
	if !isValidEnvironmentScope((opts.Scope)) {
		return fmt.Errorf("invalid environment scope: %s", opts.Scope)
	}

	writtenKeys := make([]string, 0)
	switch opts.OutputFormat {
	case "env":
		for _, variable := range variables {
			if matchesScope(variable.EnvironmentScope, opts.Scope) {
				if !strings.Contains(variable.EnvironmentScope, "*") {
					fmt.Fprintf(out, "%s=\"%s\"\n", variable.Key, variable.Value)
					writtenKeys = append(writtenKeys, variable.Key)
				}
			}
		}
		keysMap := CreateWrittenKeysMap(writtenKeys)
		for _, variable := range variables {
			if matchesScope(variable.EnvironmentScope, opts.Scope) {
				if !(keysMap[variable.Key]) && (strings.Contains(variable.EnvironmentScope, "*")) {
					fmt.Fprintf(out, "%s=\"%s\"\n", variable.Key, variable.Value)
				}
			}
		}
	case "export":
		for _, variable := range variables {
			if matchesScope(variable.EnvironmentScope, opts.Scope) {
				if !strings.Contains(variable.EnvironmentScope, "*") {
					fmt.Fprintf(out, "export %s=\"%s\"\n", variable.Key, variable.Value)
					writtenKeys = append(writtenKeys, variable.Key)
				}
			}
		}
		keysMap := CreateWrittenKeysMap(writtenKeys)
		for _, variable := range variables {
			if matchesScope(variable.EnvironmentScope, opts.Scope) {
				if !(keysMap[variable.Key]) && (strings.Contains(variable.EnvironmentScope, "*")) {
					fmt.Fprintf(out, "export %s=\"%s\"\n", variable.Key, variable.Value)
				}
			}
		}
	case "json":
		filteredVariables := make([]*gitlab.ProjectVariable, 0)
		for _, variable := range variables {
			if matchesScope(variable.EnvironmentScope, opts.Scope) {
				filteredVariables = append(filteredVariables, variable)
			}
		}
		res, err := marshalJson(filteredVariables)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(res))
	default:
		return fmt.Errorf("unsupported output format: %s", opts.OutputFormat)
	}

	return nil
}

func CreateWrittenKeysMap(writtenKeys []string) map[string]bool {
	keysMap := make(map[string]bool)
	for _, key := range writtenKeys {
		keysMap[key] = true
	}
	return keysMap
}
