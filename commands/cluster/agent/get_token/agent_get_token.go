package get_token

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientauthenticationv1 "k8s.io/client-go/pkg/apis/clientauthentication/v1"
)

var factory *cmdutils.Factory

const (
	clientAuthenticationApiV1          = "client.authentication.k8s.io/v1"
	clientAuthenticationExecCredential = "ExecCredential"
	bufferMinutes                      = 5
)

var patScopes = []string{"k8s_proxy"}

func NewCmdAgentGetToken(f *cmdutils.Factory) *cobra.Command {
	factory = f
	desc := "Create and return a k8s_proxy-scoped personal access token to authenticate with a GitLab Agents for Kubernetes."
	agentGetTokenCmd := &cobra.Command{
		Use:   "get-token [flags]",
		Short: desc,
		Long: fmt.Sprintf(`%s

This command creates a personal access token that is valid until the end of the current day.
You might receive an email from your GitLab instance that a new personal access token has been created.
`, desc),
		RunE: func(cmd *cobra.Command, args []string) error {
			factory = f
			agentID, err := cmd.Flags().GetInt("agent")
			if err != nil {
				return err
			}

			return runGetToken(agentID)
		},
	}
	agentGetTokenCmd.Flags().IntP("agent", "a", 0, "The numerical Agent ID to connect to.")
	cobra.CheckErr(agentGetTokenCmd.MarkFlagRequired("agent"))

	return agentGetTokenCmd
}

func runGetToken(agentID int) error {
	apiClient, err := factory.HttpClient()
	if err != nil {
		return err
	}

	randomBytes := make([]byte, 16)
	_, err = rand.Read(randomBytes)
	if err != nil {
		return err
	}

	patName := fmt.Sprintf("glab-k8s-proxy-%x", randomBytes)
	patExpiresAt := time.Now().Add(24 * time.Hour)
	pat, err := api.CreatePersonalAccessTokenForCurrentUser(apiClient, patName, patScopes, patExpiresAt)
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
			Token:               fmt.Sprintf("pat:%d:%s", agentID, pat.Token),
		},
	}

	e := json.NewEncoder(factory.IO.StdOut)
	e.SetIndent("", "  ")
	return e.Encode(execCredential)
}
