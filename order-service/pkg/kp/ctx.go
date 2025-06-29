package kp

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	config "github.com/sing3demons/go-order-service/configs"
	"github.com/sing3demons/go-order-service/pkg/kafka"
	"github.com/sing3demons/go-order-service/pkg/logger"
	"go.opentelemetry.io/otel/trace"
)

type Context struct {
	context.Context
	Request
	http.ResponseWriter
	kafka.Client
	detail logger.CustomLoggerService
}
type SubscribeFunc func(c *Context) error

type Request interface {
	Context() context.Context
	Param(string) string
	PathParam(string) string
	Bind(any) error
	HostName() string
	Params(string) []string
	ClientIP() string
	UserAgent() string
	Referer() string
	Method() string
	URL() string
	TransactionId() string
	SessionId() string
	RequestId() string
}
type LogService struct {
	appLog         logger.LoggerService
	detailLog      logger.LoggerService
	summaryLog     logger.LoggerService
	maskingService logger.MaskingServiceInterface
}

func newContext(w http.ResponseWriter, r Request, k kafka.Client, log LogService, conf *config.Config) *Context {
	log.appLog.Debugf("configuring context for request", conf)
	c := r.Context()
	if c == nil {
		c = context.Background()
	}
	traceID := trace.SpanFromContext(c).SpanContext().TraceID().String()
	spanId := trace.SpanFromContext(c).SpanContext().SpanID().String()

	t := logger.NewTimer()
	kpLog := logger.NewCustomLogger(log.detailLog, log.summaryLog, t, log.maskingService)
	ctx := &Context{
		Context:        c,
		Request:        r,
		ResponseWriter: w,
		Client:         k,
	}

	broker := "none"
	source := "api"
	if w == nil {
		broker = r.HostName()
		source = "event-source"
	}

	meta := logger.Metadata{
		ClientIP:  r.ClientIP(),
		UserAgent: r.UserAgent(),
		Referer:   r.Referer(),
		Method:    r.Method(),
		URL:       r.URL(),
		Source:    source,
		Broker:    broker,
		TraceId:   traceID,
		SpanId:    spanId,
	}
	hostName, _ := os.Hostname()

	customLog := logger.LogDto{
		ServiceName:      conf.App.Name,
		LogType:          "detail",
		ComponentVersion: conf.App.Version,
		Instance:         hostName,
		Metadata:         meta,
	}
	kpLog.Init(customLog)

	// kpLog.Init(commonlog.LogDto{
	// 	Channel:          "none",
	// 	UseCase:          "none",
	// 	UseCaseStep:      "none",
	// 	Broker:           broker,
	// 	TransactionId:    ctx.Request.TransactionId(),
	// 	SessionId:        ctx.Request.SessionId(),
	// 	RequestId:        ctx.Request.RequestId(),
	// 	AppName:          conf.App.Name,
	// 	ComponentVersion: conf.App.Version,
	// 	ComponentName:    conf.App.ComponentName,
	// 	Instance:         ctx.Request.HostName(),
	// 	// DateTime:         time.Now().Format(time.RFC3339),
	// 	OriginateServiceName: func() string {
	// 		if w != nil {
	// 			return "HTTP Service"
	// 		}
	// 		return "Event Source"
	// 	}(),
	// 	RecordType: "detail",
	// })

	ctx.detail = kpLog
	return ctx
}

func (c *Context) JSON(code int, v any) error {
	if c.ResponseWriter != nil {
		c.ResponseWriter.Header().Set("Content-Type", "application/json; charset=UTF8")
		c.ResponseWriter.WriteHeader(code)

		if err := json.NewEncoder(c.ResponseWriter).Encode(v); err != nil {
			return err
		}
		c.detail.Info(logger.NewOutbound("client", ""), v)
	}

	c.detail.End(code, "")

	return nil
}

func (c *Context) Log() logger.CustomLoggerService {
	return c.detail
}
