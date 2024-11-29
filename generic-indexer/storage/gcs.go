package storage

import (
	"context"
	"fmt"
	"io"

	gcs "cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

type GCSClient struct {
	*gcs.Client
}

func NewGCSClient(opts ...option.ClientOption) (*GCSClient, error) {
	c, err := gcs.NewClient(context.Background(), opts...)
	if err != nil {
		return nil, err
	}
	return &GCSClient{c}, nil
}

func (s *GCSClient) UploadFile(bucket string, objectPath string, message []byte) error {
	wc := s.Bucket(bucket).Object(objectPath).NewWriter(context.Background())
	_, err := wc.Write(message)
	if err != nil {
		return fmt.Errorf("failed to upload file %s, %v", objectPath, err)
	}
	if err = wc.Close(); err != nil {
		return fmt.Errorf("failed to close writer %s, %v", objectPath, err)
	}
	return nil
}

func (s *GCSClient) ReadFile(bucket string, objectPath string) ([]byte, error) {
	r, err := s.Bucket(bucket).Object(objectPath).NewReader(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get object %s, %v", objectPath, err)
	}
	defer r.Close()
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read object %s, %v", objectPath, err)
	}
	return content, err
}
