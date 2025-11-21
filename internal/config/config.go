// Package config пакет с инициализацией конфига
package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config основная структура конфигурации
type Config struct {
	App      AppConfig      `yaml:"app"`
	Log      LogConfig      `yaml:"log"`
	MQTT     MQTTConfig     `yaml:"mqtt"`
	HTTP     HTTPConfig     `yaml:"http"`
	DataBase DataBaseConfig `yaml:"database"`
}

type AppConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Output string `yaml:"output,omitempty"`
}

type MQTTConfig struct {
	Broker           string `yaml:"broker"`
	ClientID         string `yaml:"client_id"`
	Username         string `yaml:"username,omitempty"`
	Password         string `yaml:"password,omitempty"`
	New_divice_topic string `yaml:"new_divice_topic"`
	QoS              int    `yaml:"qos"`
	CleanSession     bool   `yaml:"clean_session"`
	KeepAlive        int    `yaml:"keep_alive"`
	ConnectTimeout   int    `yaml:"connect_timeout"`
}

type HTTPConfig struct {
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	Address string
}

type DataBaseConfig struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	Name        string `yaml:"name"`
	User        string `yaml:"user"`
	Password    string `yaml:"password"`
	DataBaseDSN string
}

func LoadConfig(configPath string) (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	config := &Config{}
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(config); err != nil {
		return nil, err
	}

	overrideFromEnv(config)
	config.HTTP.Address = fmt.Sprintf("%s:%d", config.HTTP.Host, config.HTTP.Port)
	config.DataBase.DataBaseDSN = fmt.Sprintf("%s://%s:%s@%s:%d", "postgres", config.DataBase.User, config.DataBase.Password, config.DataBase.Host, config.DataBase.Port)

	return config, nil
}

func overrideFromEnv(config *Config) {
	// надо описать для докера
}
