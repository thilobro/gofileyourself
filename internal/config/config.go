package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	HistoryLen int    `yaml:"history_len"`
	ThemePath  string `yaml:"theme_path"`
}

func NewConfig(configPath *string) (*Config, error) {
	config := &Config{
		HistoryLen: 50,
		ThemePath:  os.Getenv("HOME") + "/.gofileyourself_theme.yaml",
	}

	if configFile, err := os.ReadFile(*configPath); err == nil {
		var tempConfig Config
		if err := yaml.Unmarshal(configFile, &tempConfig); err != nil {
			log.Fatal(err)
			panic(err)
		}
		// Override defaults with config file values
		if tempConfig.HistoryLen != 0 {
			config.HistoryLen = tempConfig.HistoryLen
		}
		if tempConfig.ThemePath != "" {
			config.ThemePath = tempConfig.ThemePath
		}
	}
	return config, nil
}
