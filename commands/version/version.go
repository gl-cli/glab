package version

import (
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/spf13/cobra"
)

func NewCmdVersion(s *iostreams.IOStreams, version, commit string) *cobra.Command {
	versionCmd := &cobra.Command{
		Use:     "version",
		Short:   "Show version information for glab.",
		Long:    ``,
		Aliases: []string{"v"},
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprint(s.StdOut, Scheme(version, commit))
			return nil
		},
	}
	return versionCmd
}

func Scheme(version, commit string) string {
	return fmt.Sprintf("glab %s (%s)\n", strings.TrimPrefix(version, "v"), commit)
}
