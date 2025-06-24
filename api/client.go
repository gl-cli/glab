package api

import (
	"crypto/tls"
	"crypto/x509"
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
	"gitlab.com/gitlab-org/cli/pkg/glinstance"
	"gitlab.com/gitlab-org/cli/pkg/oauth2"

	oauthz "golang.org/x/oauth2"
)

// ClientOption represents a function that configures a Client
type ClientOption func(*Client) error

// AuthType represents an authentication type within GitLab.
type authType int

const (
	NoToken authType = iota
	OAuthToken
	PrivateToken
)

type glabInstall struct {
	version, platform, architecture string
}

var currentGlabInstall glabInstall

// Global api client to be used throughout glab
var apiClient *Client

// Client represents an argument to NewClient
type Client struct {
	// LabClient represents GitLab API client.
	// Note: this is exported for tests. Do not access it directly. Use Lab() method
	LabClient *gitlab.Client
	// internal http client
	httpClient *http.Client
	// internal http client overrider
	httpClientOverride *http.Client
	// Token type used to make authenticated API calls.
	AuthType authType
	// custom certificate
	caFile string
	// client certificate files
	clientCertFile string
	clientKeyFile  string
	// Protocol: host url protocol to make requests. Default is https
	Protocol string

	host  string
	token string

	isGraphQL          bool
	isOauth2           bool
	isJobToken         bool
	allowInsecure      bool
	RefreshLabInstance bool
}

func (i glabInstall) UserAgent() string {
	return fmt.Sprintf("glab/%s (%s, %s)", i.version, i.platform, i.architecture)
}

func SetUserAgent(version string, platform string, architecture string) {
	currentGlabInstall = glabInstall{
		version:      version,
		platform:     platform,
		architecture: architecture,
	}
}

func init() {
	// initialise the global api client to be used throughout glab
	RefreshClient()
}

// RefreshClient re-initializes the api client
func RefreshClient() {
	apiClient = &Client{
		Protocol:           "https",
		AuthType:           NoToken,
		httpClient:         &http.Client{},
		RefreshLabInstance: true,
	}
}

func (c *Client) HTTPClient() *http.Client {
	if c.httpClientOverride != nil {
		return c.httpClientOverride
	}
	if c.httpClient != nil {
		return c.httpClient
	}
	return &http.Client{}
}

func (c *Client) OverrideHTTPClient(client *http.Client) {
	c.httpClientOverride = client
}

func (c *Client) Token() string {
	return c.token
}

func (c *Client) SetProtocol(protocol string) {
	c.Protocol = protocol
}

var secureCipherSuites = []uint16{
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
}

func tlsConfig(host string, allowInsecure bool) *tls.Config {
	config := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: allowInsecure,
	}

	if host == "gitlab.com" {
		config.CipherSuites = secureCipherSuites
	}

	return config
}

// NewClient initializes a api client for use throughout glab.
func NewClient(host, token string, isGraphQL bool, isOAuth2 bool, isJobToken bool, options ...ClientOption) (*Client, error) {
	apiClient.host = host
	apiClient.token = token
	apiClient.isGraphQL = isGraphQL
	apiClient.isOauth2 = isOAuth2
	apiClient.isJobToken = isJobToken

	// Apply options
	for _, option := range options {
		if err := option(apiClient); err != nil {
			return nil, fmt.Errorf("failed to apply client option: %w", err)
		}
	}

	if apiClient.httpClientOverride == nil {
		// Create TLS configuration based on client settings
		tlsConfig := &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: apiClient.allowInsecure,
		}

		// Set secure cipher suites for gitlab.com
		if host == "gitlab.com" {
			tlsConfig.CipherSuites = secureCipherSuites
		}

		// Configure custom CA if provided
		if apiClient.caFile != "" {
			caCert, err := os.ReadFile(apiClient.caFile)
			if err != nil {
				return nil, fmt.Errorf("error reading cert file: %w", err)
			}
			// use system cert pool as a baseline
			caCertPool, err := x509.SystemCertPool()
			if err != nil {
				return nil, err
			}
			caCertPool.AppendCertsFromPEM(caCert)
			tlsConfig.RootCAs = caCertPool
		}

		// Configure client certificates if provided
		if apiClient.clientCertFile != "" && apiClient.clientKeyFile != "" {
			clientCert, err := tls.LoadX509KeyPair(apiClient.clientCertFile, apiClient.clientKeyFile)
			if err != nil {
				return nil, err
			}
			tlsConfig.Certificates = []tls.Certificate{clientCert}
		}

		// Set appropriate timeouts based on whether custom CA is used
		dialTimeout := 5 * time.Second
		keepAlive := 5 * time.Second
		idleTimeout := 30 * time.Second
		if apiClient.caFile != "" {
			dialTimeout = 30 * time.Second
			keepAlive = 30 * time.Second
			idleTimeout = 90 * time.Second
		}

		apiClient.httpClient = &http.Client{
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
	}
	apiClient.RefreshLabInstance = true
	err := apiClient.NewLab()
	return apiClient, err
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

// WithProtocol configures the client protocol
func WithProtocol(protocol string) ClientOption {
	return func(c *Client) error {
		c.Protocol = protocol
		return nil
	}
}

// NewClientWithCfg initializes the global api with the config data
func NewClientWithCfg(repoHost string, cfg config.Config, isGraphQL bool) (*Client, error) {
	if repoHost == "" {
		repoHost = glinstance.OverridableDefault()
	}

	apiHost, _ := cfg.Get(repoHost, "api_host")
	if apiHost == "" {
		apiHost = repoHost
	}

	apiProtocol, _ := cfg.Get(repoHost, "api_protocol")
	if apiProtocol != "" {
		apiClient.SetProtocol(apiProtocol) // TODO remove this together with global apiClient.
	}

	isOAuth2Cfg, _ := cfg.Get(repoHost, "is_oauth2")
	isOAuth2 := false
	if isOAuth2Cfg == "true" {
		isOAuth2 = true
		err := oauth2.RefreshToken(repoHost, cfg, "https")
		if err != nil {
			return nil, err
		}
	}

	token, _ := cfg.Get(repoHost, "token")
	jobToken, _ := cfg.Get(repoHost, "job_token")
	tlsVerify, _ := cfg.Get(repoHost, "skip_tls_verify")
	skipTlsVerify := tlsVerify == "true" || tlsVerify == "1"
	caCert, _ := cfg.Get(repoHost, "ca_cert")
	clientCert, _ := cfg.Get(repoHost, "client_cert")
	keyFile, _ := cfg.Get(repoHost, "client_key")

	authToken := token
	isJobToken := false
	if jobToken != "" {
		authToken = jobToken
		isJobToken = true
	}

	// Build options based on configuration
	var options []ClientOption

	if caCert != "" {
		options = append(options, WithCustomCA(caCert))
	}

	if clientCert != "" && keyFile != "" {
		options = append(options, WithClientCertificate(clientCert, keyFile))
	}

	if skipTlsVerify {
		options = append(options, WithInsecureSkipVerify(skipTlsVerify))
	}

	if apiProtocol != "" {
		options = append(options, WithProtocol(apiProtocol))
	}

	client, err := NewClient(apiHost, authToken, isGraphQL, isOAuth2, isJobToken, options...)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// NewLab initializes the GitLab Client
func (c *Client) NewLab() error {
	var err error
	var baseURL string
	httpClient := c.httpClient

	if c.httpClientOverride != nil {
		httpClient = c.httpClientOverride
	}
	if apiClient.RefreshLabInstance {
		if c.host == "" {
			c.host = glinstance.OverridableDefault()
		}
		if c.isGraphQL {
			baseURL = glinstance.GraphQLEndpoint(c.host, c.Protocol)
		} else {
			baseURL = glinstance.APIEndpoint(c.host, c.Protocol, "")
		}

		if c.isOauth2 {
			ts := oauthz.StaticTokenSource(&oauthz.Token{AccessToken: c.token})
			c.LabClient, err = gitlab.NewAuthSourceClient(gitlab.OAuthTokenSource{TokenSource: ts}, gitlab.WithHTTPClient(httpClient), gitlab.WithBaseURL(baseURL))
		} else if c.isJobToken {
			c.LabClient, err = gitlab.NewJobClient(c.token, gitlab.WithHTTPClient(httpClient), gitlab.WithBaseURL(baseURL))
		} else {
			c.LabClient, err = gitlab.NewClient(c.token, gitlab.WithHTTPClient(httpClient), gitlab.WithBaseURL(baseURL))
		}

		if err != nil {
			return fmt.Errorf("failed to initialize GitLab client: %v", err)
		}
		c.LabClient.UserAgent = currentGlabInstall.UserAgent()

		if c.token != "" {
			if c.isOauth2 {
				c.AuthType = OAuthToken
			} else {
				c.AuthType = PrivateToken
			}
		}
	}
	return nil
}

// Lab returns the initialized GitLab client.
// Initializes a new GitLab Client if not initialized but error is ignored
func (c *Client) Lab() *gitlab.Client {
	if c.LabClient != nil {
		return c.LabClient
	}
	err := c.NewLab()
	if err != nil {
		c.LabClient = &gitlab.Client{}
	}
	return c.LabClient
}

// BaseURL returns a copy of the BaseURL
func (c *Client) BaseURL() *url.URL {
	return c.Lab().BaseURL()
}

func NewHTTPRequest(c *Client, method string, baseURL *url.URL, body io.Reader, headers []string, bodyIsJSON bool) (*http.Request, error) {
	req, err := http.NewRequest(method, baseURL.String(), body)
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

	// TODO: support GITLAB_CI_TOKEN
	switch c.AuthType {
	case OAuthToken:
		req.Header.Set("Authorization", "Bearer "+c.Token())
	case PrivateToken:
		req.Header.Set("PRIVATE-TOKEN", c.Token())
	}

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
