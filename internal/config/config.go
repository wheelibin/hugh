package config

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
)

func InitialiseConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.config/hugh/")
	viper.AddConfigPath("/etc/hugh/")
	viper.AddConfigPath(".")    // optionally look for config in the working directory
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		log.Error(err)
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
}
