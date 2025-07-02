package api

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/test"
)

func Test_groupGraphQLVariables(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
		want map[string]any
	}{
		{
			name: "empty",
			args: map[string]any{},
			want: map[string]any{},
		},
		{
			name: "query only",
			args: map[string]any{
				"query": "QUERY",
			},
			want: map[string]any{
				"query": "QUERY",
			},
		},
		{
			name: "variables only",
			args: map[string]any{
				"name": "gitlab-bot",
			},
			want: map[string]any{
				"variables": map[string]any{
					"name": "gitlab-bot",
				},
			},
		},
		{
			name: "query + variables",
			args: map[string]any{
				"query": "QUERY",
				"name":  "gitlab-bot",
				"power": 9001,
			},
			want: map[string]any{
				"query": "QUERY",
				"variables": map[string]any{
					"name":  "gitlab-bot",
					"power": 9001,
				},
			},
		},
		{
			name: "query + operationName + variables",
			args: map[string]any{
				"query":         "query Q1{} query Q2{}",
				"operationName": "Q1",
				"power":         9001,
			},
			want: map[string]any{
				"query":         "query Q1{} query Q2{}",
				"operationName": "Q1",
				"variables": map[string]any{
					"power": 9001,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := groupGraphQLVariables(tt.args)
			assert.Equal(t, tt.want, got)
		})
	}
}

type roundTripFunc func(r *http.Request) (*http.Response, error)

func (s roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return s(r)
}

func Test_httpRequest(t *testing.T) {
	defer config.StubConfig(`---
hosts:
  gitlab.com:
    username: monalisa
    token: OTOKEN
`, "")()
	test.ClearEnvironmentVariables(t)

	client := &http.Client{}
	client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Log("Tsti")
		return &http.Response{
			StatusCode: http.StatusOK,
			Request:    req,
		}, nil
	})

	type args struct {
		host    string
		method  string
		p       string
		params  any
		headers []string
	}
	type expects struct {
		method  string
		u       string
		body    string
		headers string
	}
	tests := []struct {
		isGraphQL bool
		name      string
		args      args
		want      expects
		wantErr   bool
	}{
		{
			name: "simple GET",
			args: args{
				host:    "gitlab.com",
				method:  http.MethodGet,
				p:       "projects/gitlab-com%2Fwww-gitlab-com",
				params:  nil,
				headers: []string{},
			},
			wantErr:   false,
			isGraphQL: false,
			want: expects{
				method:  http.MethodGet,
				u:       "https://gitlab.com/api/v4/projects/gitlab-com%2Fwww-gitlab-com",
				body:    "",
				headers: "Private-Token: OTOKEN\r\nUser-Agent: glab test client\r\n",
			},
		},
		{
			name: "GET with leading slash",
			args: args{
				host:    "gitlab.com",
				method:  http.MethodGet,
				p:       "/projects/gitlab-com%2Fwww-gitlab-com",
				params:  nil,
				headers: []string{},
			},
			wantErr:   false,
			isGraphQL: false,
			want: expects{
				method:  http.MethodGet,
				u:       "https://gitlab.com/api/v4/projects/gitlab-com%2Fwww-gitlab-com",
				body:    "",
				headers: "Private-Token: OTOKEN\r\nUser-Agent: glab test client\r\n",
			},
		},
		{
			name: "GET with params",
			args: args{
				host:   "gitlab.com",
				method: http.MethodGet,
				p:      "projects/gitlab-com%2Fwww-gitlab-com",
				params: map[string]any{
					"a": "b",
				},
				headers: []string{},
			},
			wantErr:   false,
			isGraphQL: false,
			want: expects{
				method:  http.MethodGet,
				u:       "https://gitlab.com/api/v4/projects/gitlab-com%2Fwww-gitlab-com?a=b",
				body:    "",
				headers: "Private-Token: OTOKEN\r\nUser-Agent: glab test client\r\n",
			},
		},
		{
			name: "POST GraphQL",
			args: args{
				host:   "gitlab.com",
				method: http.MethodPost,
				p:      "graphql",
				params: map[string]any{
					"a": []byte("b"),
				},
				headers: []string{},
			},
			wantErr:   false,
			isGraphQL: true,
			want: expects{
				method:  http.MethodPost,
				u:       "https://gitlab.com/api/graphql/",
				body:    `{"variables":{"a":"b"}}`,
				headers: "Content-Type: application/json; charset=utf-8\r\nPrivate-Token: OTOKEN\r\nUser-Agent: glab test client\r\n",
			},
		},
		{
			name: "POST with body and type",
			args: args{
				host:   "gitlab.com",
				method: http.MethodPost,
				p:      "projects",
				params: bytes.NewBufferString("CUSTOM"),
				headers: []string{
					"content-type: text/plain",
					"accept: application/json",
				},
			},
			wantErr:   false,
			isGraphQL: false,
			want: expects{
				method:  http.MethodPost,
				u:       "https://gitlab.com/api/v4/projects",
				body:    `CUSTOM`,
				headers: "Accept: application/json\r\nContent-Type: text/plain\r\nPrivate-Token: OTOKEN\r\nUser-Agent: glab test client\r\n",
			},
		},
		{
			name: "POST with string array field and type",
			args: args{
				host:   "gitlab.com",
				method: http.MethodPost,
				p:      "projects",
				params: map[string]any{"scopes": "[api, read_api]"},
				headers: []string{
					"content-type: application/json",
					"accept: application/json",
				},
			},
			wantErr:   false,
			isGraphQL: false,
			want: expects{
				method:  http.MethodPost,
				u:       "https://gitlab.com/api/v4/projects",
				body:    `{"scopes":["api","read_api"]}`,
				headers: "Accept: application/json\r\nContent-Type: application/json\r\nPrivate-Token: OTOKEN\r\nUser-Agent: glab test client\r\n",
			},
		},
	}
	for _, tt := range tests {
		var options []api.ClientOption
		if tt.isGraphQL {
			options = append(options, api.WithBaseURL(glinstance.GraphQLEndpoint(tt.args.host, glinstance.DefaultProtocol)))
		}
		httpClient := cmdtest.NewTestApiClient(t, client, "OTOKEN", tt.args.host, options...)
		t.Run(tt.name, func(t *testing.T) {
			got, err := httpRequest(t.Context(), httpClient, tt.args.method, tt.args.p, tt.args.params, tt.args.headers)
			if (err != nil) != tt.wantErr {
				t.Errorf("httpRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !assert.NotNil(t, got) {
				return
			}
			req := got.Request
			if req.Method != tt.want.method {
				t.Errorf("Request.Method = %q, want %q", req.Method, tt.want.method)
			}
			if req.URL.String() != tt.want.u {
				t.Errorf("Request.URL = %q, want %q", req.URL.String(), tt.want.u)
			}
			if tt.want.body != "" {
				bb, err := io.ReadAll(req.Body)
				if err != nil {
					t.Errorf("Request.Body ReadAll error = %v", err)
					return
				}
				if string(bb) != tt.want.body {
					t.Errorf("Request.Body = %q, want %q", string(bb), tt.want.body)
				}
			}

			h := bytes.Buffer{}
			err = req.Header.WriteSubset(&h, map[string]bool{})
			if err != nil {
				t.Errorf("Request.Header WriteSubset error = %v", err)
				return
			}
			if h.String() != tt.want.headers {
				t.Errorf("Request.Header = %q, want %q", h.String(), tt.want.headers)
			}
		})
	}
}

func Test_addQuery(t *testing.T) {
	type args struct {
		path   string
		params map[string]any
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "string",
			args: args{
				path:   "",
				params: map[string]any{"a": "hello"},
			},
			want: "?a=hello",
		},
		{
			name: "append",
			args: args{
				path:   "path",
				params: map[string]any{"a": "b"},
			},
			want: "path?a=b",
		},
		{
			name: "append query",
			args: args{
				path:   "path?foo=bar",
				params: map[string]any{"a": "b"},
			},
			want: "path?foo=bar&a=b",
		},
		{
			name: "[]byte",
			args: args{
				path:   "",
				params: map[string]any{"a": []byte("hello")},
			},
			want: "?a=hello",
		},
		{
			name: "int",
			args: args{
				path:   "",
				params: map[string]any{"a": 123},
			},
			want: "?a=123",
		},
		{
			name: "nil",
			args: args{
				path:   "",
				params: map[string]any{"a": nil},
			},
			want: "?a=",
		},
		{
			name: "bool",
			args: args{
				path:   "",
				params: map[string]any{"a": true, "b": false},
			},
			want: "?a=true&b=false",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := parseQuery(tt.args.path, tt.args.params); got != tt.want {
				if err != nil {
					t.Error(err.Error())
				}
				t.Errorf("parseQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}
