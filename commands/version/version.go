package version

import (
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/spf13/cobra"
)

func NewCmdVersion(s *iostreams.IOStreams, version, buildDate string) *cobra.Command {
	versionCmd := &cobra.Command{
		Use:     "version",
		Short:   "Show version information for glab.",
		Long:    ``,
		Aliases: []string{"v"},
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprint(s.StdOut, Scheme(version, buildDate))
			return nil
		},
	}
	return versionCmd
}

func Scheme(version, buildDate string) string {
	version = strings.TrimPrefix(version, "v")

	if buildDate != "" {
		version = fmt.Sprintf("%s (%s)", version, buildDate)
	}

	return fmt.Sprintf("Current glab version: %s\n", version)
}
