package config

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const (
	defaultFileName         = "/.env"
	defaultOverrideFileName = "/.local.env"
)

type EnvLoader struct {
	autoCreateFile bool
}

func NewEnvFile(configFolder string, autoCreateFile bool) *Config {
	conf := &EnvLoader{autoCreateFile: autoCreateFile}

	return conf.read(configFolder)
}

func (e *EnvLoader) read(folder string) *Config {

	// .env files are expected to be in the configs directory
	var (
		defaultFile  = folder + defaultFileName
		overrideFile = folder + defaultOverrideFileName
		env          = e.Get("APP_ENV")
	)

	if strings.HasSuffix(folder, ".env") {
		// If folder ends with .env, it is a file path, not a directory
		defaultFile = folder
		overrideFile = folder
	}

	errFmt := "Failed to load config from file: %v, Err: %v"

	err := godotenv.Load(defaultFile)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Fatalf(errFmt, defaultFile, err)
		}

		log.Printf(errFmt, defaultFile, err)
	} else {
		log.Printf("Loaded config from file: %v", defaultFile)
	}

	if env != "" {
		// If 'APP_ENV' is set to x, then GoFr will read '.env' from configs directory, and then it will be overwritten
		// by configs present in file '.x.env'
		overrideFile = fmt.Sprintf("%s/.%s.env", folder, env)
	}

	writeFileDotEnv := false

	err = godotenv.Overload(overrideFile)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Fatalf(errFmt, overrideFile, err)
		}
		writeFileDotEnv = true
	} else {
		log.Printf("Loaded config from file: %v", overrideFile)
	}

	// Reload system environment variables to ensure they override any previously loaded values
	for _, envVar := range os.Environ() {
		key, value, found := strings.Cut(envVar, "=")
		if found {
			os.Setenv(key, value)
		}
	}

	cfg := &Config{
		App: App{
			Name:           e.GetOrDefault("APP_NAME", ""),
			ComponentName:  e.GetOrDefault("APP_COMPONENT_NAME", ""),
			Description:    e.GetOrDefault("APP_DESCRIPTION", ""),
			Version:        e.GetOrDefault("APP_VERSION", "1.0.0"),
			BaseApiVersion: e.GetOrDefault("APP_BASE_API_VERSION", "v1"),
			SchemaVersion:  e.GetOrDefault("APP_SCHEMA_VERSION", "1.0"),
		},
		Log: Log{
			Detail: LogConfig{
				Level:             e.GetOrDefault("LOG_DETAIL_LEVEL", "debug"),
				EnableFileLogging: parseBool("LOG_DETAIL_ENABLE_FILE_LOGGING", true),
				LogFileProperties: LogFileProperties{
					Dirname:     e.GetOrDefault("LOG_DETAIL_DIRNAME", "./logs/detail"),
					Filename:    e.GetOrDefault("LOG_DETAIL_FILENAME", "detail-%DATE%"),
					DatePattern: e.GetOrDefault("LOG_DETAIL_DATE_PATTERN", "YYYY-MM-DD-HH"),
					Extension:   e.GetOrDefault("LOG_DETAIL_EXTENSION", ".log"),
				},
			},
			Summary: LogConfig{
				Level:             e.GetOrDefault("LOG_SUMMARY_LEVEL", "info"),
				EnableFileLogging: parseBool("LOG_SUMMARY_ENABLE_FILE_LOGGING", true),
				LogFileProperties: LogFileProperties{
					Dirname:     e.GetOrDefault("LOG_SUMMARY_DIRNAME", "./logs/summary"),
					Filename:    e.GetOrDefault("LOG_SUMMARY_FILENAME", "summary-%DATE%"),
					DatePattern: e.GetOrDefault("LOG_SUMMARY_DATE_PATTERN", "YYYY-MM-DD-HH"),
					Extension:   e.GetOrDefault("LOG_SUMMARY_EXTENSION", ".log"),
				},
			},
		},
		Server: Server{
			AppPort: e.GetOrDefault("SERVER_APP_PORT", "8080"),
			AppHost: e.GetOrDefault("SERVER_APP_HOST", "localhost"),
			Https:   parseBool("SERVER_HTTPS", false),
			Cert:    e.GetOrDefault("SERVER_CERT", "./cert.pem"),
			Key:     e.GetOrDefault("SERVER_KEY", "./key.pem"),
		},
		Kafka: KafkaConfig{
			Broker:          e.GetOrDefault("KAFKA_BROKER", ""),
			BatchSize: parseInt("KAFKA_BATCH_SIZE", 100),
			BatchBytes:      parseInt("KAFKA_BATCH_BYTES", 1048576),
			BatchTimeout:    parseInt("KAFKA_BATCH_TIMEOUT", 1000),
			ConsumerGroupID: e.GetOrDefault("KAFKA_CONSUMER_GROUP_ID", "default-group"),
			Partition:       parseInt("KAFKA_PARTITION", 0),

		},
		TracerHost: e.GetOrDefault("TRACER_HOST", "localhost:4317"),
	}

	if e.autoCreateFile && writeFileDotEnv {
		if err := createEnvFile(overrideFile); err != nil {
			log.Fatalf("Failed to create default .env file: %v", err)
		}
	}


	return cfg
}

func (*EnvLoader) Get(key string) string {
	return os.Getenv(key)
}

func (*EnvLoader) GetOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}

	return defaultValue
}

func parseInt(key string, defaultValue int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}

	parsed, err := strconv.Atoi(val)
	if err != nil {
		log.Printf("Invalid integer value for %s: %v, using default: %d", key, err, defaultValue)
		return defaultValue
	}

	return parsed
}

func parseBool(key string, defaultValue bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}

	parsed, err := strconv.ParseBool(val)
	if err != nil {
		log.Printf("Invalid boolean value for %s: %v, using default: %v", key, err, defaultValue)
		return defaultValue
	}

	return parsed
}

func createEnvFile(path string) error {
	defaultEnv := `# Application
APP_NAME=MyApp
APP_COMPONENT_NAME=Logger
APP_DESCRIPTION=Logging Service
APP_VERSION=1.0.0
APP_BASE_API_VERSION=v1
APP_SCHEMA_VERSION=1.0

# Log Detail
LOG_DETAIL_LEVEL=debug
LOG_DETAIL_ENABLE_FILE_LOGGING=true
LOG_DETAIL_DIRNAME=./logs/detail
LOG_DETAIL_FILENAME=detail-%DATE%
LOG_DETAIL_DATE_PATTERN=YYYY-MM-DD-HH
LOG_DETAIL_EXTENSION=.log

# Log Summary
LOG_SUMMARY_LEVEL=info
LOG_SUMMARY_ENABLE_FILE_LOGGING=true
LOG_SUMMARY_DIRNAME=./logs/summary
LOG_SUMMARY_FILENAME=summary-%DATE%
LOG_SUMMARY_DATE_PATTERN=YYYY-MM-DD-HH
LOG_SUMMARY_EXTENSION=.log

# Server
SERVER_APP_PORT=8080
SERVER_APP_HOST=localhost
SERVER_HTTPS=false
SERVER_CERT=./cert.pem
SERVER_KEY=./key.pem

# Tracing
TRACER_HOST=localhost:4317
`

	// Only create file if it doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte(defaultEnv), 0644); err != nil {
			return fmt.Errorf("failed to create .env file: %w", err)
		}
		fmt.Printf("✅ Created default .env at: %s\n", path)
	}
	return nil
}
