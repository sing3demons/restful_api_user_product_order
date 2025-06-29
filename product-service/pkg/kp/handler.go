package kp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gorilla/websocket"
	config "github.com/sing3demons/go-product-service/configs"
	goHTTP "github.com/sing3demons/go-product-service/pkg/http"
	"github.com/sing3demons/go-product-service/pkg/kafka"
)

type Handler func(c *Context) error

type handler struct {
	function       Handler
	requestTimeout time.Duration
	kafkaClient    kafka.Client
	logService     LogService
	conf           *config.Config
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	c := newContext(w, goHTTP.NewRequest(r), h.kafkaClient, h.logService, h.conf)
	// traceID := trace.SpanFromContext(r.Context()).SpanContext().TraceID().String()

	if websocket.IsWebSocketUpgrade(r) {
		// If the request is a WebSocket upgrade, do not apply the timeout
		c.Context = r.Context()
	} else if h.requestTimeout != 0 {
		ctx, cancel := context.WithTimeout(r.Context(), h.requestTimeout)
		defer cancel()

		c.Context = ctx
	}

	done := make(chan struct{})
	panicked := make(chan struct{})

	var (
		err error
	)

	go func() {
		defer func() {
			panicRecoveryHandler(recover(), panicked)
		}()
		// Execute the handler function
		err = h.function(c)
		// h.logError(traceID, err)
		close(done)
	}()

	select {
	case <-c.Context.Done():
		// If the context's deadline has been exceeded, return a timeout error response
		if errors.Is(c.Err(), context.DeadlineExceeded) {
			err = errors.New("request timed out")
		}
	case <-done:
		handleWebSocketUpgrade(r)
	case <-panicked:
		err = errors.New("internal server error")
	}

	// Handler function completed
	if err != nil {
		c.ResponseWriter.Write([]byte(err.Error()))
		return
	}

}
func panicRecoveryHandler(re any, panicked chan struct{}) {
	if re == nil {
		return
	}

	close(panicked)
	panicLog := struct {
		Error      string `json:"error,omitempty"`
		StackTrace string `json:"stack_trace,omitempty"`
	}{
		Error:      fmt.Sprint(re),
		StackTrace: string(debug.Stack()),
	}
	fmt.Println(panicLog)
}

func handleWebSocketUpgrade(r *http.Request) {
	if websocket.IsWebSocketUpgrade(r) {
		// Do not respond with HTTP headers since this is a WebSocket request
		return
	}
}
