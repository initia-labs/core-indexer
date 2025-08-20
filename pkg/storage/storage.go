package storage

type Client interface {
	UploadFile(bucket string, objectPath string, message []byte) error
	ReadFile(bucket string, objectPath string) ([]byte, error)
}
