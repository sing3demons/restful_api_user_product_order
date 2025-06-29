package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	config "github.com/sing3demons/go-order-service/configs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type LoggerService interface {
	Debugf(format string, args ...any)
	Debug(args string)
	Logf(format string, args ...any)
	Log(data string)
	Info(msg string)
	Errorf(format string, args ...any)
	Error(args string)
	Sync() error
}

type ILogger interface {
	Debugf(format string, args ...any)
	Debug(args string)
	Logf(format string, args ...any)
	Log(data string)
	Info(msg string)
	Errorf(format string, args ...any)
	Error(args string)
	Sync() error
}

func getGoroutineID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	i := bytes.IndexByte(b, ' ')
	id, _ := strconv.ParseUint(string(b[:i]), 10, 64)
	return id
}

type defaultLoggerService struct {
}

func NewDefaultLoggerService() ILogger {
	return &defaultLoggerService{}
}
func (d *defaultLoggerService) Debugf(format string, args ...any) {
	log.Printf("[DEBUG] "+format, args...)
}
func (d *defaultLoggerService) Debug(args string) {
	log.Printf("[DEBUG] %s", args)
}
func (d *defaultLoggerService) Logf(format string, args ...any) {
	log.Printf("[INFO] "+format, args...)
}
func (d *defaultLoggerService) Log(data string) {
	log.Printf("[INFO] %s", data)
}
func (d *defaultLoggerService) Info(msg string) {
	log.Printf("[INFO] %s", msg)
}
func (d *defaultLoggerService) Errorf(format string, args ...any) {
	log.Printf("[ERROR] "+format, args...)
}
func (d *defaultLoggerService) Error(args string) {
	log.Printf("[ERROR] %s", args)
}
func (d *defaultLoggerService) Sync() error {
	// No-op for default logger
	return nil
}

type zLogger struct {
	*zap.Logger
}

func (k *zLogger) Debugf(format string, args ...any) {
	k.Logger.Sugar().Debugf(format, args...)
}

func (k *zLogger) Debug(args string) {
	k.Logger.Debug(args)
}

func (k *zLogger) Info(msg string) {
	k.Logger.Info(msg)
}

func (k *zLogger) Logf(format string, args ...any) {
	k.Logger.Sugar().Infof(format, args...)
}
func (k *zLogger) Log(data string) {
	k.Logger.Info(data)
}
func (k *zLogger) Errorf(format string, args ...any) {
	// fmt.Printf(format, args...)
	k.Logger.Sugar().Errorf(format, args...)
}
func (k *zLogger) Error(args string) {
	k.Logger.Error(args)
}

func NewLogger(cfg config.LogConfig) ILogger {
	logger, err := BuildZapLogger(cfg, true)
	if err != nil {
		log.Fatalf("failed to build detail logger: %v", err)
	}

	// _, err = BuildZapLogger(cfg.Log.Summary, true)
	// if err != nil {
	// 	log.Fatalf("failed to build summary logger: %v", err)
	// }

	if os.Getenv("MODE") == "test" {
		logger = zap.NewNop()
	}
	customLog := &zLogger{Logger: logger}

	return customLog
}

func (k *zLogger) Sync() error {
	return k.Logger.Sync()
}

// formatFilename replaces %DATE% with the actual date using the pattern
func formatFilename(dirname, filenamePattern, datePattern, extension string) string {
	now := time.Now()
	formattedDate := now.Format(convertDatePattern(datePattern)) // translate custom to Go layout
	filename := strings.ReplaceAll(filenamePattern, "%DATE%", formattedDate)
	return filepath.Join(dirname, filename+extension)
}

// convertDatePattern converts common formats like "YYYY-MM-DD-HH" to Go's time layout
func convertDatePattern(pattern string) string {
	replacer := strings.NewReplacer(
		"YYYY", "2006",
		"MM", "01",
		"DD", "02",
		"HH", "15",
		"mm", "04",
		"ss", "05",
	)
	return replacer.Replace(pattern)
}

type LogFileProperties struct {
	Dirname     string `json:"dirname" yaml:"dirname"`
	Filename    string `json:"filename" yaml:"filename"`
	DatePattern string `json:"date-pattern" yaml:"date-pattern"`
	Extension   string `json:"extension" yaml:"extension"`
}

func newWriteSyncer(p LogFileProperties) zapcore.WriteSyncer {
	file := formatFilename(p.Dirname, p.Filename, p.DatePattern, p.Extension)
	return zapcore.AddSync(&lumberjack.Logger{
		Filename:   file,
		MaxSize:    500, // MB
		MaxBackups: 3,
		MaxAge:     1, // Days
		LocalTime:  true,
		Compress:   true,
	})
}

type LogConfig struct {
	Level             string            `json:"level" yaml:"level"`
	EnableFileLogging bool              `json:"enable-file-logging" yaml:"enable-file-logging"`
	LogFileProperties LogFileProperties `json:"log-file-properties" yaml:"log-file-properties"`
}

func getZapLevel(levelStr string) zapcore.Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func BuildZapLogger(cfg config.LogConfig, withConsole bool) (*zap.Logger, error) {
	encoderConfig := map[string]any{
		"messageKey": "msg",
	}
	data, _ := json.Marshal(encoderConfig)
	var encCfg zapcore.EncoderConfig
	if err := json.Unmarshal(data, &encCfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal encoder config: %w", err)
	}
	encoder := zapcore.NewConsoleEncoder(encCfg)

	var cores []zapcore.Core

	if cfg.EnableFileLogging {
		ws := newWriteSyncer(LogFileProperties{
			Dirname:     cfg.LogFileProperties.Dirname,
			Filename:    cfg.LogFileProperties.Filename,
			DatePattern: cfg.LogFileProperties.DatePattern,
			Extension:   cfg.LogFileProperties.Extension,
		})
		level := getZapLevel(cfg.Level)
		core := zapcore.NewCore(encoder, ws, level)
		cores = append(cores, core)
	}

	if withConsole {
		consoleEncoder := zapcore.NewConsoleEncoder(encCfg)
		consoleCore := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), getZapLevel(cfg.Level))
		cores = append(cores, consoleCore)
	}

	if len(cores) == 0 {
		cores = append(cores, zapcore.NewNopCore())
	}

	return zap.New(zapcore.NewTee(cores...)), nil
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
