package create

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
)

type CreateOpts struct {
	Title           string
	Description     string
	DisplayFilename string
	Visibility      string
	Personal        bool

	FilePath string

	IO       *iostreams.IOStreams
	Lab      func() (*gitlab.Client, error)
	BaseRepo func() (glrepo.Interface, error)
}

func (opts CreateOpts) isSnippetFromFile() bool {
	return opts.FilePath != ""
}

func hasStdIn() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	return fi.Size() > 0
}

func NewCmdCreate(f *cmdutils.Factory) *cobra.Command {
	opts := &CreateOpts{}
	snippetCreateCmd := &cobra.Command{
		Use:     "create [path]",
		Short:   `Create a new snippet.`,
		Long:    ``,
		Aliases: []string{"new"},
		Example: heredoc.Doc(`
			glab snippet create script.py --title "Title of the snippet"
			echo "package main" | glab snippet new --title "Title of the snippet" --filename "main.go"
			glab snippet create main.go -t Title -f "different.go" -d Description
			glab snippet create main.go -t Title -f "different.go" -d Description --filename different.go
			glab snippet create script.py --personal --title "Personal snippet"
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.IO = f.IO
			opts.BaseRepo = f.BaseRepo
			opts.Lab = f.HttpClient
			if opts.Title == "" {
				return &cmdutils.FlagError{
					Err: errors.New("--title required for snippets"),
				}
			}
			if len(args) == 0 {
				if opts.DisplayFilename == "" {
					return &cmdutils.FlagError{Err: errors.New("if 'path' is not provided, 'filename' and stdin are required")}
				} else {
					if !hasStdIn() {
						return errors.New("stdin required if no 'path' is provided")
					}
				}
			} else {
				if opts.DisplayFilename == "" {
					opts.DisplayFilename = args[0]
				}
				opts.FilePath = args[0]
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.Lab()
			if err != nil {
				return err
			}
			if opts.Personal {
				return runCreate(client, nil, opts)
			} else {
				repo, err := opts.BaseRepo()
				if err != nil {
					return err
				}
				return runCreate(client, repo, opts)

			}
		},
	}

	snippetCreateCmd.Flags().StringVarP(&opts.Title, "title", "t", "", "Title of the snippet.")
	snippetCreateCmd.Flags().StringVarP(&opts.DisplayFilename, "filename", "f", "", "Filename of the snippet in GitLab.")
	snippetCreateCmd.Flags().StringVarP(&opts.Description, "description", "d", "", "Description of the snippet.")
	snippetCreateCmd.Flags().StringVarP(&opts.Visibility, "visibility", "v", "private", "Limit by visibility: 'public', 'internal', or 'private'")
	snippetCreateCmd.Flags().BoolVarP(&opts.Personal, "personal", "p", false, "Create a personal snippet.")

	return snippetCreateCmd
}

func runCreate(client *gitlab.Client, repo glrepo.Interface, opts *CreateOpts) error {
	content, err := readSnippetsContent(opts)
	if err != nil {
		return err
	}
	var snippet *gitlab.Snippet
	if opts.Personal {
		fmt.Fprintln(opts.IO.StdErr, "- Creating snippet in personal space")
		snippet, err = api.CreateSnippet(client, &gitlab.CreateSnippetOptions{
			Title:       &opts.Title,
			Description: &opts.Description,
			Content:     gitlab.Ptr(string(content)),
			FileName:    &opts.DisplayFilename,
			Visibility:  gitlab.Ptr(gitlab.VisibilityValue(opts.Visibility)),
		})
	} else {
		fmt.Fprintln(opts.IO.StdErr, "- Creating snippet in", repo.FullName())
		snippet, err = api.CreateProjectSnippet(client, repo.FullName(), &gitlab.CreateProjectSnippetOptions{
			Title:       &opts.Title,
			Description: &opts.Description,
			Content:     gitlab.Ptr(string(content)),
			FileName:    &opts.DisplayFilename,
			Visibility:  gitlab.Ptr(gitlab.VisibilityValue(opts.Visibility)),
		})
	}
	if err != nil {
		return fmt.Errorf("failed to create snippet: %w", err)
	}
	snippetID := opts.IO.Color().Green(fmt.Sprintf("$%d", snippet.ID))
	if opts.IO.IsaTTY {
		fmt.Fprintf(opts.IO.StdOut, "%s %s (%s)\n %s\n", snippetID, snippet.Title, snippet.FileName, snippet.WebURL)
	} else {
		fmt.Fprintln(opts.IO.StdOut, snippet.WebURL)
	}

	return nil
}

// FIXME: Adding more then one file can't be done right now because the GitLab API library
//
//	Doesn't support it yet.
//
//	See for the API reference: https://docs.gitlab.com/ee/api/snippets.html#create-new-snippet
//	See for the library docs: https://pkg.go.dev/github.com/xanzy/go-gitlab#CreateSnippetOptions
//	See for GitHub issue: https://github.com/xanzy/go-gitlab/issues/1372
func readSnippetsContent(opts *CreateOpts) (string, error) {
	if opts.isSnippetFromFile() {
		return readFromFile(opts.FilePath)
	}
	return readFromSTDIN(opts.IO)
}

func readFromSTDIN(ioStream *iostreams.IOStreams) (string, error) {
	content, err := io.ReadAll(ioStream.In)
	if err != nil {
		return "", fmt.Errorf("Failed to read snippet from STDIN. %w", err)
	}
	return string(content), nil
}

func readFromFile(filename string) (string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("Failed to read snippet from file '%s'. %w", filename, err)
	}
	return string(content), nil
}
