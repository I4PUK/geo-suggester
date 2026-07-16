package config

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/jinzhu/configor"
)

type Config struct {
	HotelContentEndpoint     string `env:"HOTEL_CONTENT_ENDPOINT"`
	ElasticSearchTemplateDIR string `env:"ELASTIC_SEARCH_TEMPLATE_DIR"`
	GeoMemoryClient          bool   `env:"GEO_MEMORY_CLIENT"`
	TopHotelsInRegions       string `env:"TOP_HOTEL_IN_REGIONS"`
	Redis                    RedisConfig
}

// RedisConfig stores redis configuration
type RedisConfig struct {
	URL      string `env:"REDIS_URL"`
	Password string `env:"REDIS_PASSWORD"`
}

// Validate implements the validation interface.
func (c *Config) Validate() error {
	return validation.ValidateStruct(c,
		validation.Field(&c.HotelContentEndpoint, validation.Required),
		validation.Field(&c.ElasticSearchTemplateDIR, validation.Required),
		validation.Field(&c.Redis),
	)
}

// Validate implements the validation interface.
func (c *RedisConfig) Validate() error {
	return validation.ValidateStruct(c,
		validation.Field(&c.URL, validation.Required),
	)
}

// ReadConfig reads config file and returns new config instance
func ReadConfig() (*Config, error) {
	var config Config
	err := configor.Load(&config)
	if err != nil {
		return nil, err
	}
	if err = config.Validate(); err != nil {
		return nil, err
	}
	return &config, nil
}
