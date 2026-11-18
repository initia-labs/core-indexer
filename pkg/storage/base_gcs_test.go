package storage

import (
	"context"
	"net/http"
	"testing"
	"time"

	"google.golang.org/api/option"
)

type readTransport func(*http.Request) (*http.Response, error)

func (f readTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestReadFileContextCancelsInflightRead(t *testing.T) {
	started := make(chan struct{}, 1)
	client, err := NewBaseGCSClient(option.WithHTTPClient(&http.Client{Transport: readTransport(func(r *http.Request) (*http.Response, error) {
		started <- struct{}{}
		<-r.Context().Done()
		return nil, r.Context().Err()
	})}))
	if err != nil {
		t.Fatal(err)
	}
	defer client.client.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := client.ReadFileContext(ctx, "bucket", "object"); done <- err }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("read did not start")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("canceled read succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("read ignored cancellation")
	}
}
