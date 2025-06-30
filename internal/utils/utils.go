package utils

import (
	"fmt"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"gitlab.com/gitlab-org/cli/internal/browser"
	"gitlab.com/gitlab-org/cli/internal/run"
)

type MarkdownRenderOpts []glamour.TermRendererOption

// OpenInBrowser opens the url in a web browser based on OS and $BROWSER environment variable
func OpenInBrowser(url, browserType string) error {
	browseCmd, err := browser.Command(url, browserType)
	if err != nil {
		return err
	}
	return run.PrepareCmd(browseCmd).Run()
}

func SanitizePathName(path string) string {
	if !strings.HasPrefix(path, "/") {
		// Prefix the path with "/" ensures that filepath.Clean removes all `/..`
		// See rule 4 of filepath.Clean for more information: https://pkg.go.dev/path/filepath#Clean
		path = "/" + path
	}
	return filepath.Clean(path)
}

func RenderMarkdown(text, glamourStyle string) (string, error) {
	opts := MarkdownRenderOpts{
		glamour.WithStylePath(getStyle(glamourStyle)),
	}

	return renderMarkdown(text, opts)
}

func RenderMarkdownWithoutIndentations(text, glamourStyle string) (string, error) {
	opts := MarkdownRenderOpts{
		glamour.WithStylePath(getStyle(glamourStyle)),
		markdownWithoutIndentation(),
	}

	return renderMarkdown(text, opts)
}

func renderMarkdown(text string, opts MarkdownRenderOpts) (string, error) {
	// Glamour rendering preserves carriage return characters in code blocks, but
	// we need to ensure that no such characters are present in the output.
	text = strings.ReplaceAll(text, "\r\n", "\n")

	tr, err := glamour.NewTermRenderer(opts...)
	if err != nil {
		return "", err
	}

	return tr.Render(text)
}

func markdownWithoutIndentation() glamour.TermRendererOption {
	overrides := []byte(`
	  {
			"document": {
				"margin": 0
			},
			"code_block": {
				"margin": 0
			}
	  }`)

	return glamour.WithStylesFromJSONBytes(overrides)
}

func getStyle(glamourStyle string) string {
	if glamourStyle == "" || glamourStyle == "none" {
		return "notty"
	}
	return glamourStyle
}

func Pluralize(num int, thing string) string {
	if num == 1 {
		return fmt.Sprintf("%d %s", num, thing)
	}
	return fmt.Sprintf("%d %ss", num, thing)
}

func fmtDuration(amount int, unit string) string {
	return fmt.Sprintf("about %s ago", Pluralize(amount, unit))
}

func PrettyTimeAgo(ago time.Duration) string {
	if ago < time.Minute {
		return "less than a minute ago"
	}
	if ago < time.Hour {
		return fmtDuration(int(ago.Minutes()), "minute")
	}
	if ago < 24*time.Hour {
		return fmtDuration(int(ago.Hours()), "hour")
	}
	if ago < 30*24*time.Hour {
		return fmtDuration(int(ago.Hours())/24, "day")
	}
	if ago < 365*24*time.Hour {
		return fmtDuration(int(ago.Hours())/24/30, "month")
	}

	return fmtDuration(int(ago.Hours()/24/365), "year")
}

func TimeToPrettyTimeAgo(d time.Time) string {
	now := time.Now()
	ago := now.Sub(d)
	return PrettyTimeAgo(ago)
}

func FmtDuration(d time.Duration) string {
	d = d.Round(time.Second)
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02dm %02ds", m, s)
}

func Humanize(s string) string {
	// Replaces - and _ with spaces.
	replace := "_-"
	h := func(r rune) rune {
		if strings.ContainsRune(replace, r) {
			return ' '
		}
		return r
	}

	return strings.Map(h, s)
}

func DisplayURL(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}
	return u.Hostname() + u.Path
}

// PresentInStringSlice take a Hay (Slice of Strings) and a Needle (string)
// and returns true based on whether or not the Needle is present in the hay.
func PresentInStringSlice(hay []string, needle string) bool {
	return slices.Contains(hay, needle)
}

// PresentInIntSlice take a Hay (Slice of Ints) and a Needle (int)
// and returns true based on whether or not the Needle is present in the hay.
func PresentInIntSlice(hay []int, needle int) bool {
	return slices.Contains(hay, needle)
}

// CommonElementsInStringSlice takes 2 Slices of Strings and returns a Third Slice
// that is the common elements between the first 2 Slices.
func CommonElementsInStringSlice(s1 []string, s2 []string) []string {
	hash := make(map[string]bool)
	for x := range s1 {
		hash[s1[x]] = true
	}
	var arr []string
	for i := range s2 {
		if hash[s2[i]] {
			arr = append(arr, s2[i])
		}
	}
	return arr
}

// isValidUrl tests a string to determine if it is a well-structured url or not.
func IsValidURL(toTest string) bool {
	_, err := url.ParseRequestURI(toTest)
	if err != nil {
		return false
	}

	u, err := url.Parse(toTest)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}

	return true
}

// Map transfers the elements of its first argument using the result of the second fn(e)
func Map[T1, T2 any](elems []T1, fn func(T1) T2) []T2 {
	r := make([]T2, len(elems))

	for i, v := range elems {
		r[i] = fn(v)
	}

	return r
}
