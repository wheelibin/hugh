package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
)

type Schedule struct {
	Type           string   `json:"type"`
	LightIds       []string `json:"lightIds"`
	Rooms          []string `json:"rooms"`
	Zones          []string `json:"zones"`
	SunriseMin     string   `json:"sunriseMin"`
	SunriseMax     string   `json:"sunriseMax"`
	SunsetMin      string   `json:"sunsetMin"`
	SunsetMax      string   `json:"sunsetMax"`
	DefaultPattern struct {
		Time        string `json:"time"`
		Temperature int    `json:"temperature"`
		Brightness  int    `json:"brightness"`
	} `json:"defaultPattern"`
	DayPattern []struct {
		Time        string `json:"time"`
		Temperature int    `json:"temperature"`
		Brightness  int    `json:"brightness"`
	} `json:"dayPattern"`
}

type Config struct {
	BridgeIP    string     `json:"bridgeIp"`
	HueAppKey   string     `json:"hueApplicationKey"`
	GeoLocation string     `json:"geoLocation"`
	Schedules   []Schedule `json:"schedules"`
}

var AppConfig Config

func ReadConfig() *Config {

	configFilename := "/home/jon/dev/hugh/config.json"

	config := Config{}
	fileBytes, _ := os.ReadFile(configFilename)
	_ = json.Unmarshal(fileBytes, &config)

	AppConfig = config

	return &config
}

func InitialiseConfig() {
	viper.SetConfigName("config")              // name of config file (without extension)
	viper.SetConfigType("json")                // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath("/etc/hugh/")          // path to look for the config file in
	viper.AddConfigPath("$HOME/.config/hugh/") // call multiple times to add many search paths
	viper.AddConfigPath(".")                   // optionally look for config in the working directory
	err := viper.ReadInConfig()                // Find and read the config file
	if err != nil {                            // Handle errors reading the config file
		log.Error(err)
		panic(fmt.Errorf("fatal error config file: %w", err))

	}
}
