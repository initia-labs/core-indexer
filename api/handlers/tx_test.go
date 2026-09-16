package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/requestlog"
	"github.com/initia-labs/core-indexer/api/services"
)

type callerTxService struct{ services.TxService }

func (s callerTxService) GetTxByHash(ctx context.Context, hash string) (*dto.TxByHashResponse, error) {
	// Verify that downstream services receive the same logging context.
	requestlog.FromContext(ctx).Info().Str("hash", hash).Msg("service lookup")
	return &dto.TxByHashResponse{}, nil
}

func TestGetTxByHashCallerLogging(t *testing.T) {
	for _, tt := range []struct{ name, header, caller, path string }{
		{"direct", "", "direct", ""},
		{"scan", "/v1/initia/interwoven-1/txs/ABC?secret=excluded", "scan-api", "/v1/initia/interwoven-1/txs/ABC"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			previous := log.Logger
			log.Logger = zerolog.New(&output)
			defer func() { log.Logger = previous }()
			app := fiber.New()
			app.Get("/indexer/tx/v1/txs/:tx_hash", NewTxHandler(callerTxService{}).GetTxByHash)
			req := httptest.NewRequest("GET", "/indexer/tx/v1/txs/ABC", nil)
			if tt.header != "" {
				req.Header.Set("X-Scan-Api-Path", tt.header)
			}
			res, err := app.Test(req)
			require.NoError(t, err)
			defer res.Body.Close()
			require.Equal(t, 200, res.StatusCode)
			decoder := json.NewDecoder(&output)
			for _, message := range []string{"Transaction lookup requested", "service lookup"} {
				var entry map[string]any
				require.NoError(t, decoder.Decode(&entry))
				require.Equal(t, message, entry["message"])
				require.Equal(t, tt.caller, entry["caller"])
				require.Equal(t, "/indexer/tx/v1/txs/ABC", entry["indexer_path"])
				require.Equal(t, "ABC", entry["hash"])
				if tt.path == "" {
					require.NotContains(t, entry, "scan_api_path")
				} else {
					require.Equal(t, tt.path, entry["scan_api_path"])
				}
			}
		})
	}
}
