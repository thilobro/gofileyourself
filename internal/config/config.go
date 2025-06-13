package config

import (
	"log"
	"os"

	"github.com/creasty/defaults"

	"gopkg.in/yaml.v3"
)

type Config struct {
	HistoryLen int `default:"50" yaml:"history_len"`
}

func NewConfig(configPath *string) (*Config, error) {
	configFile, err := os.ReadFile(*configPath)
	if err != nil {
		return &Config{HistoryLen: 50}, nil
	}
	var config Config
	if err := defaults.Set(&config); err != nil {
		log.Fatal(err)
		panic(err)
	}
	if err := yaml.Unmarshal(configFile, &config); err != nil {
		log.Fatal(err)
		panic(err)
	}
	log.Print(config)
	return &config, nil
}
