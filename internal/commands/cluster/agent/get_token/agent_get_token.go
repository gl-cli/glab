package get_token

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientauthenticationv1 "k8s.io/client-go/pkg/apis/clientauthentication/v1"
)

const (
	clientAuthenticationApiV1          = "client.authentication.k8s.io/v1"
	clientAuthenticationExecCredential = "ExecCredential"
	bufferMinutes                      = 5
)

var patScopes = []string{"k8s_proxy"}

type options struct {
	httpClient func() (*gitlab.Client, error)
	io         *iostreams.IOStreams

	agentID int64
}

func NewCmdAgentGetToken(f cmdutils.Factory) *cobra.Command {
	opts := options{
		io:         f.IO(),
		httpClient: f.HttpClient,
	}
	desc := "Create and return a k8s_proxy-scoped personal access token to authenticate with a GitLab Agents for Kubernetes."
	agentGetTokenCmd := &cobra.Command{
		Use:   "get-token [flags]",
		Short: desc,
		Long: fmt.Sprintf(`%s

This command creates a personal access token that is valid until the end of the current day.
You might receive an email from your GitLab instance that a new personal access token has been created.
`, desc),
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.run()
		},
	}
	agentGetTokenCmd.Flags().Int64VarP(&opts.agentID, "agent", "a", 0, "The numerical Agent ID to connect to.")
	cobra.CheckErr(agentGetTokenCmd.MarkFlagRequired("agent"))

	return agentGetTokenCmd
}

func (o *options) run() error {
	apiClient, err := o.httpClient()
	if err != nil {
		return err
	}

	randomBytes := make([]byte, 16)
	_, err = rand.Read(randomBytes)
	if err != nil {
		return err
	}

	patName := fmt.Sprintf("glab-k8s-proxy-%x", randomBytes)
	patExpiresAt := time.Now().Add(24 * time.Hour).UTC()

	pat, _, err := apiClient.Users.CreatePersonalAccessTokenForCurrentUser(&gitlab.CreatePersonalAccessTokenForCurrentUserOptions{
		Name:      gitlab.Ptr(patName),
		Scopes:    gitlab.Ptr(patScopes),
		ExpiresAt: gitlab.Ptr(gitlab.ISOTime(patExpiresAt)),
	})
	if err != nil {
		return err
	}

	execCredential := clientauthenticationv1.ExecCredential{
		TypeMeta: metav1.TypeMeta{
			APIVersion: clientAuthenticationApiV1,
			Kind:       clientAuthenticationExecCredential,
		},
		Status: &clientauthenticationv1.ExecCredentialStatus{
			ExpirationTimestamp: &metav1.Time{Time: time.Time(*pat.ExpiresAt).Add(-bufferMinutes * time.Minute)},
			Token:               fmt.Sprintf("pat:%d:%s", o.agentID, pat.Token),
		},
	}

	e := json.NewEncoder(o.io.StdOut)
	e.SetIndent("", "  ")
	return e.Encode(execCredential)
}
