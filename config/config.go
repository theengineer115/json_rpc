// config/config.go
package config

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	ServerPort  string `mapstructure:"server_port"`
	KafkaBroker string `mapstructure:"kafka_broker"`
	KafkaTopic  string `mapstructure:"kafka_topic"`
	RedisURL    string `mapstructure:"redis_url"`
	InfuraURL   string `mapstructure:"infura_url"`
}

func LoadConfig() (*Config, error) {
    viper.SetConfigFile(filepath.Join(".", "config.yaml"))

    err := viper.ReadInConfig()
    if err != nil {
        return nil, fmt.Errorf("error reading config file: %w", err)
    }

    var cfg Config
    err = viper.Unmarshal(&cfg)
    if err != nil {
        return nil, fmt.Errorf("error unmarshaling config: %w", err)
    }

    return &cfg, nil
}

