package cluster

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/test"
)

func TestNewCmdCluster(t *testing.T) {
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	assert.Nil(t, NewCmdCluster(&cmdutils.Factory{
		IO: &iostreams.IOStreams{
			StdOut: os.Stdout,
		},
		HttpClient: func() (*gitlab.Client, error) { return nil, nil },
		BaseRepo:   func() (glrepo.Interface, error) { return glrepo.New("OWNER", "REPO"), nil },
	}).Execute())

	out := test.ReturnBuffer(old, r, w)

	assert.Contains(t, out, "Manage GitLab Agents for Kubernetes and their clusters")
}
