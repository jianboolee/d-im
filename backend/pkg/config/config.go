package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/viper"
)

// Config 全局配置结构体
type Config struct {
	App     AppConfig     `mapstructure:"app"`
	Auth    AuthConfig    `mapstructure:"auth"`
	Server  ServerConfig  `mapstructure:"server"`
	MongoDB MongoDBConfig `mapstructure:"mongodb"`
	Redis   RedisConfig   `mapstructure:"redis"`
	NATS    NATSConfig    `mapstructure:"nats"`
	JWT     JWTConfig     `mapstructure:"jwt"`
	Log     LogConfig     `mapstructure:"log"`
	Storage StorageConfig `mapstructure:"storage"`
}

type AppConfig struct {
	Name        string `mapstructure:"name"`
	Version     string `mapstructure:"version"`
	Env         string `mapstructure:"env"`
	FrontendURL string `mapstructure:"frontend_url"` // 前端地址：http://localhost:5173
}

type AuthConfig struct {
	SuperPassword string `mapstructure:"super_password"`
}

type ServerConfig struct {
	Gateway   GatewayConfig   `mapstructure:"gateway"`
	Connector ConnectorConfig `mapstructure:"connector"`
}

type GatewayConfig struct {
	HTTPPort int `mapstructure:"http_port"`
	GRPCPort int `mapstructure:"grpc_port"`
}

type ConnectorConfig struct {
	WSPort  int `mapstructure:"ws_port"`
	TCPPort int `mapstructure:"tcp_port"`
}

type MongoDBConfig struct {
	URI      string `mapstructure:"uri"`
	Database string `mapstructure:"database"`
	PoolSize uint64 `mapstructure:"pool_size"`
	Timeout  int    `mapstructure:"timeout"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

type NATSConfig struct {
	URL            string       `mapstructure:"url"`
	User           string       `mapstructure:"user"`
	Password       string       `mapstructure:"password"`
	PublishTimeout string       `mapstructure:"publish_timeout"`
	Subjects       NATSSubjects `mapstructure:"subjects"`
}

type NATSSubjects struct {
	MessageSend  string `mapstructure:"message_send"`
	MessagePush  string `mapstructure:"message_push"`
	MessageEvent string `mapstructure:"message_event"`
	UserCreated  string `mapstructure:"user_created"`
	UserUpdated  string `mapstructure:"user_updated"`
	UserStatus   string `mapstructure:"user_status"`
	UserDeleted  string `mapstructure:"user_deleted"`
}

type JWTConfig struct {
	Secret        string `mapstructure:"secret"`
	AccessExpire  string `mapstructure:"access_expire"`  // access_token 过期: 15m
	RefreshExpire string `mapstructure:"refresh_expire"` // refresh_token 过期: 168h (7d)
	TicketExpire  string `mapstructure:"ticket_expire"`  // 一次性 ticket 过期: 5m
	APIKey        string `mapstructure:"api_key"`        // 业务系统调用内部接口的 API Key
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

type StorageConfig struct {
	Provider      string             `mapstructure:"provider"`
	PublicBaseURL string             `mapstructure:"public_base_url"`
	MaxImageSize  int64              `mapstructure:"max_image_size"`
	Local         LocalStorageConfig `mapstructure:"local"`
	AliyunOSS     AliyunOSSConfig    `mapstructure:"aliyun_oss"`
	Qiniu         QiniuStorageConfig `mapstructure:"qiniu"`
}

type LocalStorageConfig struct {
	RootDir   string `mapstructure:"root_dir"`
	URLPrefix string `mapstructure:"url_prefix"`
}

type AliyunOSSConfig struct {
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	Bucket          string `mapstructure:"bucket"`
	Directory       string `mapstructure:"directory"`
	PublicBaseURL   string `mapstructure:"public_base_url"`
}

type QiniuStorageConfig struct {
	AccessKey     string `mapstructure:"access_key"`
	SecretKey     string `mapstructure:"secret_key"`
	Bucket        string `mapstructure:"bucket"`
	Region        string `mapstructure:"region"`
	PublicBaseURL string `mapstructure:"public_base_url"`
}

// Load 加载配置文件
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// 设置配置文件路径
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	// 支持环境变量覆盖：MONGODB_URI 覆盖 mongodb.uri
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	_ = v.BindEnv("auth.super_password", "IM_SUPER_PASSWORD")
	_ = v.BindEnv("storage.provider", "STORAGE_PROVIDER")
	_ = v.BindEnv("storage.public_base_url", "STORAGE_PUBLIC_BASE_URL")
	_ = v.BindEnv("storage.max_image_size", "STORAGE_MAX_IMAGE_SIZE")
	_ = v.BindEnv("storage.local.root_dir", "STORAGE_LOCAL_ROOT_DIR")
	_ = v.BindEnv("storage.local.url_prefix", "STORAGE_LOCAL_URL_PREFIX")
	_ = v.BindEnv("storage.aliyun_oss.endpoint", "STORAGE_ALIYUN_OSS_ENDPOINT")
	_ = v.BindEnv("storage.aliyun_oss.access_key_id", "STORAGE_ALIYUN_OSS_ACCESS_KEY_ID")
	_ = v.BindEnv("storage.aliyun_oss.access_key_secret", "STORAGE_ALIYUN_OSS_ACCESS_KEY_SECRET")
	_ = v.BindEnv("storage.aliyun_oss.bucket", "STORAGE_ALIYUN_OSS_BUCKET")
	_ = v.BindEnv("storage.aliyun_oss.directory", "STORAGE_ALIYUN_OSS_DIRECTORY")
	_ = v.BindEnv("storage.aliyun_oss.public_base_url", "STORAGE_ALIYUN_OSS_PUBLIC_BASE_URL")
	_ = v.BindEnv("storage.qiniu.access_key", "STORAGE_QINIU_ACCESS_KEY")
	_ = v.BindEnv("storage.qiniu.secret_key", "STORAGE_QINIU_SECRET_KEY")
	_ = v.BindEnv("storage.qiniu.bucket", "STORAGE_QINIU_BUCKET")
	_ = v.BindEnv("storage.qiniu.region", "STORAGE_QINIU_REGION")
	_ = v.BindEnv("storage.qiniu.public_base_url", "STORAGE_QINIU_PUBLIC_BASE_URL")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	// 强制环境变量覆盖：逐个 key 重新 Get（触发 AutomaticEnv 生效）
	for _, key := range v.AllKeys() {
		v.Set(key, v.Get(key))
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	log.Printf("[config] loaded: %s (env=%s)", v.ConfigFileUsed(), cfg.App.Env)
	return &cfg, nil
}
