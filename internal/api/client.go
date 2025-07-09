package api

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/oauth2"
)

// ClientOption represents a function that configures a Client
type ClientOption func(*Client) error

type BuildInfo struct {
	Version, Commit, Platform, Architecture string
}

func (i BuildInfo) UserAgent() string {
	return fmt.Sprintf("glab/%s (%s, %s)", i.Version, i.Platform, i.Architecture)
}

// Client represents an argument to NewClient
type Client struct {
	// gitlabClient represents GitLab API client.
	gitlabClient *gitlab.Client
	// internal http client
	httpClient *http.Client
	// custom certificate
	caFile string
	// client certificate files
	clientCertFile string
	clientKeyFile  string

	baseURL    string
	authSource gitlab.AuthSource

	allowInsecure bool

	userAgent string
}

func (c *Client) HTTPClient() *http.Client {
	return c.httpClient
}

// AuthSource returns the auth source
// TODO: clarify use cases for this.
func (c *Client) AuthSource() gitlab.AuthSource {
	return c.authSource
}

// Lab returns the initialized GitLab client.
func (c *Client) Lab() *gitlab.Client {
	return c.gitlabClient
}

var secureCipherSuites = []uint16{
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
}

type newAuthSource func(c *http.Client) (authSource gitlab.AuthSource, err error)

// NewClient initializes a api client for use throughout glab.
func NewClient(newAuthSource newAuthSource, options ...ClientOption) (*Client, error) {
	// 0. initialize empty Client
	client := &Client{}

	// 1. apply provided option functions to populate client
	for _, option := range options {
		if err := option(client); err != nil {
			return nil, fmt.Errorf("failed to apply client option: %w", err)
		}
	}

	// 2. initialize HTTP client used by the auth source and by the GitLab client
	if err := client.initializeHTTPClient(); err != nil {
		return nil, err
	}

	// 3. initialize the auth source
	// We need to delay this because sources like OAuth2 need a valid
	// HTTP client to refresh the token.
	authSource, err := newAuthSource(client.httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize auth source: %w", err)
	}
	client.authSource = authSource

	// 4. initialize the GitLab client
	if client.gitlabClient != nil {
		return client, nil
	}

	if client.authSource == nil {
		return nil, errors.New("unable to initialize GitLab Client because no authentication source is provided. Login first")
	}

	gitlabClient, err := gitlab.NewAuthSourceClient(
		client.authSource,
		gitlab.WithHTTPClient(client.httpClient),
		gitlab.WithBaseURL(client.baseURL),
		gitlab.WithUserAgent(client.userAgent),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GitLab client: %v", err)
	}

	client.gitlabClient = gitlabClient
	return client, nil
}

func (c *Client) initializeHTTPClient() error {
	if c.httpClient != nil {
		return nil
	}

	// Create TLS configuration based on client settings
	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: c.allowInsecure,
	}

	// Set secure cipher suites for gitlab.com
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return err
	}
	if !glinstance.IsSelfHosted(u.Hostname()) {
		tlsConfig.CipherSuites = secureCipherSuites
	}

	// Configure custom CA if provided
	if c.caFile != "" {
		caCert, err := os.ReadFile(c.caFile)
		if err != nil {
			return fmt.Errorf("error reading cert file: %w", err)
		}
		// use system cert pool as a baseline
		caCertPool, err := x509.SystemCertPool()
		if err != nil {
			return err
		}
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
	}

	// Configure client certificates if provided
	if c.clientCertFile != "" && c.clientKeyFile != "" {
		clientCert, err := tls.LoadX509KeyPair(c.clientCertFile, c.clientKeyFile)
		if err != nil {
			return err
		}
		tlsConfig.Certificates = []tls.Certificate{clientCert}
	}

	// Set appropriate timeouts based on whether custom CA is used
	dialTimeout := 5 * time.Second
	keepAlive := 5 * time.Second
	idleTimeout := 30 * time.Second
	if c.caFile != "" {
		dialTimeout = 30 * time.Second
		keepAlive = 30 * time.Second
		idleTimeout = 90 * time.Second
	}

	c.httpClient = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   dialTimeout,
				KeepAlive: keepAlive,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       idleTimeout,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig:       tlsConfig,
		},
	}
	return nil
}

// WithCustomCA configures the client to use a custom CA certificate
func WithCustomCA(caFile string) ClientOption {
	return func(c *Client) error {
		c.caFile = caFile
		return nil
	}
}

// WithClientCertificate configures the client to use client certificates for mTLS
func WithClientCertificate(certFile, keyFile string) ClientOption {
	return func(c *Client) error {
		c.clientCertFile = certFile
		c.clientKeyFile = keyFile
		return nil
	}
}

// WithInsecureSkipVerify configures the client to skip TLS verification
func WithInsecureSkipVerify(skip bool) ClientOption {
	return func(c *Client) error {
		c.allowInsecure = skip
		return nil
	}
}

// WithHTTPClient configures the HTTP client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) error {
		c.httpClient = httpClient
		return nil
	}
}

// WithGitLabClient configures the GitLab client
func WithGitLabClient(client *gitlab.Client) ClientOption {
	return func(c *Client) error {
		c.gitlabClient = client
		return nil
	}
}

// WithBaseURL configures the base URL for the GitLab instance
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) error {
		c.baseURL = baseURL
		return nil
	}
}

// WithUserAgent configures the user agent to use
func WithUserAgent(userAgent string) ClientOption {
	return func(c *Client) error {
		c.userAgent = userAgent
		return nil
	}
}

// NewClientFromConfig initializes the global api with the config data
func NewClientFromConfig(repoHost string, cfg config.Config, isGraphQL bool, userAgent string) (*Client, error) {
	apiHost, _ := cfg.Get(repoHost, "api_host")
	if apiHost == "" {
		apiHost = repoHost
	}

	apiProtocol, _ := cfg.Get(repoHost, "api_protocol")
	if apiProtocol == "" {
		apiProtocol = glinstance.DefaultProtocol
	}

	isOAuth2Cfg, _ := cfg.Get(repoHost, "is_oauth2")

	token, _ := cfg.Get(repoHost, "token")
	jobToken, _ := cfg.Get(repoHost, "job_token")
	tlsVerify, _ := cfg.Get(repoHost, "skip_tls_verify")
	skipTlsVerify := tlsVerify == "true" || tlsVerify == "1"
	caCert, _ := cfg.Get(repoHost, "ca_cert")
	clientCert, _ := cfg.Get(repoHost, "client_cert")
	keyFile, _ := cfg.Get(repoHost, "client_key")

	// Build options based on configuration
	options := []ClientOption{
		WithUserAgent(userAgent),
	}

	// determine auth source
	var newAuthSource newAuthSource
	switch {
	case isOAuth2Cfg == "true":
		newAuthSource = func(client *http.Client) (gitlab.AuthSource, error) {
			ts, err := oauth2.NewConfigTokenSource(cfg, client, glinstance.DefaultProtocol, repoHost)
			if err != nil {
				return nil, err
			}
			return gitlab.OAuthTokenSource{TokenSource: ts}, nil
		}
	case jobToken != "":
		newAuthSource = func(*http.Client) (gitlab.AuthSource, error) {
			return gitlab.JobTokenAuthSource{Token: jobToken}, nil
		}
	default:
		newAuthSource = func(*http.Client) (gitlab.AuthSource, error) {
			return gitlab.AccessTokenAuthSource{Token: token}, nil
		}
	}

	var baseURL string
	if isGraphQL {
		baseURL = glinstance.GraphQLEndpoint(apiHost, apiProtocol)
	} else {
		baseURL = glinstance.APIEndpoint(apiHost, apiProtocol, "")
	}
	options = append(options, WithBaseURL(baseURL))

	if caCert != "" {
		options = append(options, WithCustomCA(caCert))
	}

	if clientCert != "" && keyFile != "" {
		options = append(options, WithClientCertificate(clientCert, keyFile))
	}

	if skipTlsVerify {
		options = append(options, WithInsecureSkipVerify(skipTlsVerify))
	}

	return NewClient(newAuthSource, options...)
}

func NewHTTPRequest(ctx context.Context, c *Client, method string, baseURL *url.URL, body io.Reader, headers []string, bodyIsJSON bool) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, baseURL.String(), body)
	if err != nil {
		return nil, err
	}

	for _, h := range headers {
		idx := strings.IndexRune(h, ':')
		if idx == -1 {
			return nil, fmt.Errorf("header %q requires a value separated by ':'", h)
		}
		name, value := h[0:idx], strings.TrimSpace(h[idx+1:])
		if strings.EqualFold(name, "Content-Length") {
			length, err := strconv.ParseInt(value, 10, 0)
			if err != nil {
				return nil, err
			}
			req.ContentLength = length
		} else {
			req.Header.Add(name, value)
		}
	}

	if bodyIsJSON && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}

	if c.Lab().UserAgent != "" {
		req.Header.Set("User-Agent", c.Lab().UserAgent)
	}

	name, value, err := c.authSource.Header(ctx)
	if err != nil {
		return nil, err
	}
	req.Header.Set(name, value)

	return req, nil
}

// Is404 checks if the error represents a 404 response
func Is404(err error) bool {
	// If the error is a typed response
	if errResponse, ok := err.(*gitlab.ErrorResponse); ok &&
		errResponse.Response != nil &&
		errResponse.Response.StatusCode == http.StatusNotFound {
		return true
	}

	// This can also come back as a string 404 from gitlab client-go
	if err != nil && err.Error() == "404 Not Found" {
		return true
	}

	return false
}
