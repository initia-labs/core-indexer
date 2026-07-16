package indexercron

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func resetKeybaseImageCache() {
	keybaseImageCacheMu.Lock()
	keybaseImageCache = make(map[string]keybaseImageCacheEntry)
	keybaseImageCacheMu.Unlock()
}

// newKeybaseTestServer serves the Keybase lookup endpoint and the image it points to.
// lookupHits counts how many times the lookup endpoint was called.
func newKeybaseTestServer(imageData []byte, lookupHits *atomic.Int64) *httptest.Server {
	mux := http.NewServeMux()
	var srv *httptest.Server
	mux.HandleFunc("/_/api/1.0/user/lookup.json", func(w http.ResponseWriter, r *http.Request) {
		lookupHits.Add(1)
		fmt.Fprintf(w, `{"them":[{"pictures":{"primary":{"url":"%s/image"}}}]}`, srv.URL)
	})
	mux.HandleFunc("/image", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(imageData)
	})
	srv = httptest.NewServer(mux)
	return srv
}

func withKeybaseTestServer(t *testing.T, srv *httptest.Server) {
	t.Helper()
	orig := keybaseBaseURL
	keybaseBaseURL = srv.URL
	t.Cleanup(func() {
		keybaseBaseURL = orig
		srv.Close()
	})
	resetKeybaseImageCache()
	t.Cleanup(resetKeybaseImageCache)
}

func TestFetchImageDataFromKeybase_EmptyIdentity(t *testing.T) {
	image, fromCache := fetchImageDataFromKeybase("")
	if image != "" || fromCache {
		t.Fatalf("expected empty result for empty identity, got image=%q fromCache=%v", image, fromCache)
	}
}

func TestFetchImageDataFromKeybase_FetchesAndCaches(t *testing.T) {
	// Not a valid image, so normalizeImageToJPEG fails and the original bytes are kept.
	imageData := []byte("fake-image-bytes")
	var lookupHits atomic.Int64
	srv := newKeybaseTestServer(imageData, &lookupHits)
	withKeybaseTestServer(t, srv)

	want := base64.StdEncoding.EncodeToString(imageData)

	image, fromCache := fetchImageDataFromKeybase("identity1")
	if image != want {
		t.Fatalf("first call: expected image %q, got %q", want, image)
	}
	if fromCache {
		t.Fatal("first call: expected fromCache=false")
	}
	if got := lookupHits.Load(); got != 1 {
		t.Fatalf("first call: expected 1 lookup hit, got %d", got)
	}

	image, fromCache = fetchImageDataFromKeybase("identity1")
	if image != want {
		t.Fatalf("second call: expected image %q, got %q", want, image)
	}
	if !fromCache {
		t.Fatal("second call: expected fromCache=true")
	}
	if got := lookupHits.Load(); got != 1 {
		t.Fatalf("second call: expected no extra lookup hit, got %d total", got)
	}
}

func TestFetchImageDataFromKeybase_ExpiredEntryIsRefetched(t *testing.T) {
	imageData := []byte("new-image-bytes")
	var lookupHits atomic.Int64
	srv := newKeybaseTestServer(imageData, &lookupHits)
	withKeybaseTestServer(t, srv)

	keybaseImageCacheMu.Lock()
	keybaseImageCache["identity1"] = keybaseImageCacheEntry{
		image:     "stale-image",
		fetchedAt: time.Now().Add(-keybaseImageCacheTTL - time.Minute),
	}
	keybaseImageCacheMu.Unlock()

	image, fromCache := fetchImageDataFromKeybase("identity1")
	want := base64.StdEncoding.EncodeToString(imageData)
	if image != want {
		t.Fatalf("expected refetched image %q, got %q", want, image)
	}
	if fromCache {
		t.Fatal("expected fromCache=false for expired entry")
	}
	if got := lookupHits.Load(); got != 1 {
		t.Fatalf("expected 1 lookup hit for expired entry, got %d", got)
	}

	// The cache entry should have been refreshed.
	keybaseImageCacheMu.RLock()
	entry := keybaseImageCache["identity1"]
	keybaseImageCacheMu.RUnlock()
	if entry.image != want {
		t.Fatalf("expected cache to hold refreshed image, got %q", entry.image)
	}
	if time.Since(entry.fetchedAt) >= keybaseImageCacheTTL {
		t.Fatal("expected cache entry timestamp to be refreshed")
	}
}

func TestFetchImageDataFromKeybase_FreshEntryIsServedFromCache(t *testing.T) {
	var lookupHits atomic.Int64
	srv := newKeybaseTestServer([]byte("should-not-be-fetched"), &lookupHits)
	withKeybaseTestServer(t, srv)

	keybaseImageCacheMu.Lock()
	keybaseImageCache["identity1"] = keybaseImageCacheEntry{
		image:     "cached-image",
		fetchedAt: time.Now(),
	}
	keybaseImageCacheMu.Unlock()

	image, fromCache := fetchImageDataFromKeybase("identity1")
	if image != "cached-image" {
		t.Fatalf("expected cached image, got %q", image)
	}
	if !fromCache {
		t.Fatal("expected fromCache=true for fresh entry")
	}
	if got := lookupHits.Load(); got != 0 {
		t.Fatalf("expected no lookup hits for fresh entry, got %d", got)
	}
}

func TestFetchImageDataFromKeybase_FailedRefetchDoesNotUpdateCache(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	withKeybaseTestServer(t, srv)

	staleEntry := keybaseImageCacheEntry{
		image:     "stale-image",
		fetchedAt: time.Now().Add(-keybaseImageCacheTTL - time.Minute),
	}
	keybaseImageCacheMu.Lock()
	keybaseImageCache["identity1"] = staleEntry
	keybaseImageCacheMu.Unlock()

	image, fromCache := fetchImageDataFromKeybase("identity1")
	if image != "" || fromCache {
		t.Fatalf("expected empty result on failed refetch, got image=%q fromCache=%v", image, fromCache)
	}

	keybaseImageCacheMu.RLock()
	entry := keybaseImageCache["identity1"]
	keybaseImageCacheMu.RUnlock()
	if entry != staleEntry {
		t.Fatalf("expected cache entry to be unchanged on failure, got %+v", entry)
	}
}
