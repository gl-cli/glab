package api

import (
	"io"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func TestConfigHeadersViaNewHTTPRequest(t *testing.T) {
	tests := []struct {
		name          string
		customHeaders map[string]string
		expectValue   map[string]string
	}{
		{
			name: "basic header injection",
			customHeaders: map[string]string{
				"X-Custom-Header": "custom-value",
			},
			expectValue: map[string]string{"X-Custom-Header": "custom-value"},
		},
		{
			name: "proxy authorization header",
			customHeaders: map[string]string{
				"Proxy-Authorization": "Bearer token123",
			},
			expectValue: map[string]string{"Proxy-Authorization": "Bearer token123"},
		},
		{
			name: "authorization header",
			customHeaders: map[string]string{
				"Authorization": "Bearer token456",
			},
			expectValue: map[string]string{"Authorization": "Bearer token456"},
		},
		{
			name: "multiple headers",
			customHeaders: map[string]string{
				"X-Custom-Header":     "value1",
				"Authorization":       "Bearer token",
				"Proxy-Authorization": "Bearer proxy-token",
			},
			expectValue: map[string]string{
				"X-Custom-Header":     "value1",
				"Authorization":       "Bearer token",
				"Proxy-Authorization": "Bearer proxy-token",
			},
		},
		{
			name: "cloudflare access headers",
			customHeaders: map[string]string{
				"Cf-Access-Client-Id":     "client-123",
				"Cf-Access-Client-Secret": "secret-456",
			},
			expectValue: map[string]string{
				"Cf-Access-Client-Id":     "client-123",
				"Cf-Access-Client-Secret": "secret-456",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create client with custom headers
			mockGitLabClient := &gitlab.Client{}
			client := &Client{
				gitlabClient:  mockGitLabClient,
				authSource:    gitlab.AccessTokenAuthSource{Token: "test-token"},
				customHeaders: tc.customHeaders,
			}

			// Create request using NewHTTPRequest
			baseURL, _ := url.Parse("https://example.com/api")
			req, err := NewHTTPRequest(t.Context(), client, "GET", baseURL, nil, []string{}, false)
			require.NoError(t, err)

			// Check that expected headers are present
			for expectedHeader, expectedValue := range tc.expectValue {
				actualValue := req.Header.Get(expectedHeader)
				require.Equal(t, expectedValue, actualValue, "Header %s should be %s but got %s", expectedHeader, expectedValue, actualValue)
			}
		})
	}
}

func TestCustomHeadersIntegration(t *testing.T) {
	// Create a client with custom headers configured
	mockGitLabClient := &gitlab.Client{}
	client := &Client{
		gitlabClient: mockGitLabClient,
		authSource:   gitlab.AccessTokenAuthSource{Token: "test-token"},
		customHeaders: map[string]string{
			"X-Test-Header":       "test-value",
			"Proxy-Authorization": "Bearer test-token",
		},
	}

	// Create request using NewHTTPRequest
	baseURL, _ := url.Parse("https://example.com/api")
	req, err := NewHTTPRequest(t.Context(), client, "GET", baseURL, nil, []string{}, false)
	require.NoError(t, err)

	require.Equal(t, "test-value", req.Header.Get("X-Test-Header"))
	require.Equal(t, "Bearer test-token", req.Header.Get("Proxy-Authorization"))
}

func TestCustomHeadersWithRequestBody(t *testing.T) {
	// Create a client with custom headers
	mockGitLabClient := &gitlab.Client{}
	client := &Client{
		gitlabClient: mockGitLabClient,
		authSource:   gitlab.AccessTokenAuthSource{Token: "test-token"},
		customHeaders: map[string]string{
			"X-Custom-Header":     "custom-value",
			"Content-Type":        "application/json",
			"Proxy-Authorization": "Bearer proxy-token",
		},
	}

	// Create request with a non-empty body using NewHTTPRequest
	baseURL, _ := url.Parse("https://example.com/api")
	body := strings.NewReader(`{"key": "value", "data": "test"}`)
	req, err := NewHTTPRequest(t.Context(), client, "POST", baseURL, body, []string{}, false)
	require.NoError(t, err)

	// Check that custom headers are present even with a request body
	require.Equal(t, "custom-value", req.Header.Get("X-Custom-Header"))
	require.Equal(t, "application/json", req.Header.Get("Content-Type"))
	require.Equal(t, "Bearer proxy-token", req.Header.Get("Proxy-Authorization"))

	// Verify that the request body is preserved
	bodyBytes, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	require.Equal(t, `{"key": "value", "data": "test"}`, string(bodyBytes))
}

func TestClientInitializationWithNoCustomHeaders(t *testing.T) {
	tests := []struct {
		name          string
		customHeaders map[string]string
	}{
		{
			name:          "nil custom headers",
			customHeaders: nil,
		},
		{
			name:          "empty custom headers map",
			customHeaders: map[string]string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create client with no custom headers
			mockGitLabClient := &gitlab.Client{}
			client := &Client{
				gitlabClient:  mockGitLabClient,
				authSource:    gitlab.AccessTokenAuthSource{Token: "test-token"},
				customHeaders: tc.customHeaders,
			}

			// Create request using NewHTTPRequest - should not fail
			baseURL, _ := url.Parse("https://example.com/api")
			req, err := NewHTTPRequest(t.Context(), client, "GET", baseURL, nil, []string{}, false)
			require.NoError(t, err)
			require.NotNil(t, req)

			// Verify no custom headers are present (only standard headers like User-Agent, etc.)
			require.Empty(t, req.Header.Get("X-Custom-Header"))
			require.Empty(t, req.Header.Get("Proxy-Authorization"))
			require.Empty(t, req.Header.Get("Authorization")) // Should be empty since we're testing custom headers, not auth
		})
	}
}
