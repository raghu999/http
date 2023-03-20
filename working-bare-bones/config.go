package httplogreceiver

import "go.uber.org/config"

type Config struct {
	Endpoint string `mapstructure:"endpoint"`
}

func (*Config) Validate() error {
	return nil
}

func (*Config) Sanitize() {
}

func DefaultConfig() *Config {
	return &Config{
		Endpoint: defaultEndpoint,
	}
}

func NewConfig(cfg config.Provider) (*Config, error) {
	config := DefaultConfig()
	err := cfg.Get("endpoint").Populate(&config.Endpoint)
	if err != nil {
		return nil, err
	}
	return config, nil
}
