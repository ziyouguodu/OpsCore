package main

import (
	"net/http"
	"testing"
	"time"
)

func TestHTTPServerUsesProductionTimeouts(t *testing.T) {
	server := newHTTPServer(":0", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	if server.ReadHeaderTimeout != 5*time.Second || server.ReadTimeout != 15*time.Second {
		t.Fatalf("unexpected read timeouts: header=%s read=%s", server.ReadHeaderTimeout, server.ReadTimeout)
	}
	if server.WriteTimeout != 30*time.Second || server.IdleTimeout != 60*time.Second {
		t.Fatalf("unexpected write timeouts: write=%s idle=%s", server.WriteTimeout, server.IdleTimeout)
	}
}
