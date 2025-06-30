package version

import (
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"

	"github.com/spf13/cobra"
)

func NewCmdVersion(f cmdutils.Factory) *cobra.Command {
	versionCmd := &cobra.Command{
		Use:     "version",
		Short:   "Show version information for glab.",
		Long:    ``,
		Aliases: []string{"v"},
		RunE: func(cmd *cobra.Command, args []string) error {
			buildInfo := f.BuildInfo()
			fmt.Fprint(f.IO().StdOut, Scheme(buildInfo.Version, buildInfo.Commit))
			return nil
		},
	}
	return versionCmd
}

func Scheme(version, commit string) string {
	return fmt.Sprintf("glab %s (%s)\n", strings.TrimPrefix(version, "v"), commit)
}
