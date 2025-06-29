package kp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	config "github.com/sing3demons/go-user-service/configs"
	goHttp "github.com/sing3demons/go-user-service/pkg/http"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type httpServer struct {
	router      *goHttp.Router
	port        string
	srv         *http.Server
	certFile    string
	keyFile     string
	staticFiles map[string]string
}

var (
	errInvalidCertificateFile = errors.New("invalid certificate file")
	errInvalidKeyFile         = errors.New("invalid key file")
)

func otelMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		// Start a new span
		tr := otel.GetTracerProvider().Tracer("gokp")
		spanName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		ctx, span := tr.Start(ctx, spanName)
		defer span.End()

		fmt.Println("Span started for: ---> ", spanName)
		fmt.Println("TraceID:", span.SpanContext().TraceID().String())
		fmt.Println("SpanID:", span.SpanContext().SpanID().String())

		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.url", r.URL.String()),
			attribute.String("http.user_agent", r.UserAgent()),
			attribute.String("http.client_ip", r.RemoteAddr),
		)

		handler.ServeHTTP(w, r.WithContext(ctx))
	})
}

func newHTTPServer(conf *config.Config, tp *sdktrace.TracerProvider) *httpServer {
	router := goHttp.NewRouter()
	if tp != nil {
		otel.SetTracerProvider(tp)
		otel.SetTextMapPropagator(propagation.TraceContext{})

		router.UseMiddleware(otelMiddleware)
	}

	srv := &http.Server{
		Addr:    ":" + conf.Server.AppPort,
		Handler: router,
	}

	httpSrv := &httpServer{
		router:      router,
		port:        conf.Server.AppPort,
		srv:         srv,
		certFile:    conf.Server.Cert,
		keyFile:     conf.Server.Key,
		staticFiles: make(map[string]string),
	}

	return httpSrv
}

func (s *httpServer) validateCertificateAndKeyFiles(certificateFile, keyFile string) bool {
	if certificateFile == "" || keyFile == "" {
		return false
	}
	if _, err := os.Stat(certificateFile); os.IsNotExist(err) {
		fmt.Printf("%v : %v\n", errInvalidCertificateFile, certificateFile)
		return false
	}

	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		fmt.Printf("%v : %v\n", errInvalidKeyFile, keyFile)
		return false
	}

	return true
}

func (s *httpServer) run() {

	s.srv = &http.Server{
		Addr:    ":" + s.port,
		Handler: s.router,
	}

	s.srv.ReadTimeout = 10 * time.Second
	s.srv.WriteTimeout = 10 * time.Second
	s.srv.MaxHeaderBytes = 1 << 20 // 1 MB

	if s.validateCertificateAndKeyFiles(s.certFile, s.keyFile) {
		fmt.Println("Starting HTTPS server...", s.certFile, s.keyFile)
		if err := s.srv.ListenAndServeTLS(s.certFile, s.keyFile); err != nil {
			fmt.Printf("Error starting HTTPS server: %v\n", err)
		}
	} else {
		fmt.Println("Starting HTTP server port:", s.port)
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Error starting HTTP server: %v\n", err)
		}
	}
}

func (s *httpServer) Shutdown(ctx context.Context) error {
	if s.srv == nil {
		return nil
	}

	return ShutdownWithContext(ctx, func(ctx context.Context) error {
		return s.srv.Shutdown(ctx)
	}, func() error {
		if err := s.srv.Close(); err != nil {
			return err
		}

		return nil
	})
}

func ShutdownWithContext(ctx context.Context, shutdownFunc func(ctx context.Context) error, forceCloseFunc func() error) error {
	errCh := make(chan error, 1) // Channel to receive shutdown error

	go func() {
		errCh <- shutdownFunc(ctx) // Run shutdownFunc in a goroutine and send any error to errCh
	}()

	// Wait for either the context to be done or shutdownFunc to complete
	select {
	case <-ctx.Done(): // Context timeout reached
		err := ctx.Err()

		if forceCloseFunc != nil {
			err = errors.Join(err, forceCloseFunc()) // Attempt force close if available
		}

		return err
	case err := <-errCh:
		return err
	}
}

const shutDownTimeout time.Duration = 30 * time.Second

func getShutdownTimeoutFromConfig() (time.Duration, error) {
	value := "30s"
	if value == "" {
		return shutDownTimeout, nil
	}

	timeout, err := time.ParseDuration(value)
	if err != nil {
		return shutDownTimeout, err
	}

	return timeout, nil
}
