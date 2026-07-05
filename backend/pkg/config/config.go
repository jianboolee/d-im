package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/viper"
)

// Config 全局配置结构体
type Config struct {
	App       AppConfig       `mapstructure:"app"`
	Server    ServerConfig    `mapstructure:"server"`
	MongoDB   MongoDBConfig   `mapstructure:"mongodb"`
	Redis     RedisConfig     `mapstructure:"redis"`
	NATS      NATSConfig      `mapstructure:"nats"`
	Snowflake SnowflakeConfig `mapstructure:"snowflake"`
	JWT       JWTConfig       `mapstructure:"jwt"`
	Log       LogConfig       `mapstructure:"log"`
}

type AppConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
	Env     string `mapstructure:"env"`
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

type SnowflakeConfig struct {
	WorkerID     int64 `mapstructure:"worker_id"`
	DatacenterID int64 `mapstructure:"datacenter_id"`
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
