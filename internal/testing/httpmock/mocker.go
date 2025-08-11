package httpmock

import (
	"fmt"
	"net/http"
	"sync"
)

type matchType int

const (
	PathOnly matchType = iota
	HostOnly
	FullURL
	HostAndPath
	PathAndQuerystring
)

type Mocker struct {
	mu       sync.Mutex
	stubs    []*Stub
	Requests []*http.Request
	MatchURL matchType // if false, only matches the path and if true, matches full url
	Debug    bool      // Add a debug flag for additional logging
}

func New() *Mocker {
	return &Mocker{}
}

func (r *Mocker) RegisterResponder(method, path string, resp Responder) {
	matcher, url := newRequest(method, path, r.MatchURL)
	r.stubs = append(r.stubs, &Stub{
		Matcher:   matcher,
		Reusable:  false,
		Responder: resp,
		Method:    method,
		URL:       url,
	})
}

func (r *Mocker) RegisterResponderWithBody(method, path, body string, resp Responder) {
	matcher, url := newRequestWithBody(method, path, body)
	r.stubs = append(r.stubs, &Stub{
		Matcher:   matcher,
		Reusable:  false,
		Responder: resp,
		body:      body,
		Method:    method,
		URL:       url,
	})
}

func (r *Mocker) RegisterReusableResponder(method, path string, resp Responder) {
	matcher, url := newRequest(method, path, r.MatchURL)
	r.stubs = append(r.stubs, &Stub{
		Matcher:   matcher,
		Reusable:  true,
		Responder: resp,
		Method:    method,
		URL:       url,
	})
}

func (r *Mocker) RegisterReusableResponderWithBody(method, path, body string, resp Responder) {
	matcher, url := newRequestWithBody(method, path, body)
	r.stubs = append(r.stubs, &Stub{
		Matcher:   matcher,
		Reusable:  true,
		Responder: resp,
		body:      body,
		Method:    method,
		URL:       url,
	})
}

type Testing interface {
	Errorf(string, ...any)
	Helper()
	Logf(string, ...interface{})
}

func (r *Mocker) Verify(t Testing) {
	n := 0
	for _, s := range r.stubs {
		if !s.Used {
			n++
			if r.Debug {
				url := "unknown"
				if s.URL != nil {
					url = s.URL.String()
				}
				t.Logf("Unmatched HTTP stub: %s %s", s.Method, url)
			}
		}
	}
	if n > 0 {
		t.Helper()
		t.Errorf("%d unmatched HTTP stubs", n)
	}
}

// RoundTrip satisfies http.RoundTripper
func (r *Mocker) RoundTrip(req *http.Request) (*http.Response, error) {
	var stub *Stub

	r.mu.Lock()
	for _, s := range r.stubs {
		if (!s.Reusable && s.Used) || !s.Matcher(req) {
			continue
		}
		if stub != nil {
			r.mu.Unlock()
			return nil, fmt.Errorf("more than one stub matched %v.", req)
		}
		stub = s
	}
	if stub != nil {
		stub.Used = true // Changed from stub.matched to stub.Used
	}

	if stub == nil {
		if r.Debug {
			// Print useful debugging information
			r.mu.Unlock()
			return nil, fmt.Errorf("no registered stubs matched: %s %s", req.Method, req.URL.String())
		}
		r.mu.Unlock()
		return nil, fmt.Errorf("no registered stubs matched %v.", req)
	}

	r.Requests = append(r.Requests, req)
	r.mu.Unlock()

	return stub.Responder(req)
}

func (r *Mocker) GetStubs() []*Stub {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Create a copy of the stubs slice to prevent external modifications
	stubs := make([]*Stub, len(r.stubs))
	copy(stubs, r.stubs)
	return stubs
}

// PathPrefix returns a matcher that matches any path with the given prefix
func PathPrefix(prefix string) string {
	return prefix + "*"
}
