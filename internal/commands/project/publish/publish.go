package publish

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	publishCatalogCmd "gitlab.com/gitlab-org/cli/internal/commands/project/publish/catalog"
)

func NewCmdPublish(f cmdutils.Factory) *cobra.Command {
	publishCmd := &cobra.Command{
		Use:   "publish <command> [flags]",
		Short: `Publishes resources in the project.`,
		Long: heredoc.Doc(`Publishes resources in the project.
    
    Currently only supports publishing CI/CD components to the catalog.
    `),
	}

	publishCmd.AddCommand(publishCatalogCmd.NewCmdPublishCatalog(f))

	return publishCmd
}
