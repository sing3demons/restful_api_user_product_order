package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel/trace"
)

type Client interface {
	Publish(ctx context.Context, topic string, message []byte) error
	Subscribe(ctx context.Context, topic string) (*Message, error)

	CreateTopic(context context.Context, name string) error
	DeleteTopic(context context.Context, name string) error

	Close() error
}

type Committer interface {
	Commit()
}

var errNotPointer = errors.New("input should be a pointer to a variable")

type Message struct {
	ctx context.Context

	Topic    string
	Value    []byte
	MetaData any

	Committer

	TransactionID string
	SessionID     string
	RequestID     string
}

type MsgConsumer struct {
	Header struct {
		Broker      string
		UseCase     string
		UseCaseStep string
		Session     string
		Transaction string
	}
	Body interface{}
}

func NewMessage(ctx context.Context, msg kafka.Message) *Message {
	if ctx == nil {
		ctx = context.Background()
	}

	data := MsgConsumer{}

	var extractHeaders bool = true

	// Extract headers if available
	if len(msg.Headers) != 0 {
		for _, header := range msg.Headers {
			switch header.Key {
			case strings.ToLower("X-Transaction-ID"):
				data.Header.Transaction = string(header.Value)
				extractHeaders = false
			case strings.ToLower("X-Session-ID"):
				data.Header.Session = string(header.Value)
				extractHeaders = false
			}
		}
	}

	if extractHeaders {
		json.Unmarshal(msg.Value, &data)
	}

	if data.Header.Transaction == "" {
		data.Header.Transaction = uuid.NewString()
	}
	if data.Header.Session == "" {
		data.Header.Session = uuid.NewString()
	}
	traceID := trace.SpanFromContext(ctx).SpanContext().TraceID().String()

	return &Message{
		ctx:           ctx,
		TransactionID: data.Header.Transaction,
		SessionID:     data.Header.Session,
		RequestID:     traceID,
	}
}

func (m *Message) Context() context.Context {
	return m.ctx
}

func (m *Message) Param(p string) string {
	if p == "topic" {
		return m.Topic
	}

	return ""
}

func (m *Message) PathParam(p string) string {
	return m.Param(p)
}

func (m *Message) ClientIP() string {
	return ""
}
func (m *Message) UserAgent() string {
	return ""
}
func (m *Message) Method() string {
	return "kafka" // Kafka messages are typically sent via POST
}
func (m *Message) URL() string {
	return m.Topic // Using topic as URL for Kafka messages
}
func (m *Message) TransactionId() string {
	return m.TransactionID // Kafka messages can have a transaction ID
}
func (m *Message) SessionId() string {
	return m.SessionID // Kafka messages can have a session ID
}
func (m *Message) RequestId() string {
	return m.RequestID // Kafka messages can have a request ID
}
func (m *Message) Referer() string {
	return "" // Kafka messages do not have a referer by default
}
func (m *Message) HostName() string {
	return "" // Kafka messages do not have a hostname like HTTP requests
}
func (m *Message) Headers() map[string]any {
	return nil // Kafka messages do not have headers like HTTP requests
}
func (m *Message) Header(key string) string {
	return "" // Kafka messages do not have headers like HTTP requests
}

// Bind binds the message value to the input variable. The input should be a pointer to a variable.
func (m *Message) Bind(i any) error {
	if reflect.ValueOf(i).Kind() != reflect.Ptr {
		return errNotPointer
	}

	switch v := i.(type) {
	case *string:
		return m.bindString(v)
	case *float64:
		return m.bindFloat64(v)
	case *int:
		return m.bindInt(v)
	case *bool:
		return m.bindBool(v)
	default:
		return m.bindStruct(i)
	}
}

func (m *Message) bindString(v *string) error {
	*v = string(m.Value)
	return nil
}

func (m *Message) bindFloat64(v *float64) error {
	f, err := strconv.ParseFloat(string(m.Value), 64)
	if err != nil {
		return err
	}

	*v = f

	return nil
}

func (m *Message) bindInt(v *int) error {
	in, err := strconv.Atoi(string(m.Value))
	if err != nil {
		return err
	}

	*v = in

	return nil
}

func (m *Message) bindBool(v *bool) error {
	b, err := strconv.ParseBool(string(m.Value))
	if err != nil {
		return err
	}

	*v = b

	return nil
}

func (m *Message) bindStruct(i any) error {
	return json.Unmarshal(m.Value, i)
}

func (*Message) Params(string) []string {
	return nil
}
