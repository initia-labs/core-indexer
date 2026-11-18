package main

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/rs/zerolog"
)

type uploadStorage struct {
	data                                         []byte
	readBucket, readPath, writeBucket, writePath string
	uploadErr                                    error
}

func (s *uploadStorage) ReadFile(bucket, path string) ([]byte, error) {
	s.readBucket, s.readPath = bucket, path
	return s.data, nil
}
func (s *uploadStorage) UploadFile(bucket, path string, data []byte) error {
	s.writeBucket, s.writePath, s.data = bucket, path, bytes.Clone(data)
	return s.uploadErr
}

func TestProcessPubSubResponse(t *testing.T) {
	log := zerolog.Nop()
	oldLogger := logger
	logger = &log
	t.Cleanup(func() { logger = oldLogger })
	for _, claimCheck := range []bool{false, true} {
		t.Run(map[bool]string{false: "inline", true: "claim-check"}[claimCheck], func(t *testing.T) {
			payload := []byte(`{"txhash":"ABC"}`)
			storage := &uploadStorage{data: payload}
			uploader := &TxResponseUploader{storageClient: storage, config: &Config{LCDTXResponseBucket: "tx-responses", ClaimCheckBucket: "claim-checks"}}
			msg := &mq.Message{Data: payload, Attributes: map[string]string{"key": "NEW_LCD_TX_RESPONSE_abc", "tx_hash": "abc", "height": "7"}}
			if claimCheck {
				msg.Attributes["key"] = "NEW_LCD_TX_RESPONSE_CLAIM_CHECK_abc"
				msg.Data = []byte(`{"object_path":"abc_7"}`)
			}
			if err := uploader.process(context.Background(), msg); err != nil {
				t.Fatal(err)
			}
			if storage.writeBucket != "tx-responses" || storage.writePath != "ABC/7" || !bytes.Equal(storage.data, payload) {
				t.Fatal("wrong archived response")
			}
			if claimCheck && (storage.readBucket != "claim-checks" || storage.readPath != "abc_7") {
				t.Fatal("wrong claim check location")
			}
			storage.uploadErr = errors.New("upload unavailable")
			if err := uploader.process(context.Background(), msg); !errors.Is(err, storage.uploadErr) {
				t.Fatalf("lost upload error: %v", err)
			}
		})
	}
}
