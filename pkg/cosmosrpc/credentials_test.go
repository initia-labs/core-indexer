package cosmosrpc

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestRPCCredentialsStayOutOfLogsAndErrors(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/path-secret" || r.URL.Query().Get("key") != "query-secret" || r.Header.Get("X-API-Key") != "header-secret" {
			t.Error("request credentials or destination changed")
		}
		username, password, ok := r.BasicAuth()
		if !ok || username != "user-secret" || password != "password-secret" {
			t.Error("request authentication changed")
		}
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, "unavailable")
	}))
	defer server.Close()
	endpoint := strings.Replace(server.URL, "http://", "http://user-secret:password-secret@", 1) + "/path-secret?key=query-secret"
	var output bytes.Buffer
	logger := zerolog.New(&output)
	hub := NewHub([]ClientConfig{{URL: endpoint, ClientOption: &ClientOption{CustomHeaders: map[string]string{"X-API-Key": "header-secret"}}}}, &logger, time.Second)
	err := hub.Rebalance(context.Background())
	if err == nil {
		t.Fatal("expected HTTP failure")
	}
	if requests != 1 {
		t.Fatalf("requests=%d, want 1", requests)
	}
	combined := output.String() + err.Error() + hub.GetIdentifier()
	for _, secret := range []string{"user-secret", "password-secret", "path-secret", "query-secret", "header-secret"} {
		if strings.Contains(combined, secret) {
			t.Fatalf("RPC diagnostics expose %s", secret)
		}
	}
	if !strings.Contains(combined, server.URL) || !strings.Contains(combined, "502") {
		t.Fatalf("diagnostics lost endpoint/status: %s", combined)
	}
	plain := NewClient(endpoint)
	if plain.GetIdentifier() != server.URL {
		t.Fatal("default client identifier contains credentials")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = plain.Status(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("redaction lost cancellation identity: %v", err)
	}
	for _, secret := range []string{"user-secret", "password-secret", "path-secret", "query-secret"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("transport error exposes %s", secret)
		}
	}
}
