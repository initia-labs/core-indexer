package storage

import (
	"bytes"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
)

type StorageClient interface {
	UploadFile(bucket string, objectPath string, message []byte) error
	ReadFile(bucket string, objectPath string) ([]byte, error)
}

type S3Client struct {
	*s3.Client
	*manager.Uploader
}

func NewS3Client(awsAccessKey string, awsSecretKey string) (*S3Client, error) {
	creds := credentials.NewStaticCredentialsProvider(awsAccessKey, awsSecretKey, "")
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithCredentialsProvider(creds), config.WithRegion("ap-southeast-1"))
	if err != nil {
		return nil, err
	}
	s3Client := s3.NewFromConfig(cfg)
	return &S3Client{s3Client, manager.NewUploader(s3Client)}, nil
}

func (s *S3Client) UploadFile(bucket string, objectPath string, message []byte) error {
	reader := bytes.NewReader(message)
	_, err := s.Upload(context.Background(), &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(objectPath),
		ContentType: aws.String("application/json"),
		Body:        reader,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file %s, %v", objectPath, err)
	}

	return nil
}

func (s *S3Client) ReadFile(bucket string, objectPath string) ([]byte, error) {
	resp, err := s.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectPath),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get object %s, %v", objectPath, err)
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	return buf.Bytes(), nil
}
