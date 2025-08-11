package httpmock

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
)

type (
	Matcher   func(req *http.Request) bool
	Responder func(req *http.Request) (*http.Response, error)
)

type Stub struct {
	Used      bool
	Reusable  bool
	Matcher   Matcher
	Responder Responder
	body      string
	Method    string   // Field to track the method
	URL       *url.URL // Field to track the URL
}

func newRequest(method, path string, match matchType) (Matcher, *url.URL) {
	u, _ := url.Parse(path) // Parse the URL here to reuse it

	matcher := func(req *http.Request) bool {
		if !strings.EqualFold(req.Method, method) {
			return false
		}
		if match == PathOnly {
			if !strings.HasPrefix(path, "/api/v4") {
				path = "/api/v4" + path
			}
			return req.URL.Path == path
		}

		if u != nil {
			if match == FullURL {
				return req.URL.String() == u.String()
			}
			if match == HostOnly {
				return req.URL.Host == u.Host
			}
			if match == HostAndPath {
				return req.URL.Host == u.Host && req.URL.Path == u.Path
			}
			if match == PathAndQuerystring {
				return req.URL.RawQuery == u.RawQuery && req.URL.Path == u.Path
			}
		}
		return false
	}

	return matcher, u
}

func newRequestWithBody(method, path, body string) (Matcher, *url.URL) {
	u, _ := url.Parse(path) // Parse the URL here to reuse it

	matcher := func(req *http.Request) bool {
		if !strings.EqualFold(req.Method, method) {
			return false
		}

		bytedata, _ := io.ReadAll(req.Body)
		reqBodyString := string(bytedata)

		return req.URL.RawQuery == u.RawQuery && req.URL.Path == u.Path && bodyEqual(reqBodyString, body)
	}

	return matcher, u
}

func NewStringResponse(status int, body string) Responder {
	return func(req *http.Request) (*http.Response, error) {
		return httpResponse(status, req, bytes.NewBufferString(body), nil), nil
	}
}

func NewStringResponseWithHeader(status int, body string, header http.Header) Responder {
	return func(req *http.Request) (*http.Response, error) {
		return httpResponse(status, req, bytes.NewBufferString(body), header), nil
	}
}

func NewJSONResponse(status int, body any) Responder {
	return func(req *http.Request) (*http.Response, error) {
		b, _ := json.Marshal(body)
		return httpResponse(status, req, bytes.NewBuffer(b), nil), nil
	}
}

func NewFileResponse(status int, filename string) Responder {
	return func(req *http.Request) (*http.Response, error) {
		f, err := os.Open(filename)
		if err != nil {
			return nil, err
		}
		return httpResponse(status, req, f, nil), nil
	}
}

func httpResponse(status int, req *http.Request, body io.Reader, header http.Header) *http.Response {
	return &http.Response{
		StatusCode: status,
		Request:    req,
		Body:       io.NopCloser(body),
		Header:     header,
	}
}

func bodyEqual(expected, actual string) bool {
	var expectedJSON, actualJSON any

	_ = json.Unmarshal([]byte(expected), &expectedJSON)
	_ = json.Unmarshal([]byte(actual), &actualJSON)

	return reflect.DeepEqual(expectedJSON, actualJSON)
}
