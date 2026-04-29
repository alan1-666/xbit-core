package streambridge

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Publisher interface {
	Publish(ctx context.Context, topic string, payload []byte, qos byte, retained bool) error
	Close()
}

type MemoryPublisher struct {
	mu       sync.RWMutex
	messages []PublishedMessage
}

type PublishedMessage struct {
	Topic    string    `json:"topic"`
	Payload  []byte    `json:"payload"`
	QoS      byte      `json:"qos"`
	Retained bool      `json:"retained"`
	SentAt   time.Time `json:"sentAt"`
}

func NewMemoryPublisher() *MemoryPublisher {
	return &MemoryPublisher{messages: make([]PublishedMessage, 0)}
}

func (p *MemoryPublisher) Publish(_ context.Context, topic string, payload []byte, qos byte, retained bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	copied := append([]byte(nil), payload...)
	p.messages = append(p.messages, PublishedMessage{Topic: topic, Payload: copied, QoS: qos, Retained: retained, SentAt: time.Now().UTC()})
	return nil
}

func (p *MemoryPublisher) Close() {}

func (p *MemoryPublisher) Messages() []PublishedMessage {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]PublishedMessage, len(p.messages))
	copy(out, p.messages)
	return out
}

type MQTTPublisher struct {
	client mqtt.Client
	logger *slog.Logger
}

func NewMQTTPublisher(cfg Config, logger *slog.Logger) (*MQTTPublisher, error) {
	if strings.TrimSpace(cfg.BrokerURL) == "" {
		return nil, fmt.Errorf("mqtt broker url is required")
	}
	if strings.TrimSpace(cfg.ClientID) == "" {
		cfg.ClientID = "xbit-stream-bridge"
	}
	opts := mqtt.NewClientOptions().
		AddBroker(cfg.BrokerURL).
		SetClientID(cfg.ClientID).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(time.Second).
		SetWriteTimeout(10 * time.Second)
	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
		opts.SetPassword(cfg.Password)
	}
	client := mqtt.NewClient(opts)
	token := client.Connect()
	if !token.WaitTimeout(10 * time.Second) {
		return nil, fmt.Errorf("connect mqtt broker timed out")
	}
	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("connect mqtt broker: %w", err)
	}
	return &MQTTPublisher{client: client, logger: logger}, nil
}

func (p *MQTTPublisher) Publish(ctx context.Context, topic string, payload []byte, qos byte, retained bool) error {
	token := p.client.Publish(topic, qos, retained, payload)
	done := make(chan struct{})
	go func() {
		token.Wait()
		close(done)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		if err := token.Error(); err != nil {
			return fmt.Errorf("mqtt publish %s: %w", topic, err)
		}
		return nil
	}
}

func (p *MQTTPublisher) Close() {
	if p.client != nil && p.client.IsConnected() {
		p.client.Disconnect(250)
	}
}

func EncodeEnvelope(envelope Envelope) ([]byte, error) {
	payload, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("marshal stream envelope: %w", err)
	}
	return payload, nil
}
