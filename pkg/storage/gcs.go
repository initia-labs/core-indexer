package storage

import (
	"google.golang.org/api/option"
)

type GCSClient struct {
	*BaseGCSClient
}

func NewGCSClient(opts ...option.ClientOption) (*GCSClient, error) {
	baseClient, err := NewBaseGCSClient(opts...)
	if err != nil {
		return nil, err
	}
	return &GCSClient{baseClient}, nil
}
