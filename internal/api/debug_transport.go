package api

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var sensitiveHeaders = []string{
	gitlab.AccessTokenHeaderName,
	gitlab.JobTokenHeaderName,
	"Authorization",
}

type debugTransport struct {
	rt http.RoundTripper
	w  io.Writer
}

func (d *debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	reqDump, err := httputil.DumpRequestOut(redactKnownSensitiveHeaders(req), true)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(d.w, "REQUEST:\n%s\n\n", reqDump)

	// Do request
	resp, err := d.rt.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// Dump response
	respDump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(d.w, "RESPONSE:\n%s\n\n", respDump)

	return resp, nil
}

func redactKnownSensitiveHeaders(req *http.Request) *http.Request {
	cloned := req.Clone(req.Context())
	for _, h := range sensitiveHeaders {
		if cloned.Header.Get(h) != "" {
			cloned.Header.Set(h, "[REDACTED]")
		}
	}
	return cloned
}
