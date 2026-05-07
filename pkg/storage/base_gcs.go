package storage

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

type BaseGCSClient struct {
	client *storage.Client
}

func NewBaseGCSClient(opts ...option.ClientOption) (*BaseGCSClient, error) {
	c, err := storage.NewClient(context.Background(), opts...)
	if err != nil {
		return nil, err
	}
	return &BaseGCSClient{client: c}, nil
}

func (b *BaseGCSClient) UploadFile(bucket string, objectPath string, message []byte) error {
	wc := b.client.Bucket(bucket).Object(objectPath).NewWriter(context.Background())
	_, err := wc.Write(message)
	if err != nil {
		return fmt.Errorf("failed to upload file %s, %v", objectPath, err)
	}
	if err = wc.Close(); err != nil {
		return fmt.Errorf("failed to close writer %s, %v", objectPath, err)
	}
	return nil
}

func (b *BaseGCSClient) ReadFile(bucket string, objectPath string) ([]byte, error) {
	r, err := b.client.Bucket(bucket).Object(objectPath).NewReader(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get object %s, %v", objectPath, err)
	}
	defer r.Close()
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read object %s, %v", objectPath, err)
	}
	return content, nil
}
