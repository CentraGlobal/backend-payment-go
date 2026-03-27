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

// PCIBookingConfig holds the PCI Booking settings.
type PCIBookingConfig struct {
	APIKey  string `envconfig:"API_KEY"`
	BaseURL string `envconfig:"BASE_URL" default:"https://service.pcibooking.net"`
}

// ProcessorConfig holds the active processor selection.
type ProcessorConfig struct {
	Name string `envconfig:"NAME" default:"vaultera"` // "vaultera", "pcibooking", or "pci_booking_upg"
}

// AuthConfig holds the server-to-server shared secret auth settings.
type AuthConfig struct {
	SharedSecret string `envconfig:"SHARED_SECRET"`
	HeaderName   string `envconfig:"HEADER_NAME" default:"X-Payment-Service-Auth"`
	Require      bool   `envconfig:"REQUIRE" default:"true"`
}

// Config aggregates all service configuration.
type Config struct {
	App        AppConfig
	Database   DatabaseConfig
	ARIDB      ARIDBConfig
	Redis      RedisConfig
	Vaultera   VaulteraConfig
	PCIBooking PCIBookingConfig
	Processor  ProcessorConfig
	Auth       AuthConfig
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
	if err := envconfig.Process("PCI_BOOKING", &cfg.PCIBooking); err != nil {
		return nil, err
	}
	if err := envconfig.Process("PROCESSOR", &cfg.Processor); err != nil {
		return nil, err
	}
	if err := envconfig.Process("AUTH", &cfg.Auth); err != nil {
		return nil, err
	}

	return cfg, nil
}
