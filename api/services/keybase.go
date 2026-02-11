package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/initia-labs/core-indexer/api/cache"
	"github.com/initia-labs/core-indexer/pkg/logger"
)

// KeybaseService handles interactions with the Keybase API
type KeybaseService struct {
	client     *http.Client
	imageCache *cache.TTLCache[string, string]
}

// keybaseResponse represents the structure of Keybase API response
type keybaseResponse struct {
	Them []struct {
		Pictures struct {
			Primary struct {
				URL string `json:"url"`
			} `json:"primary"`
		} `json:"pictures"`
	} `json:"them"`
}

// NewKeybaseService creates a new Keybase service with caching
func NewKeybaseService(cacheTTL time.Duration) *KeybaseService {
	return &KeybaseService{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		imageCache: cache.New[string, string](cacheTTL, 1000), // Cache up to 1000 identities
	}
}

// GetImageURL fetches and caches the validator image URL from Keybase
// Returns the image URL or empty string if not available
func (s *KeybaseService) GetImageURL(identity string) string {
	if identity == "" {
		return ""
	}

	// Check cache first
	if imageURL, found := s.imageCache.Get(identity); found {
		return imageURL
	}

	// Cache miss - fetch from Keybase
	imageURL := s.fetchImageURL(identity)

	// Store in cache (even if empty to avoid repeated failed lookups)
	s.imageCache.Set(identity, imageURL)

	return imageURL
}

// fetchImageURL makes the actual HTTP request to Keybase API
func (s *KeybaseService) fetchImageURL(identity string) string {
	url := fmt.Sprintf("https://keybase.io/_/api/1.0/user/lookup.json?key_suffix=%s&fields=pictures", identity)

	resp, err := s.client.Get(url)
	if err != nil {
		logger.Get().Warn().
			Err(err).
			Str("identity", identity).
			Msg("Failed to fetch Keybase image")
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Get().Warn().
			Int("status", resp.StatusCode).
			Str("identity", identity).
			Msg("Keybase API returned non-200 status")
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Get().Warn().
			Err(err).
			Str("identity", identity).
			Msg("Failed to read Keybase response")
		return ""
	}

	var keybaseResp keybaseResponse
	if err := json.Unmarshal(body, &keybaseResp); err != nil {
		logger.Get().Warn().
			Err(err).
			Str("identity", identity).
			Msg("Failed to parse Keybase response")
		return ""
	}

	// Extract image URL if available
	if len(keybaseResp.Them) > 0 && keybaseResp.Them[0].Pictures.Primary.URL != "" {
		return keybaseResp.Them[0].Pictures.Primary.URL
	}

	return ""
}

// ClearCache clears all cached image URLs
func (s *KeybaseService) ClearCache() {
	s.imageCache.Clear()
}

// GetCacheSize returns the current number of cached identities
func (s *KeybaseService) GetCacheSize() int {
	return s.imageCache.Size()
}
