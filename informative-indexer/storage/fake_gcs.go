package storage

import (
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"google.golang.org/api/option"
	"io"
)

type GCSFakeClient struct {
	*storage.Client
}

func NewGCSFakeClient(opts ...option.ClientOption) (*GCSFakeClient, error) {
	c, err := storage.NewClient(context.TODO(), option.WithoutAuthentication(), option.WithEndpoint("http://storage:9184/storage/v1/"))
	if err != nil {
		return nil, err
	}
	return &GCSFakeClient{c}, nil
}

func (s *GCSFakeClient) UploadFile(bucket string, objectPath string, message []byte) error {
	wc := s.Bucket(bucket).Object(objectPath).NewWriter(context.TODO())
	_, err := wc.Write(message)
	if err != nil {
		return fmt.Errorf("failed to upload file %s, %v", objectPath, err)

	}
	if err = wc.Close(); err != nil {
		return fmt.Errorf("failed to close writer %s, %v", objectPath, err)
	}
	return nil
}

func (s *GCSFakeClient) ReadFile(bucket string, objectPath string) ([]byte, error) {
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
