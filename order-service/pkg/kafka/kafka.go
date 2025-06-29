package kafka

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
	"go.opentelemetry.io/otel"
)

var (
	ErrConsumerGroupNotProvided    = errors.New("consumer group id not provided")
	errFailedToConnectBrokers      = errors.New("failed to connect to any kafka brokers")
	errBrokerNotProvided           = errors.New("kafka broker address not provided")
	errPublisherNotConfigured      = errors.New("can't publish message. Publisher not configured or topic is empty")
	errBatchSize                   = errors.New("KAFKA_BATCH_SIZE must be greater than 0")
	errBatchBytes                  = errors.New("KAFKA_BATCH_BYTES must be greater than 0")
	errBatchTimeout                = errors.New("KAFKA_BATCH_TIMEOUT must be greater than 0")
	errClientNotConnected          = errors.New("kafka client not connected")
	errUnsupportedSASLMechanism    = errors.New("unsupported SASL mechanism")
	errSASLCredentialsMissing      = errors.New("SASL credentials missing")
	errUnsupportedSecurityProtocol = errors.New("unsupported security protocol")
	errNoActiveConnections         = errors.New("no active connections to brokers")
	errCACertFileRead              = errors.New("failed to read CA certificate file")
	errClientCertLoad              = errors.New("failed to load client certificate")
)

const (
	DefaultBatchSize       = 100
	DefaultBatchBytes      = 1048576
	DefaultBatchTimeout    = 1000
	defaultRetryTimeout    = 10 * time.Second
	protocolPlainText      = "PLAINTEXT"
	protocolSASL           = "SASL_PLAINTEXT"
	protocolSSL            = "SSL"
	protocolSASLSSL        = "SASL_SSL"
	messageMultipleBrokers = "MULTIPLE_BROKERS"
	brokerStatusUp         = "UP"
)

type (
	Config struct {
		Brokers          []string
		Partition        int
		ConsumerGroupID  string
		OffSet           int
		BatchSize        int
		BatchBytes       int
		BatchTimeout     int
		RetryTimeout     time.Duration
		SASLMechanism    string
		SASLUser         string
		SASLPassword     string
		SecurityProtocol string
		TLS              TLSConfig
	}

	TLSConfig struct {
		CertFile           string
		KeyFile            string
		CACertFile         string
		InsecureSkipVerify bool
	}

	kafkaClient struct {
		dialer *kafka.Dialer
		conn   *multiConn
		writer Writer
		reader map[string]Reader
		mu     *sync.RWMutex
		config Config
	}

	multiConn struct {
		conns  []Connection
		dialer *kafka.Dialer
		mu     sync.RWMutex
	}

	kafkaMessage struct {
		msg    *kafka.Message
		reader Reader
	}
)

func New(conf *Config) *kafkaClient {
	if err := validateConfigs(conf); err != nil {
		return nil
	}
	client := &kafkaClient{
		config: *conf,
		mu:     &sync.RWMutex{},
	}
	ctx := context.Background()
	if err := client.initialize(ctx); err != nil {
		go client.retryConnect(ctx)
	}
	return client
}

func (k *kafkaClient) initialize(ctx context.Context) error {
	dialer, err := setupDialer(&k.config)
	if err != nil {
		return err
	}
	conns, err := connectToBrokers(ctx, k.config.Brokers, dialer)
	if err != nil {
		return err
	}
	k.dialer = dialer
	k.conn = &multiConn{conns: conns, dialer: dialer}
	k.writer = createKafkaWriter(&k.config, dialer)
	k.reader = make(map[string]Reader)
	return nil
}

func (k *kafkaClient) getNewReader(topic string) Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		GroupID:     k.config.ConsumerGroupID,
		Brokers:     k.config.Brokers,
		Topic:       topic,
		MinBytes:    10e3,
		MaxBytes:    10e6,
		Dialer:      k.dialer,
		StartOffset: int64(k.config.OffSet),
	})
}

func (k *kafkaClient) DeleteTopic(_ context.Context, name string) error {
	return k.conn.DeleteTopics(name)
}

func (k *kafkaClient) Controller() (kafka.Broker, error) {
	return k.conn.Controller()
}

func (k *kafkaClient) CreateTopic(_ context.Context, name string) error {
	return k.conn.CreateTopics(kafka.TopicConfig{Topic: name, NumPartitions: 1, ReplicationFactor: 1})
}

func (m *multiConn) Controller() (kafka.Broker, error) {
	if len(m.conns) == 0 {
		return kafka.Broker{}, errNoActiveConnections
	}
	for _, conn := range m.conns {
		if conn == nil {
			continue
		}
		controller, err := conn.Controller()
		if err == nil {
			return controller, nil
		}
	}
	return kafka.Broker{}, errNoActiveConnections
}

func (m *multiConn) CreateTopics(topics ...kafka.TopicConfig) error {
	controller, err := m.Controller()
	if err != nil {
		return err
	}
	controllerAddr := net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port))
	controllerResolvedAddr, err := net.ResolveTCPAddr("tcp", controllerAddr)
	if err != nil {
		return err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, conn := range m.conns {
		if conn == nil {
			continue
		}
		if tcpAddr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
			if tcpAddr.IP.Equal(controllerResolvedAddr.IP) && tcpAddr.Port == controllerResolvedAddr.Port {
				return conn.CreateTopics(topics...)
			}
		}
	}
	conn, err := m.dialer.DialContext(context.Background(), "tcp", controllerAddr)
	if err != nil {
		return err
	}
	m.conns = append(m.conns, conn)
	return conn.CreateTopics(topics...)
}

func (m *multiConn) DeleteTopics(topics ...string) error {
	controller, err := m.Controller()
	if err != nil {
		return err
	}
	controllerAddr := net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port))
	controllerResolvedAddr, err := net.ResolveTCPAddr("tcp", controllerAddr)
	if err != nil {
		return err
	}
	for _, conn := range m.conns {
		if conn == nil {
			continue
		}
		if tcpAddr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
			if tcpAddr.IP.Equal(controllerResolvedAddr.IP) && tcpAddr.Port == controllerResolvedAddr.Port {
				return conn.DeleteTopics(topics...)
			}
		}
	}
	conn, err := m.dialer.DialContext(context.Background(), "tcp", controllerAddr)
	if err != nil {
		return err
	}
	m.conns = append(m.conns, conn)
	return conn.DeleteTopics(topics...)
}

func (m *multiConn) Close() error {
	var err error
	for _, conn := range m.conns {
		if conn != nil {
			err = errors.Join(err, conn.Close())
		}
	}
	return err
}

func validateConfigs(conf *Config) error {
	if err := validateRequiredFields(conf); err != nil {
		return err
	}
	setDefaultSecurityProtocol(conf)
	if err := validateSASLConfigs(conf); err != nil {
		return err
	}
	if err := validateTLSConfigs(conf); err != nil {
		return err
	}
	return validateSecurityProtocol(conf)
}

func validateRequiredFields(conf *Config) error {
	if len(conf.Brokers) == 0 {
		return errBrokerNotProvided
	}
	if conf.BatchSize <= 0 {
		return fmt.Errorf("batch size must be greater than 0: %w", errBatchSize)
	}
	if conf.BatchBytes <= 0 {
		return fmt.Errorf("batch bytes must be greater than 0: %w", errBatchBytes)
	}
	if conf.BatchTimeout <= 0 {
		return fmt.Errorf("batch timeout must be greater than 0: %w", errBatchTimeout)
	}
	return nil
}

func (k *kafkaClient) retryConnect(ctx context.Context) {
	for {
		time.Sleep(defaultRetryTimeout)
		if err := k.initialize(ctx); err != nil {
			brokers := k.config.Brokers
			if len(brokers) == 1 {
				fmt.Printf("Retrying connection to Kafka at '%v'...\n", brokers[0])
			} else {
				fmt.Printf("Retrying connection to Kafka at '%v'...\n", brokers)
			}
			continue
		}
		return
	}
}

func (k *kafkaClient) isConnected() bool {
	if k.conn == nil {
		return false
	}
	_, err := k.conn.Controller()
	return err == nil
}

func setupDialer(conf *Config) (*kafka.Dialer, error) {
	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
	}
	if conf.SecurityProtocol == protocolSASL || conf.SecurityProtocol == protocolSASLSSL {
		mechanism, err := getSASLMechanism(conf.SASLMechanism, conf.SASLUser, conf.SASLPassword)
		if err != nil {
			return nil, err
		}
		dialer.SASLMechanism = mechanism
	}
	if conf.SecurityProtocol == protocolSSL || conf.SecurityProtocol == protocolSASLSSL {
		tlsConfig, err := createTLSConfig(&conf.TLS)
		if err != nil {
			return nil, err
		}
		dialer.TLS = tlsConfig
	}
	return dialer, nil
}

func connectToBrokers(ctx context.Context, brokers []string, dialer *kafka.Dialer) ([]Connection, error) {
	if len(brokers) == 0 {
		return nil, errBrokerNotProvided
	}
	var conns []Connection
	for _, broker := range brokers {
		conn, err := dialer.DialContext(ctx, "tcp", broker)
		if err != nil {
			continue
		}
		conns = append(conns, conn)
	}
	if len(conns) == 0 {
		return nil, errFailedToConnectBrokers
	}
	return conns, nil
}

func createKafkaWriter(conf *Config, dialer *kafka.Dialer) Writer {
	return kafka.NewWriter(kafka.WriterConfig{
		Brokers:      conf.Brokers,
		Dialer:       dialer,
		BatchSize:    conf.BatchSize,
		BatchBytes:   conf.BatchBytes,
		BatchTimeout: time.Duration(conf.BatchTimeout),
	})
}

func setDefaultSecurityProtocol(conf *Config) {
	if conf.SecurityProtocol == "" {
		conf.SecurityProtocol = protocolPlainText
	}
}

func validateSecurityProtocol(conf *Config) error {
	switch strings.ToUpper(conf.SecurityProtocol) {
	case protocolPlainText, protocolSASL, protocolSASLSSL, protocolSSL:
		return nil
	default:
		return fmt.Errorf("unsupported security protocol: %s: %w", conf.SecurityProtocol, errUnsupportedSecurityProtocol)
	}
}

func getSASLMechanism(mechanism, username, password string) (sasl.Mechanism, error) {
	switch strings.ToUpper(mechanism) {
	case "PLAIN":
		return plain.Mechanism{Username: username, Password: password}, nil
	case "SCRAM-SHA-256":
		mech, _ := scram.Mechanism(scram.SHA256, username, password)
		return mech, nil
	case "SCRAM-SHA-512":
		mech, _ := scram.Mechanism(scram.SHA512, username, password)
		return mech, nil
	default:
		return nil, fmt.Errorf("%w: %s", errUnsupportedSASLMechanism, mechanism)
	}
}

func validateSASLConfigs(conf *Config) error {
	protocol := strings.ToUpper(conf.SecurityProtocol)
	if protocol == protocolSASL || protocol == protocolSASLSSL {
		if conf.SASLMechanism == "" || conf.SASLUser == "" || conf.SASLPassword == "" {
			return fmt.Errorf("SASL credentials missing: %w", errSASLCredentialsMissing)
		}
	}
	return nil
}

func createTLSConfig(tlsConf *TLSConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: tlsConf.InsecureSkipVerify,
	}
	if tlsConf.CACertFile != "" {
		caCert, err := os.ReadFile(tlsConf.CACertFile)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", errCACertFileRead, err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
	}
	if tlsConf.CertFile != "" && tlsConf.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(tlsConf.CertFile, tlsConf.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", errClientCertLoad, err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	return tlsConfig, nil
}

func validateTLSConfigs(conf *Config) error {
	protocol := strings.ToUpper(conf.SecurityProtocol)
	if protocol == protocolSSL || protocol == protocolSASLSSL {
		if conf.TLS.CACertFile == "" && !conf.TLS.InsecureSkipVerify && conf.TLS.CertFile == "" {
			return fmt.Errorf("for %s, provide either CA cert, client certs, or enable insecure mode: %w",
				protocol, errUnsupportedSecurityProtocol)
		}
	}
	return nil
}

func (k *kafkaClient) Publish(parentCtx context.Context, topic string, message []byte) error {
	ctx, span := otel.GetTracerProvider().Tracer("gokp").Start(parentCtx, "kafka-publish")
	defer span.End()
	if k.writer == nil || topic == "" {
		return errPublisherNotConfigured
	}
	err := k.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Value: message,
		Time:  time.Now(),
	})
	if err != nil {
		return err
	}
	var hostName string
	if len(k.config.Brokers) > 1 {
		hostName = messageMultipleBrokers
	} else {
		hostName = k.config.Brokers[0]
	}
	fmt.Println("hostName", hostName, "k.config.Brokers", span.SpanContext().TraceID().String())
	return nil
}

func (k *kafkaClient) Subscribe(parentCtx context.Context, topic string) (*Message, error) {
	if !k.isConnected() {
		time.Sleep(defaultRetryTimeout)
		return nil, errClientNotConnected
	}
	if k.config.ConsumerGroupID == "" {
		return &Message{}, ErrConsumerGroupNotProvided
	}

	ctx, span := otel.GetTracerProvider().Tracer("go-kp").Start(parentCtx, "kafka-subscribe")
	defer span.End()
	k.mu.Lock()
	if k.reader == nil {
		k.reader = make(map[string]Reader)
	}
	if k.reader[topic] == nil {
		k.reader[topic] = k.getNewReader(topic)
	}
	k.mu.Unlock()
	reader := k.reader[topic]
	msg, err := reader.FetchMessage(ctx)
	if err != nil {
		return nil, err
	}
	m := NewMessage(ctx, msg)
	m.Value = msg.Value
	m.Topic = topic
	m.Committer = newKafkaMessage(&msg, k.reader[topic])

	return m, err
}

func (k *kafkaClient) Close() (err error) {
	for _, r := range k.reader {
		err = errors.Join(err, r.Close())
	}
	if k.writer != nil {
		err = errors.Join(err, k.writer.Close())
	}
	if k.conn != nil {
		err = errors.Join(err, k.conn.Close())
	}
	return err
}

func newKafkaMessage(msg *kafka.Message, reader Reader) *kafkaMessage {
	return &kafkaMessage{msg: msg, reader: reader}
}

func (kmsg *kafkaMessage) Commit() {
	if kmsg.reader != nil {
		_ = kmsg.reader.CommitMessages(context.Background(), *kmsg.msg)
	}
}
