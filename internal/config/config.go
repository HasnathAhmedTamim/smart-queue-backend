package config

import (
	"flag"
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config holds the application configuration loaded from a YAML file and environment variables.
type Config struct {
	Env     string  `yaml:"env" env:"ENV" env-default:"local"`
	HTTP    HTTP    `yaml:"http"`
	Storage Storage `yaml:"storage"`
	Queue   Queue   `yaml:"queue"`
	CORS    CORS    `yaml:"cors"`
}

// HTTP contains the configuration for the HTTP server.
type HTTP struct {
	Server HTTPServer `yaml:"server"`
}

// HTTPServer contains the configuration for the HTTP server.
type HTTPServer struct {
	Address string `yaml:"address" env:"HTTP_ADDRESS" env-default:":8080"`
}

// Storage contains the configuration for the storage system.
type Storage struct {
	Path string `yaml:"path" env:"STORAGE_PATH" env-required:"true"`
}

// Queue contains the configuration for the queue system.
type Queue struct {
	AvgServiceMinutes int    `yaml:"avg_service_minutes" env:"AVG_SERVICE_MINUTES" env-default:"3"`
	AdminKey          string `yaml:"admin_key" env:"ADMIN_KEY" env-required:"true"`
}

type CORS struct {
	AllowedOrigin string `yaml:"allowed_origin" env:"CORS_ALLOWED_ORIGIN" env-default:"http://localhost:3000"`
}

// Load reads the configuration from a YAML file specified by the CONFIG_PATH environment variable or the -config flag, and returns a Config struct.
func Load() (*Config, error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		p := flag.String("config", "", "path to config file")
		flag.Parse()
		configPath = *p
	}

	if configPath == "" {
		return nil, fmt.Errorf("config path not provided (set CONFIG_PATH or use -config)")
	}
	if _, err := os.Stat(configPath); err != nil {
		return nil, fmt.Errorf("config file error: %w", err)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	return &cfg, nil
}

// MustLoad is a helper function that calls Load and panics if there is an error, returning the loaded Config.
func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(err)
	}
	return cfg
}
