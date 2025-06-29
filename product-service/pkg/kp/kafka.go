package kp

import (
	"context"
	"runtime/debug"

	config "github.com/sing3demons/go-product-service/configs"
	"github.com/sing3demons/go-product-service/pkg/kafka"
	"github.com/sing3demons/go-product-service/pkg/logger"
)

type KafkaClient struct {
	kafkaClient    kafka.Client
	subscriptions  map[string]SubscribeFunc
	log            LogService
	maskingService logger.MaskingServiceInterface
	conf           *config.Config
}

func newKafkaClient(kafkaClient kafka.Client, log LogService, conf *config.Config) *KafkaClient {
	return &KafkaClient{
		kafkaClient:    kafkaClient,
		subscriptions:  make(map[string]SubscribeFunc),
		log:            log,
		maskingService: log.maskingService,
		conf:           conf,
	}
}

func (kc *KafkaClient) startKafkaConsumer(ctx context.Context, topic string, handler SubscribeFunc) error {
	for {
		select {
		case <-ctx.Done():
			kc.log.appLog.Logf("shutting down subscriber for topic %s", topic)
			return nil
		default:
			err := kc.handleSubscription(ctx, topic, handler)
			if err != nil {
				kc.log.appLog.Errorf("error in subscription for topic %s: %v", topic, err)
			}
		}
	}
}

func (kc *KafkaClient) handleSubscription(ctx context.Context, topic string, handler SubscribeFunc) error {
	msg, err := kc.kafkaClient.Subscribe(ctx, topic)
	if err != nil {
		kc.log.appLog.Errorf("error subscribing to topic %s: %v", topic, err)
		return err
	}

	if msg == nil {
		return nil
	}

	msgCtx := newContext(nil, msg, kc.kafkaClient, kc.log, kc.conf)
	err = func(ctx *Context) error {
		defer func() {
			panicRecovery(recover(), kc.log.appLog)
		}()

		return handler(ctx)
	}(msgCtx)

	if err != nil {
		kc.log.appLog.Errorf("error in handler for topic %s: %v", topic, err)
		return nil
	}

	if msg.Committer != nil {
		msg.Commit()
	}

	return nil
}

func panicRecovery(re any, log logger.ILogger) {
	if re == nil {
		return
	}

	var e string
	switch t := re.(type) {
	case string:
		e = t
	case error:
		e = t.Error()
	default:
		e = "Unknown panic type"
	}

	// log.Error(panicLog{
	// 	Error:      e,
	// 	StackTrace: string(debug.Stack()),
	// })

	log.Errorf("panic recovered: %v", panicLog{
		Error:      e,
		StackTrace: string(debug.Stack()),
	})
}

type panicLog struct {
	Error      string `json:"error,omitempty"`
	StackTrace string `json:"stack_trace,omitempty"`
}
