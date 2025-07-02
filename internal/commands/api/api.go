package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gitlab.com/gitlab-org/cli/internal/iostreams"

	"gitlab.com/gitlab-org/cli/internal/api"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	jsonPretty "github.com/tidwall/pretty"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
)

type options struct {
	io *iostreams.IOStreams

	apiClient func(repoHost string, cfg config.Config) (*api.Client, error)
	baseRepo  func() (glrepo.Interface, error)
	branch    func() (string, error)
	config    config.Config

	hostname            string
	requestMethod       string
	requestMethodPassed bool
	requestPath         string
	requestInputFile    string
	magicFields         []string
	rawFields           []string
	requestHeaders      []string
	showResponseHeaders bool
	paginate            bool
	silent              bool
}

func NewCmdApi(f cmdutils.Factory, runF func(*options) error) *cobra.Command {
	opts := options{
		io:        f.IO(),
		apiClient: f.ApiClient,
		baseRepo:  f.BaseRepo,
		branch:    f.Branch,
	}

	cmd := &cobra.Command{
		Use:   "api <endpoint>",
		Short: "Make an authenticated request to the GitLab API.",
		Long: heredoc.Docf(`
		Makes an authenticated HTTP request to the GitLab API, and prints the response.
		The endpoint argument should either be a path of a GitLab API v4 endpoint, or
		"graphql" to access the GitLab GraphQL API.

		- [GitLab REST API documentation](https://docs.gitlab.com/api/)
		- [GitLab GraphQL documentation](https://docs.gitlab.com/api/graphql/)

		If the current directory is a Git directory, uses the GitLab authenticated host in the current
		directory. Otherwise, %[1]sgitlab.com%[1]s will be used.
		To override the GitLab hostname, use '--hostname'.

		These placeholder values, when used in the endpoint argument, are
		replaced with values from the repository of the current directory:

		- %[1]s:branch%[1]s
		- %[1]s:fullpath%[1]s
		- %[1]s:group%[1]s
		- %[1]s:id%[1]s
		- %[1]s:namespace%[1]s
		- %[1]s:repo%[1]s
		- %[1]s:user%[1]s
		- %[1]s:username%[1]s

		Methods: the default HTTP request method is "GET", if no parameters are added, and "POST" otherwise. Override the method with '--method'.

		Pass one or more '--raw-field' values in "key=value" format to add
		JSON-encoded string parameters to the POST body.

		The '--field' flag behaves like '--raw-field' with magic type conversion based
		on the format of the value:

		- Literal values "true", "false", "null", and integer numbers are converted to
		  appropriate JSON types.
		- Placeholder values ":namespace", ":repo", and ":branch" are populated with values
		  from the repository of the current directory.
		- If the value starts with "@", the rest of the value is interpreted as a
		  filename to read the value from. Pass "-" to read from standard input.

		For GraphQL requests, all fields other than "query" and "operationName" are
		interpreted as GraphQL variables.

		Raw request body can be passed from the outside via a file specified by '--input'.
		Pass "-" to read from standard input. In this mode, parameters specified with
		'--field' flags are serialized into URL query parameters.

		In '--paginate' mode, all pages of results are requested sequentially until
		no more pages of results remain. For GraphQL requests:

		- The original query must accept an '$endCursor: String' variable.
		- The query must fetch the 'pageInfo{ hasNextPage, endCursor }' set of fields from a collection.
		`, "`"),
		Example: heredoc.Doc(`
			- glab api projects/:fullpath/releases

			- glab api projects/gitlab-com%2Fwww-gitlab-com/issues

			- glab api issues --paginate

			$ glab api graphql -f query="query { currentUser { username } }"

			$ glab api graphql -f query='
			  query {
			    project(fullPath: "gitlab-org/gitlab-docs") {
			      name
			      forksCount
			      statistics {
			        wikiSize
			      }
			      issuesEnabled
			      boards {
			        nodes {
			          id
			          name
			        }
			      }
			    }
			  }
			'

			$ glab api graphql --paginate -f query='
			  query($endCursor: String) {
			    project(fullPath: "gitlab-org/graphql-sandbox") {
			      name
			      issues(first: 2, after: $endCursor) {
			        edges {
			          node {
			            title
			          }
			        }
			        pageInfo {
			          endCursor
			          hasNextPage
			        }
			      }
			    }
			  }'
		`),
		Annotations: map[string]string{
			"help:environment": heredoc.Doc(`
				GITLAB_TOKEN, OAUTH_TOKEN (in order of precedence): an authentication token for API requests.
				GITLAB_HOST, GITLAB_URI, GITLAB_URL: specify a GitLab host to make request to.
			`),
		},
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.config = f.Config()

			opts.complete(cmd, args)

			if err := opts.validate(cmd); err != nil {
				return err
			}

			if runF != nil {
				return runF(&opts)
			}
			return opts.run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&opts.hostname, "hostname", "", "The GitLab hostname for the request. Defaults to \"gitlab.com\", or the authenticated host in the current Git directory.")
	cmd.Flags().StringVarP(&opts.requestMethod, "method", "X", "GET", "The HTTP method for the request.")
	cmd.Flags().StringArrayVarP(&opts.magicFields, "field", "F", nil, "Add a parameter of inferred type. Changes the default HTTP method to \"POST\".")
	cmd.Flags().StringArrayVarP(&opts.rawFields, "raw-field", "f", nil, "Add a string parameter.")
	cmd.Flags().StringArrayVarP(&opts.requestHeaders, "header", "H", nil, "Add an additional HTTP request header.")
	cmd.Flags().BoolVarP(&opts.showResponseHeaders, "include", "i", false, "Include HTTP response headers in the output.")
	cmd.Flags().BoolVar(&opts.paginate, "paginate", false, "Make additional HTTP requests to fetch all pages of results.")
	cmd.Flags().StringVar(&opts.requestInputFile, "input", "", "The file to use as the body for the HTTP request.")
	cmd.Flags().BoolVar(&opts.silent, "silent", false, "Do not print the response body.")
	cmd.MarkFlagsMutuallyExclusive("paginate", "input")
	return cmd
}

func (o *options) complete(cmd *cobra.Command, args []string) {
	o.requestPath = args[0]
	o.requestMethodPassed = cmd.Flags().Changed("method")
}

func (o *options) validate(cmd *cobra.Command) error {
	if cmd.Flags().Changed("hostname") {
		if err := glinstance.HostnameValidator(o.hostname); err != nil {
			return &cmdutils.FlagError{Err: fmt.Errorf("error parsing --hostname: %w.", err)}
		}
	}

	if o.paginate && !strings.EqualFold(o.requestMethod, http.MethodGet) && o.requestPath != "graphql" {
		return &cmdutils.FlagError{Err: errors.New(`the '--paginate' option is not supported for non-GET requests.`)}
	}

	return nil
}

func (o *options) run(ctx context.Context) error {
	params, err := parseFields(o)
	if err != nil {
		return err
	}
	isGraphQL := o.requestPath == "graphql"
	requestPath, err := fillPlaceholders(o.requestPath, o)
	if err != nil {
		return fmt.Errorf("unable to expand placeholder in path: %w", err)
	}
	method := o.requestMethod
	requestHeaders := o.requestHeaders
	var requestBody any = params

	if !o.requestMethodPassed && (len(params) > 0 || o.requestInputFile != "") {
		method = http.MethodPost
	}

	if o.paginate && !isGraphQL {
		requestPath = addPerPage(requestPath, 100, params)
	}

	if o.requestInputFile != "" {
		file, size, err := openUserFile(o.requestInputFile, o.io.In)
		if err != nil {
			return err
		}
		defer file.Close()
		requestPath, err = parseQuery(requestPath, params)
		if err != nil {
			return err
		}
		requestBody = file
		if size >= 0 {
			requestHeaders = append([]string{fmt.Sprintf("Content-Length: %d", size)}, requestHeaders...)
		}
	}

	headersOutputStream := o.io.StdOut
	if o.silent {
		o.io.StdOut = io.Discard
	} else {
		err := o.io.StartPager()
		if err != nil {
			return err
		}
		defer o.io.StopPager()
	}

	// NOTE: check if repository is available
	br, err := o.baseRepo()
	var repoHost string
	if err == nil {
		repoHost = br.RepoHost()
	}
	client, err := o.apiClient(repoHost, o.config)
	if err != nil {
		return err
	}

	hasNextPage := true
	for hasNextPage {
		resp, err := httpRequest(ctx, client, method, requestPath, requestBody, requestHeaders)
		if err != nil {
			return err
		}

		endCursor, err := processResponse(resp, o, headersOutputStream)
		if err != nil {
			return err
		}

		if !o.paginate {
			break
		}

		if isGraphQL {
			hasNextPage = endCursor != ""
			if hasNextPage {
				params["endCursor"] = endCursor
			}
		} else {
			requestPath, hasNextPage = findNextPage(resp)
		}

		if hasNextPage && o.showResponseHeaders {
			fmt.Fprint(o.io.StdOut, "\n")
		}
	}

	return nil
}

func processResponse(resp *http.Response, opts *options, headersOutputStream io.Writer) (string, error) {
	if opts.showResponseHeaders {
		fmt.Fprintln(headersOutputStream, resp.Proto, resp.Status)
		printHeaders(headersOutputStream, resp.Header, opts.io.ColorEnabled())
		fmt.Fprint(headersOutputStream, "\r\n")
	}

	if resp.StatusCode == http.StatusNoContent {
		return "", nil
	}
	var responseBody io.Reader = resp.Body

	isJSON, _ := regexp.MatchString(`[/+]json(;|$)`, resp.Header.Get("Content-Type"))

	var serverError string
	if isJSON && (opts.requestPath == "graphql" || resp.StatusCode >= http.StatusBadRequest) {
		var err error
		responseBody, serverError, err = parseErrorResponse(responseBody, resp.StatusCode)
		if err != nil {
			return "", err
		}
	}

	var bodyCopy *bytes.Buffer
	isGraphQLPaginate := isJSON && resp.StatusCode == http.StatusOK && opts.paginate && opts.requestPath == "graphql"
	if isGraphQLPaginate {
		bodyCopy = &bytes.Buffer{}
		responseBody = io.TeeReader(responseBody, bodyCopy)
	}

	var err error
	if isJSON && opts.io.ColorEnabled() {
		out := &bytes.Buffer{}
		_, err = io.Copy(out, responseBody)
		if err == nil {
			result := jsonPretty.Color(jsonPretty.Pretty(out.Bytes()), nil)
			_, err = fmt.Fprintln(opts.io.StdOut, string(result))
		}
	} else {
		_, err = io.Copy(opts.io.StdOut, responseBody)
	}
	if err != nil {
		return "", err
	}

	if serverError != "" {
		fmt.Fprintf(opts.io.StdErr, "glab: %s\n", serverError)
		return "", cmdutils.SilentError
	} else if resp.StatusCode > 299 {
		fmt.Fprintf(opts.io.StdErr, "glab: HTTP %d\n", resp.StatusCode)
		return "", cmdutils.SilentError
	}

	if isGraphQLPaginate {
		return findEndCursor(bodyCopy), nil
	}

	return "", nil
}

var placeholderRE = regexp.MustCompile(`:(group/:namespace/:repo|namespace/:repo|fullpath|id|user|username|group|namespace|repo|branch)\b`)

// fillPlaceholders populates `:namespace` and `:repo` placeholders with values from the current repository
func fillPlaceholders(value string, opts *options) (string, error) {
	if !placeholderRE.MatchString(value) {
		return value, nil
	}

	var err error
	filled := placeholderRE.ReplaceAllStringFunc(value, func(m string) string {
		switch m {
		case ":id":
			baseRepo, baseRepoErr := opts.baseRepo()
			if baseRepoErr != nil {
				err = baseRepoErr
				return ""
			}

			h, _ := opts.apiClient(baseRepo.RepoHost(), opts.config)
			project, e := baseRepo.Project(h.Lab())
			if e == nil && project != nil {
				return strconv.Itoa(project.ID)
			}
			err = e
			return ""
		case ":group/:namespace/:repo", ":fullpath":
			baseRepo, baseRepoErr := opts.baseRepo()
			if baseRepoErr != nil {
				err = baseRepoErr
				return ""
			}
			return url.PathEscape(baseRepo.FullName())
		case ":namespace/:repo":
			baseRepo, baseRepoErr := opts.baseRepo()
			if baseRepoErr != nil {
				err = baseRepoErr
				return ""
			}

			return url.PathEscape(baseRepo.RepoNamespace() + "/" + baseRepo.RepoName())
		case ":group":
			baseRepo, baseRepoErr := opts.baseRepo()
			if baseRepoErr != nil {
				err = baseRepoErr
				return ""
			}

			return baseRepo.RepoGroup()
		case ":user", ":username":
			h, _ := opts.apiClient("", opts.config)
			u, _, e := h.Lab().Users.CurrentUser()
			if e == nil && u != nil {
				return u.Username
			}
			err = e
			return m
		case ":namespace":
			baseRepo, baseRepoErr := opts.baseRepo()
			if baseRepoErr != nil {
				err = baseRepoErr
				return ""
			}

			return baseRepo.RepoNamespace()
		case ":repo":
			baseRepo, baseRepoErr := opts.baseRepo()
			if baseRepoErr != nil {
				err = baseRepoErr
				return ""
			}

			return baseRepo.RepoName()
		case ":branch":
			branch, e := opts.branch()
			if e != nil {
				err = e
			}
			return branch
		default:
			err = fmt.Errorf("invalid placeholder: %q", m)
			return ""
		}
	})

	if err != nil {
		return value, err
	}

	return filled, nil
}

func printHeaders(w io.Writer, headers http.Header, colorize bool) {
	var names []string
	for name := range headers {
		if name == "Status" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	var headerColor, headerColorReset string
	if colorize {
		headerColor = "\x1b[1;34m" // bright blue
		headerColorReset = "\x1b[m"
	}
	for _, name := range names {
		fmt.Fprintf(w, "%s%s%s: %s\r\n", headerColor, name, headerColorReset, strings.Join(headers[name], ", "))
	}
}

func parseFields(opts *options) (map[string]any, error) {
	params := make(map[string]any)
	for _, f := range opts.rawFields {
		key, value, err := parseField(f)
		if err != nil {
			return params, err
		}
		params[key] = value
	}
	for _, f := range opts.magicFields {
		key, strValue, err := parseField(f)
		if err != nil {
			return params, err
		}
		value, err := magicFieldValue(strValue, opts)
		if err != nil {
			return params, fmt.Errorf("error parsing %q value: %w", key, err)
		}
		params[key] = value
	}
	return params, nil
}

func parseField(f string) (string, string, error) {
	idx := strings.IndexRune(f, '=')
	if idx == -1 {
		return f, "", fmt.Errorf("field %q requires a value separated by an '=' sign.", f)
	}
	return f[0:idx], f[idx+1:], nil
}

func magicFieldValue(v string, opts *options) (any, error) {
	if strings.HasPrefix(v, "@") {
		return readUserFile(v[1:], opts.io.In)
	}

	if n, err := strconv.Atoi(v); err == nil {
		return n, nil
	}

	switch v {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "null":
		return nil, nil
	default:
		return fillPlaceholders(v, opts)
	}
}

func readUserFile(fn string, stdin io.ReadCloser) ([]byte, error) {
	var r io.ReadCloser
	if fn == "-" {
		r = stdin
	} else {
		var err error
		r, err = os.Open(fn)
		if err != nil {
			return nil, err
		}
	}
	defer r.Close()
	return io.ReadAll(r)
}

func openUserFile(fn string, stdin io.ReadCloser) (io.ReadCloser, int64, error) {
	if fn == "-" {
		return stdin, -1, nil
	}

	r, err := os.Open(fn)
	if err != nil {
		return r, -1, err
	}

	s, err := os.Stat(fn)
	if err != nil {
		return r, -1, err
	}

	return r, s.Size(), nil
}

func parseErrorResponse(r io.Reader, statusCode int) (io.Reader, string, error) {
	bodyCopy := &bytes.Buffer{}
	b, err := io.ReadAll(io.TeeReader(r, bodyCopy))
	if err != nil {
		return r, "", err
	}

	var parsedBody struct {
		Message string
		Errors  []json.RawMessage
	}
	err = json.Unmarshal(b, &parsedBody)
	if err != nil {
		// in cases where it's an object within an object we can try to parse it as is
		var t any
		err = json.Unmarshal(b, &t)
		if err != nil {
			return r, "", err
		}
		return bodyCopy, fmt.Sprintf("%v+", t), nil
	}
	if parsedBody.Message != "" {
		return bodyCopy, fmt.Sprintf("%s (HTTP %d)", parsedBody.Message, statusCode), nil
	}

	type errorMessage struct {
		Message string
	}
	var respErrors []string
	for _, rawErr := range parsedBody.Errors {
		if len(rawErr) == 0 {
			continue
		}
		if rawErr[0] == '{' {
			var objectError errorMessage
			err := json.Unmarshal(rawErr, &objectError)
			if err != nil {
				return r, "", err
			}
			respErrors = append(respErrors, objectError.Message)
		} else if rawErr[0] == '"' {
			var stringError string
			err := json.Unmarshal(rawErr, &stringError)
			if err != nil {
				return r, "", err
			}
			respErrors = append(respErrors, stringError)
		}
	}

	if len(respErrors) > 0 {
		return bodyCopy, strings.Join(respErrors, "\n"), nil
	}

	return bodyCopy, "", nil
}
