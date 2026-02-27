package config

import "github.com/kelseyhightower/envconfig"

// AppConfig holds general application settings.
type AppConfig struct {
	Port string `envconfig:"PORT" default:"3000"`
	Env  string `envconfig:"ENV" default:"development"`
}

// DatabaseConfig holds the primary database connection settings.
type DatabaseConfig struct {
	Host     string `envconfig:"HOST" default:"localhost"`
	Port     int    `envconfig:"PORT" default:"5432"`
	Name     string `envconfig:"NAME" default:"payment"`
	User     string `envconfig:"USER" default:"postgres"`
	Password string `envconfig:"PASSWORD"`
	SSLMode  string `envconfig:"SSLMODE" default:"disable"`
}

// ARIDBConfig holds the ARI (availability & rates) database connection settings.
type ARIDBConfig struct {
	Host     string `envconfig:"HOST" default:"localhost"`
	Port     int    `envconfig:"PORT" default:"5432"`
	Name     string `envconfig:"NAME" default:"ari"`
	User     string `envconfig:"USER" default:"postgres"`
	Password string `envconfig:"PASSWORD"`
	SSLMode  string `envconfig:"SSLMODE" default:"disable"`
}

// RedisConfig holds the Redis connection settings.
type RedisConfig struct {
	Host     string `envconfig:"HOST" default:"localhost"`
	Port     int    `envconfig:"PORT" default:"6379"`
	Password string `envconfig:"PASSWORD"`
	DB       int    `envconfig:"DB" default:"0"`
}

// VaulteraConfig holds the Vaultera PCI proxy settings.
type VaulteraConfig struct {
	APIKey  string `envconfig:"API_KEY"`
	BaseURL string `envconfig:"BASE_URL" default:"https://pci.vaultera.co/api/v1"`
}

// Config aggregates all service configuration.
type Config struct {
	App      AppConfig
	Database DatabaseConfig
	ARIDB    ARIDBConfig
	Redis    RedisConfig
	Vaultera VaulteraConfig
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{}

	if err := envconfig.Process("APP", &cfg.App); err != nil {
		return nil, err
	}
	if err := envconfig.Process("DATABASE", &cfg.Database); err != nil {
		return nil, err
	}
	if err := envconfig.Process("ARI_DB", &cfg.ARIDB); err != nil {
		return nil, err
	}
	if err := envconfig.Process("REDIS", &cfg.Redis); err != nil {
		return nil, err
	}
	if err := envconfig.Process("VAULTERA", &cfg.Vaultera); err != nil {
		return nil, err
	}

	return cfg, nil
}
