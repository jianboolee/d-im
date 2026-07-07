package upload

// StorageConfig 阿里云 OSS 存储配置
type StorageConfig struct {
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	BucketName      string
	CustomDomain    string
	Directory       string
}
