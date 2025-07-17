package get_token

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/cluster/agent/agentutils"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientauthenticationv1 "k8s.io/client-go/pkg/apis/clientauthentication/v1"
)

const (
	clientAuthenticationApiV1          = "client.authentication.k8s.io/v1"
	clientAuthenticationExecCredential = "ExecCredential"
	buffer                             = 5 * time.Minute
	flagTokenExpiryDuration            = "token-expiry-duration"
	tokenExpiryDurationDefault         = 24 * time.Hour
	minTokenExpiryDuration             = 24 * time.Hour
)

var patScopes = []string{"k8s_proxy"}

type options struct {
	httpClient func() (*gitlab.Client, error)
	io         *iostreams.IOStreams

	agentID             int64
	tokenExpiryDuration time.Duration
	cacheMode           agentutils.CacheMode
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
			if err := opts.validate(); err != nil {
				return err
			}
			return opts.run()
		},
	}
	fl := agentGetTokenCmd.Flags()
	fl.Int64VarP(&opts.agentID, "agent", "a", 0, "The numerical Agent ID to connect to.")
	fl.DurationVar(&opts.tokenExpiryDuration, flagTokenExpiryDuration, tokenExpiryDurationDefault, "Duration for how long the generated tokens should be valid for. Minimum is 1 day and the effective expiry is always at the end of the day, the time is ignored.")
	agentutils.AddTokenCacheModeFlag(fl, &opts.cacheMode)
	cobra.CheckErr(agentGetTokenCmd.MarkFlagRequired("agent"))

	return agentGetTokenCmd
}

func (o *options) validate() error {
	if o.tokenExpiryDuration < minTokenExpiryDuration {
		return fmt.Errorf("token expiry duration must be at least 24 hours")
	}
	return nil
}

func (o *options) run() error {
	pat, err := o.cachedPAT()
	if err != nil {
		return err
	}

	execCredential := clientauthenticationv1.ExecCredential{
		TypeMeta: metav1.TypeMeta{
			APIVersion: clientAuthenticationApiV1,
			Kind:       clientAuthenticationExecCredential,
		},
		Status: &clientauthenticationv1.ExecCredentialStatus{
			ExpirationTimestamp: &metav1.Time{Time: time.Time(*pat.ExpiresAt).Add(-buffer)},
			Token:               fmt.Sprintf("pat:%d:%s", o.agentID, pat.Token),
		},
	}

	e := json.NewEncoder(o.io.StdOut)
	e.SetIndent("", "  ")
	return e.Encode(execCredential)
}

func (o *options) cachedPAT() (*gitlab.PersonalAccessToken, error) {
	apiClient, err := o.httpClient()
	if err != nil {
		return nil, err
	}

	createFunc := func() (*gitlab.PersonalAccessToken, error) {
		randomBytes := make([]byte, 16)

		_, err = rand.Read(randomBytes)
		if err != nil {
			return nil, err
		}

		patName := fmt.Sprintf("glab-k8s-proxy-%x", randomBytes)
		patExpiresAt := time.Now().Add(o.tokenExpiryDuration).UTC()

		pat, _, err := apiClient.Users.CreatePersonalAccessTokenForCurrentUser(&gitlab.CreatePersonalAccessTokenForCurrentUserOptions{
			Name:      gitlab.Ptr(patName),
			Scopes:    gitlab.Ptr(patScopes),
			ExpiresAt: gitlab.Ptr(gitlab.ISOTime(patExpiresAt)),
		})
		if err != nil {
			return nil, err
		}
		return pat, nil
	}

	gitlabInstance := base64.StdEncoding.EncodeToString([]byte(apiClient.BaseURL().String()))
	id := fmt.Sprintf("%s-%d", gitlabInstance, o.agentID)

	switch o.cacheMode {
	case agentutils.NoCacheCacheMode:
		return createFunc()
	case agentutils.ForcedKeyringCacheMode:
		return fromKeyringCache(id, createFunc)
	case agentutils.ForcedFilesystemCacheMode:
		return fromFilesystemCache(id, createFunc)
	case agentutils.KeyringFilesystemFallback:
		pat, err := fromKeyringCache(id, createFunc)
		if err != nil {
			if !errors.Is(err, keyring.ErrUnsupportedPlatform) {
				return nil, err
			}
			return fromFilesystemCache(id, createFunc)
		}
		return pat, nil
	default:
		panic(fmt.Sprintf("unimplemented cache mode: %s. This is a programming error, please report at https://gitlab.com/gitlab-org/cli/-/issues", o.cacheMode))
	}
}

func fromKeyringCache(id string, createFunc func() (*gitlab.PersonalAccessToken, error)) (*gitlab.PersonalAccessToken, error) {
	c := cache{
		storage:    &keyringStorage{},
		id:         id,
		createFunc: createFunc,
	}

	return c.get()
}

func fromFilesystemCache(id string, createFunc func() (*gitlab.PersonalAccessToken, error)) (*gitlab.PersonalAccessToken, error) {
	fs, err := newFileStorage()
	if err != nil {
		return nil, err
	}
	defer fs.close()

	c := cache{
		id:         id,
		createFunc: createFunc,
		storage:    fs,
	}
	return c.get()
}
