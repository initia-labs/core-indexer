package storage

import (
	"google.golang.org/api/option"
)

type GCSFakeClient struct {
	*BaseGCSClient
}

func NewGCSFakeClient() (*GCSFakeClient, error) {
	baseClient, err := NewBaseGCSClient(option.WithoutAuthentication(), option.WithEndpoint("http://storage:9184/storage/v1/"))
	if err != nil {
		return nil, err
	}
	return &GCSFakeClient{baseClient}, nil
}
