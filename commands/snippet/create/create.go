package create

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/dbg"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
)

type CreateOpts struct {
	Title           string
	Description     string
	DisplayFilename string
	Visibility      string
	Personal        bool

	Files []*gitlab.CreateSnippetFileOptions

	IO       *iostreams.IOStreams
	BaseRepo func() (glrepo.Interface, error)
}

func (opts *CreateOpts) addFile(path, content *string) {
	opts.Files = append(opts.Files, &gitlab.CreateSnippetFileOptions{
		FilePath: path,
		Content:  content,
	})
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
		Use: `create [flags] -t <title> <file1> [<file2>...]
glab snippet create [flags] -t <title> -f <filename>  # reads from stdin`,
		Short:   `Create a new snippet.`,
		Long:    ``,
		Aliases: []string{"new"},
		Example: heredoc.Doc(`
			- glab snippet create script.py --title "Title of the snippet"
			- echo "package main" | glab snippet new --title "Title of the snippet" --filename "main.go"
			- glab snippet create -t Title -f "different.go" -d Description main.go
			- glab snippet create -t Title -f "different.go" -d Description --filename different.go main.go
			- glab snippet create --personal --title "Personal snippet" script.py
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.IO = f.IO
			opts.BaseRepo = f.BaseRepo
			if opts.Title == "" {
				return &cmdutils.FlagError{
					Err: errors.New("--title required for snippets"),
				}
			}
			if len(args) == 0 {
				if opts.DisplayFilename == "" {
					return &cmdutils.FlagError{Err: errors.New("if 'path' is not provided, 'filename' and stdin are required")}
				} else {
					if !f.IO.IsInTTY && !hasStdIn() {
						return errors.New("stdin required if no 'path' is provided")
					}
				}
				fmt.Fprintln(f.IO.StdOut, "reading from stdin (Ctrl+D to finish, Ctrl+C to abort):")
				content, err := readFromSTDIN(f.IO)
				if err != nil {
					return err
				}
				opts.addFile(&opts.DisplayFilename, &content)
			} else {
				for _, path := range args {
					filename := path
					if len(args) == 1 && opts.DisplayFilename != "" {
						filename = opts.DisplayFilename
					}

					content, err := readFromFile(path)
					if err != nil {
						return err
					}
					dbg.Debug("Adding:", filename)
					opts.addFile(&filename, &content)
				}
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.HttpClient()
			if err != nil {
				return err
			}
			if opts.Personal {
				return runCreate(client, nil, opts)
			} else {
				repo, err := opts.BaseRepo()
				if err != nil {
					redCheck := opts.IO.Color().FailedIcon()
					return fmt.Errorf("%s Project snippet needs a repository. Do you want --personal?", redCheck)
				}
				return runCreate(client, repo, opts)

			}
		},
	}

	snippetCreateCmd.Flags().StringVarP(&opts.Title, "title", "t", "", "(required) Title of the snippet.")
	snippetCreateCmd.Flags().StringVarP(&opts.DisplayFilename, "filename", "f", "", "Filename of the snippet in GitLab.")
	snippetCreateCmd.Flags().StringVarP(&opts.Description, "description", "d", "", "Description of the snippet.")
	snippetCreateCmd.Flags().StringVarP(&opts.Visibility, "visibility", "v", "private", "Limit by visibility: 'public', 'internal', or 'private'")
	snippetCreateCmd.Flags().BoolVarP(&opts.Personal, "personal", "p", false, "Create a personal snippet.")

	return snippetCreateCmd
}

func runCreate(client *gitlab.Client, repo glrepo.Interface, opts *CreateOpts) error {
	var snippet *gitlab.Snippet
	var err error
	if opts.Personal {
		fmt.Fprintln(opts.IO.StdErr, "- Creating snippet in personal space")
		snippet, err = api.CreateSnippet(client, &gitlab.CreateSnippetOptions{
			Title:       &opts.Title,
			Description: &opts.Description,
			Visibility:  gitlab.Ptr(gitlab.VisibilityValue(opts.Visibility)),
			Files:       &opts.Files,
		})
	} else {
		fmt.Fprintln(opts.IO.StdErr, "- Creating snippet in", repo.FullName())
		snippet, err = api.CreateProjectSnippet(client, repo.FullName(), &gitlab.CreateProjectSnippetOptions{
			Title:       &opts.Title,
			Description: &opts.Description,
			Visibility:  gitlab.Ptr(gitlab.VisibilityValue(opts.Visibility)),
			Files:       &opts.Files,
		})
	}
	if err != nil {
		return fmt.Errorf("failed to create snippet: %w", err)
	}
	snippetID := opts.IO.Color().Green(fmt.Sprintf("$%d", snippet.ID))
	var files []string
	for _, file := range opts.Files {
		files = append(files, *file.FilePath)
	}
	names := strings.Join(files, " ")
	if opts.IO.IsaTTY {
		fmt.Fprintf(opts.IO.StdOut, "%s %s (%s)\n %s\n", snippetID, snippet.Title, names, snippet.WebURL)
	} else {
		fmt.Fprintln(opts.IO.StdOut, snippet.WebURL)
	}

	return nil
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
