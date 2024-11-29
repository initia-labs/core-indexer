package storage

type StorageClient interface {
	UploadFile(bucket string, objectPath string, message []byte) error
	ReadFile(bucket string, objectPath string) ([]byte, error)
}
