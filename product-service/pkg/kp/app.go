package kp

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	config "github.com/sing3demons/go-product-service/configs"
	"github.com/sing3demons/go-product-service/pkg/kafka"
	"github.com/sing3demons/go-product-service/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	"golang.org/x/sync/errgroup"
)

type App struct {
	httpServer  *httpServer
	kafkaClient *KafkaClient
	conf        *config.Config

	traceProvider *trace.TracerProvider

	maskingService logger.MaskingServiceInterface
	AppLog         logger.LoggerService
	DetailLog      logger.LoggerService
	SummaryLog     logger.LoggerService
}

func (a *App) Shutdown(ctx context.Context) error {
	if a.httpServer != nil {
		if err := a.httpServer.Shutdown(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) startConsumer(ctx context.Context) error {
	if len(a.kafkaClient.subscriptions) == 0 {
		return nil
	}

	group := errgroup.Group{}
	// Start subscribers concurrently using go-routines
	for topic, handler := range a.kafkaClient.subscriptions {
		subscriberTopic, subscriberHandler := topic, handler

		group.Go(func() error {
			return a.kafkaClient.startKafkaConsumer(ctx, subscriberTopic, subscriberHandler)
		})
	}

	return group.Wait()
}

func (a *App) add(method, pattern string, h Handler) {
	hf := handler{
		function:       h,
		requestTimeout: time.Duration(10) * time.Second,
		logService: LogService{
			appLog:         a.AppLog,
			detailLog:      a.DetailLog,
			summaryLog:     a.SummaryLog,
			maskingService: a.maskingService,
		},
		conf: a.conf,
	}

	if a.kafkaClient != nil {
		hf.kafkaClient = a.kafkaClient.kafkaClient
	}
	a.httpServer.router.Add(method, pattern, hf)
}

type IApplication interface {
	Get(pattern string, handler Handler)
	Post(pattern string, handler Handler)
	Put(pattern string, handler Handler)
	Patch(pattern string, handler Handler)
	Delete(pattern string, handler Handler)
	Consumer(topic string, handler SubscribeFunc)
	Start()
	CreateTopic(topic string)

	StartKafka()

	LogDetail(logger logger.LoggerService)
	LogSummary(logger logger.LoggerService)
}

func NewApplication(conf *config.Config, log logger.ILogger) IApplication {
	noopLogger := logger.NewDefaultLoggerService()

	var traceProvider *trace.TracerProvider
	if conf.TracerHost != "" {
		tp, err := startTracing(conf.App.Name, conf.TracerHost)
		if err != nil {
			log.Errorf("Failed to start tracing: %v", err)
		} else {
			traceProvider = tp

			otel.SetTracerProvider(traceProvider)
			otel.SetTextMapPropagator(propagation.TraceContext{})
		}
	}

	app := &App{
		conf:           conf,
		AppLog:         log,
		DetailLog:      noopLogger,
		SummaryLog:     noopLogger,
		maskingService: logger.NewMaskingService(),
	}

	app.httpServer = newHTTPServer(conf, traceProvider)
	// app.kafkaClient = kafka.New(&kafka.Config{})

	return app
}

func (a *App) StartKafka() {

	if a.conf.Kafka.Broker == "" {
		panic("Kafka broker is not configured.")
	}

	kafkaClient := kafka.New(&kafka.Config{
		Brokers:         strings.Split(a.conf.Kafka.Broker, ","),
		BatchSize:       a.conf.Kafka.BatchSize,
		BatchBytes:      a.conf.Kafka.BatchBytes,
		BatchTimeout:    a.conf.Kafka.BatchTimeout,
		ConsumerGroupID: a.conf.Kafka.ConsumerGroupID,
	})

	a.kafkaClient = newKafkaClient(kafkaClient, LogService{
		maskingService: a.maskingService,
		appLog:         a.AppLog,
		detailLog:      a.DetailLog,
		summaryLog:     a.SummaryLog,
	}, a.conf)
	fmt.Println("Kafka client initialized...")

}

func (a *App) LogDetail(logger logger.LoggerService) {
	a.DetailLog = logger
}

func (a *App) LogSummary(logger logger.LoggerService) {
	a.SummaryLog = logger
}

func (a *App) Get(pattern string, handler Handler) {
	a.add(http.MethodGet, pattern, handler)
}
func (a *App) Post(pattern string, handler Handler) {
	a.add(http.MethodPost, pattern, handler)
}
func (a *App) Put(pattern string, handler Handler) {
	a.add(http.MethodPut, pattern, handler)
}
func (a *App) Patch(pattern string, handler Handler) {
	a.add(http.MethodPatch, pattern, handler)
}
func (a *App) Delete(pattern string, handler Handler) {
	a.add(http.MethodDelete, pattern, handler)
}

func (a *App) Consumer(topic string, handler SubscribeFunc) {
	if a.kafkaClient == nil {
		fmt.Println("Kafka client is not initialized.")
		return
	}

	if _, exists := a.kafkaClient.subscriptions[topic]; exists {
		fmt.Printf("Subscription for topic %s already exists.\n", topic)
		return
	}

	a.kafkaClient.subscriptions[topic] = handler
	fmt.Printf("Subscribed to topic %s successfully.\n", topic)
}

func (a *App) CreateTopic(topic string) {
	if a.kafkaClient == nil {
		fmt.Println("Kafka client is not initialized.")
		return
	}

	if err := a.kafkaClient.kafkaClient.CreateTopic(context.Background(), topic); err != nil {
		fmt.Printf("Error creating topic %s: %v\n", topic, err)
	} else {
		fmt.Printf("Topic %s created successfully.\n", topic)
	}
}

func startTracing(appName, endpoint string) (*trace.TracerProvider, error) {
	headers := map[string]string{
		"content-type": "application/json",
	}

	exporter, err := otlptrace.New(
		context.Background(),
		otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(endpoint),
			otlptracehttp.WithHeaders(headers),
			otlptracehttp.WithInsecure(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating new exporter: %w", err)
	}

	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(
			exporter,
			trace.WithMaxExportBatchSize(trace.DefaultMaxExportBatchSize),
			trace.WithBatchTimeout(trace.DefaultScheduleDelay*time.Millisecond),
			trace.WithMaxExportBatchSize(trace.DefaultMaxExportBatchSize),
		),
		trace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(appName),
			),
		),
	)

	otel.SetTracerProvider(tracerProvider)

	return tracerProvider, nil
}

func (a *App) Start() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	timeout, _ := getShutdownTimeoutFromConfig()

	wg := sync.WaitGroup{}

	if a.httpServer != nil {
		wg.Add(1)
		a.AppLog.Debugf("Starting HTTP server on port %d", a.httpServer.port)
		go func(s *httpServer) {
			defer wg.Done()
			s.run()
		}(a.httpServer)
	}

	if a.kafkaClient != nil {
		wg.Add(1)
		a.AppLog.Debugf("Starting Kafka consumer with subscriptions: %v", a.kafkaClient.subscriptions)
		go func() {
			defer wg.Done()
			if err := a.startConsumer(ctx); err != nil {
				a.AppLog.Errorf("Error starting Kafka consumer: %v", err)
			}
		}()
	}

	if a.traceProvider != nil {
		wg.Add(1)
		a.AppLog.Debug("Starting tracer provider shutdown")
		go func() {
			defer wg.Done()
			// Wait for the tracer provider to be shutdown
			if err := a.traceProvider.Shutdown(ctx); err != nil {
				a.AppLog.Errorf("Error shutting down tracer provider: %v", err)
			} else {
				a.AppLog.Debug("Tracer provider shutdown completed successfully.")
			}
		}()
	}

	go func() {
		<-ctx.Done()

		// Give services time to shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := a.Shutdown(shutdownCtx); err != nil {
			a.AppLog.Errorf("Server shutdown failed: %v", err)
		} else {
			a.AppLog.Debug("Server shutdown completed successfully.")
		}
	}()

	wg.Wait()
}
