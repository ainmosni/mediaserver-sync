package config

import (
	"github.com/spf13/viper"
)

const (
	CONFIG_NAME = "config"
)

var CONFIG_PATHS = [...]string{
	".",
	"/etc/mediasync",
	"~/.config/mediasync",
}

func GetConfig() (*Configuration, error) {
	viper.SetConfigName(CONFIG_NAME)
	for _, cp := range CONFIG_PATHS {
		viper.AddConfigPath(cp)
	}

	err := viper.ReadInConfig()
	if err != nil {
		return &Configuration{}, err
	}

	var c Configuration
	err = viper.Unmarshal(&c)

	if err != nil {
		return &Configuration{}, err
	}

	return &c, nil
}
