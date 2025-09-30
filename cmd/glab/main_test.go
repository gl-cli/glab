package main

import (
	"testing"

	"go.uber.org/goleak"
)

// Test started when the test binary is started
// and calls the main function
func TestGlab(t *testing.T) { // nolint:unparam
	main()
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("internal/poll.runtime_pollWait"), // HTTP keep-alive connections
	)
}
