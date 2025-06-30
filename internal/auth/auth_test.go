package auth

import (
	"errors"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	gitlab_testing "gitlab.com/gitlab-org/api/client-go/testing"

	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/config"
)

func Test_GetAuthenticatedClient(t *testing.T) {
	tests := []struct {
		name       string
		HttpClient func() (*gitlab.Client, error)
		Config     func() config.Config
		wantErr    bool
	}{
		{
			name: "everything ok!",
		},
		{
			name: "no hosts",
			Config: func() config.Config {
				return config.NewFromString(heredoc.Doc(`
				hosts:
			`))
			},
			wantErr: true,
		},
		{
			name: "bad httpclient",
			HttpClient: func() (*gitlab.Client, error) {
				return nil, errors.New("oopsies")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, _ := cmdtest.TestIOStreams()
			tc := gitlab_testing.NewTestClient(t)
			f := cmdtest.InitFactory(ios, nil)
			f.HttpClientStub = func() (*gitlab.Client, error) {
				return tc.Client, nil
			}

			if tt.Config != nil {
				f.ConfigStub = tt.Config
			}

			if tt.HttpClient != nil {
				f.HttpClientStub = tt.HttpClient
			}

			_, err := GetAuthenticatedClient(f)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
