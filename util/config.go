package util

import (
	"time"

	"github.com/spf13/viper"
)

// Config stores all configuration of the application.
// The values are read by viper from a config file or environment variables.
type Config struct {
	DBDriver                string        `mapstructure:"DB_DRIVER"`
	DBSource                string        `mapstructure:"DB_SOURCE"`
	HTTPServerAddress       string        `mapstructure:"HTTP_SERVER_ADDRESS"`
	TokenSymmetricKey       string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuration     time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration    time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
	WSReadBufferSize        int           `mapstructure:"WS_READ_BUFFER_SIZE"`
	WSWriteBufferSize       int           `mapstructure:"WS_WRITE_BUFFER_SIZE"`
	WSMaxConnectionsPerUser int           `mapstructure:"WS_MAX_CONNECTIONS_PER_USER"`
	WSPingInterval          time.Duration `mapstructure:"WS_PING_INTERVAL"`
	WSPongTimeout           time.Duration `mapstructure:"WS_PONG_TIMEOUT"`
	// File storage configuration
	FileStoragePath         string `mapstructure:"FILE_STORAGE_PATH"`
	FileMaxSize             int64  `mapstructure:"FILE_MAX_SIZE"`
	FileAllowedTypes        string `mapstructure:"FILE_ALLOWED_TYPES"`
	EnableFileDeduplication bool   `mapstructure:"ENABLE_FILE_DEDUPLICATION"`
	EnableThumbnails        bool   `mapstructure:"ENABLE_THUMBNAILS"`
	// AWS S3 configuration (optional)
	AWSS3Bucket  string `mapstructure:"AWS_S3_BUCKET"`
	AWSRegion    string `mapstructure:"AWS_REGION"`
	UseS3Storage bool   `mapstructure:"USE_S3_STORAGE"`
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	// Set default values for WebSocket configuration
	viper.SetDefault("WS_READ_BUFFER_SIZE", 1024)
	viper.SetDefault("WS_WRITE_BUFFER_SIZE", 1024)
	viper.SetDefault("WS_MAX_CONNECTIONS_PER_USER", 5)
	viper.SetDefault("WS_PING_INTERVAL", "54s")
	viper.SetDefault("WS_PONG_TIMEOUT", "60s")

	// Set default values for file storage configuration
	viper.SetDefault("FILE_STORAGE_PATH", "./uploads")
	viper.SetDefault("FILE_MAX_SIZE", 10485760) // 10MB
	viper.SetDefault("FILE_ALLOWED_TYPES", "image/jpeg,image/png,image/gif,image/webp,application/pdf,text/plain,application/zip")
	viper.SetDefault("ENABLE_FILE_DEDUPLICATION", true)
	viper.SetDefault("ENABLE_THUMBNAILS", true)
	viper.SetDefault("USE_S3_STORAGE", false)

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}
