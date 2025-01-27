package list

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
)

func Test_listRun_group(t *testing.T) {
	reg := &httpmock.Mocker{}
	defer reg.Verify(t)

	varContent := `KEY	PROTECTED	MASKED	EXPANDED	SCOPE
TEST_VAR	false	false	true	*
`

	body := []struct {
		Key              string `json:"key"`
		VariableType     string `json:"variable_type"`
		Value            string `json:"value"`
		Protected        bool   `json:"protected"`
		Masked           bool   `json:"masked"`
		EnvironmentScope string `json:"environment_scope"`
	}{
		{
			Key:              "TEST_VAR",
			VariableType:     "env_var",
			Value:            varContent,
			Protected:        false,
			Masked:           false,
			EnvironmentScope: "*",
		},
	}

	reg.RegisterResponder(http.MethodGet, "/groups/example/variables",
		httpmock.NewJSONResponse(http.StatusOK, body),
	)

	io, _, stdout, _ := iostreams.Test()

	opts := &ListOpts{
		HTTPClient: func() (*gitlab.Client, error) {
			a, _ := api.TestClient(&http.Client{Transport: reg}, "", "gitlab.com", false)
			return a.Lab(), nil
		},
		BaseRepo: func() (glrepo.Interface, error) {
			return glrepo.FromFullName("example/project")
		},
		IO:    io,
		Group: "example",
	}
	_, _ = opts.HTTPClient()

	err := listRun(opts)
	assert.NoError(t, err)
	assert.Equal(t, varContent, stdout.String())
}
