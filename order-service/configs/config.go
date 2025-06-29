package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type LogFileProperties struct {
	Dirname     string `json:"dirname" yaml:"dirname"`
	Filename    string `json:"filename" yaml:"filename"`
	DatePattern string `json:"date-pattern" yaml:"date-pattern"`
	Extension   string `json:"extension" yaml:"extension"`
}

type LogConfig struct {
	Level             string            `json:"level" yaml:"level"`
	EnableFileLogging bool              `json:"enable-file-logging" yaml:"enable-file-logging"`
	LogFileProperties LogFileProperties `json:"log-file-properties" yaml:"log-file-properties"`
}

type Log struct {
	App     LogConfig `json:"app" yaml:"app"`
	Detail  LogConfig `json:"detail" yaml:"detail"`
	Summary LogConfig `json:"summary" yaml:"summary"`
}

type App struct {
	Name           string `json:"name" yaml:"name"`
	ComponentName  string `json:"component-name" yaml:"component-name"`
	Description    string `json:"description" yaml:"description"`
	Version        string `json:"version" yaml:"version"`
	BaseApiVersion string `json:"baseApiVersion" yaml:"baseApiVersion"`
	SchemaVersion  string `json:"schemaVersion" yaml:"schemaVersion"`
}

type Server struct {
	AppPort string `json:"app_port" yaml:"app_port"`
	AppHost string `json:"app_host" yaml:"app_host"`
	Https   bool   `json:"https" yaml:"https"`
	Cert    string `json:"cert" yaml:"cert"`
	Key     string `json:"key" yaml:"key"`
}

type TLSKafkaConfig struct {
	CertFile           string `json:"certFile" yaml:"certFile"`
	KeyFile            string `json:"keyFile" yaml:"keyFile"`
	CACertFile         string `json:"caCertFile" yaml:"caCertFile"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify" yaml:"insecureSkipVerify"`
}

type KafkaConfig struct {
	Broker           string         `json:"broker" yaml:"broker"`
	Partition        int            `json:"partition" yaml:"partition"`
	ConsumerGroupID  string         `json:"consumerGroupID" yaml:"consumerGroupID"`
	OffSet           int            `json:"offSet" yaml:"offSet"`
	BatchSize        int            `json:"batchSize" yaml:"batchSize"`
	BatchBytes       int            `json:"batchBytes" yaml:"batchBytes"`
	BatchTimeout     int            `json:"batchTimeout" yaml:"batchTimeout"`
	RetryTimeout     time.Duration  `json:"retryTimeout" yaml:"retryTimeout"`
	SASLMechanism    string         `json:"SASLMechanism" yaml:"SASLMechanism"`
	SASLUser         string         `json:"SASLUser" yaml:"SASLUser"`
	SASLPassword     string         `json:"SASLPassword" yaml:"SASLPassword"`
	SecurityProtocol string         `json:"securityProtocol" yaml:"securityProtocol"`
	AutoCreateTopic  bool           `json:"autoCreateTopic" yaml:"autoCreateTopic"`
	TLS              TLSKafkaConfig `json:"TLS" yaml:"TLS"`
}

type Config struct {
	App        App         `json:"app" yaml:"app"`
	Log        Log         `json:"log" yaml:"log"`
	Server     Server      `json:"server" yaml:"server"`
	Kafka      KafkaConfig `json:"kafka" yaml:"kafka"`
	TracerHost string      `json:"tracer_host" yaml:"tracer_host"`
}

type IConfig interface {
	Get(string) string
	GetOrDefault(string, string) string
}

func NewConfig(conf ...Config) *Config {
	cfg := Config{}
	if len(conf) > 0 {
		cfg = conf[0]
	}
	return &cfg
}

func (c *Config) LoadEnv(configFolder string) *Config {
	newCfg := NewEnvFile(configFolder, false)
	*c = *newCfg
	return c
}

func (c *Config) LoadEnvFile(filepath string) *Config {
	newCfg := NewEnvFile(filepath, true)
	*c = *newCfg
	return c
}

func (*Config) Get(key string) string {
	return os.Getenv(key)
}

func (*Config) GetOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}

	return defaultValue
}

func (cfg *Config) LoadConfigJson(filepath string) *Config {
	// Check file existence
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		file, createErr := os.Create(filepath)
		if createErr != nil {
			panic(createErr)
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ") // prettify
		if err := encoder.Encode(cfg); err != nil {
			panic(fmt.Errorf("failed to write default config JSON file: %w", err))
		}
		log.Printf("Created default config JSON file at %s", filepath)
	}

	// Open the JSON config file
	file, err := os.Open(filepath)
	if err != nil {
		panic(fmt.Errorf("failed to open config JSON file: %w", err))
	}
	defer file.Close()

	// Decode the JSON into the Config struct
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(cfg); err != nil {
		panic(fmt.Errorf("failed to decode config JSON file: %w", err))
	}

	return cfg
}

func (cfg *Config) LoadConfigYml(filepath string) *Config {

	// Check if file exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		fmt.Println("YAML config file not found. Creating default config file...")

		// Define default config

		// Write default YAML to file
		file, createErr := os.Create(filepath)
		if createErr != nil {
			log.Fatalf("failed to create YAML config file: %v", createErr)
		}
		defer file.Close()

		encoder := yaml.NewEncoder(file)
		defer encoder.Close()
		if err := encoder.Encode(&cfg); err != nil {
			log.Fatalf("failed to write default YAML config file: %v", err)
		}

		return cfg
	}

	// File exists — open and decode
	file, err := os.Open(filepath)
	if err != nil {
		log.Fatalf("failed to open YAML config file: %v", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		log.Fatalf("failed to decode YAML config file: %v", err)
	}

	return cfg
}
