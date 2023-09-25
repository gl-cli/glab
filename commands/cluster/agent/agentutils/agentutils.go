package agentutils

import (
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/pkg/tableprinter"
	"gitlab.com/gitlab-org/cli/pkg/utils"
)

func DisplayAllAgents(io *iostreams.IOStreams, agents []*gitlab.Agent) string {
	c := io.Color()
	table := tableprinter.NewTablePrinter()
	table.AddRow(c.Bold("ID"), c.Bold("Name"), c.Bold(c.Gray("Created At")))
	for _, r := range agents {
		table.AddRow(r.ID, r.Name, c.Gray(utils.TimeToPrettyTimeAgo(*r.CreatedAt)))
	}
	return table.Render()
}
