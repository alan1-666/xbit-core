package streambridge

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	publisher Publisher
	now       func() time.Time

	mu        sync.RWMutex
	seq       int64
	events    map[string][]Envelope
	maxEvents int
}

func NewService(publisher Publisher) *Service {
	if publisher == nil {
		publisher = NewMemoryPublisher()
	}
	return &Service{
		publisher: publisher,
		now:       time.Now,
		events:    map[string][]Envelope{},
		maxEvents: 200,
	}
}

func (s *Service) Publish(ctx context.Context, event Event) (PublishResult, error) {
	event = normalizeEvent(event, s.now().UTC())
	if event.Type == "" {
		return PublishResult{}, fmt.Errorf("event type is required")
	}
	topics := s.resolveTopics(event)
	if len(topics) == 0 {
		return PublishResult{}, fmt.Errorf("no topic resolved for event type %q", event.Type)
	}

	result := PublishResult{EventID: event.ID, Topics: topics, Envelopes: make([]Envelope, 0, len(topics))}
	for _, topic := range topics {
		envelope := s.nextEnvelope(topic, event)
		payload, err := EncodeEnvelope(envelope)
		if err != nil {
			return PublishResult{}, err
		}
		if err := s.publisher.Publish(ctx, topic, payload, event.QoS, event.Retain); err != nil {
			return PublishResult{}, err
		}
		s.remember(envelope)
		result.Envelopes = append(result.Envelopes, envelope)
	}
	return result, nil
}

func (s *Service) PublishBatch(ctx context.Context, events []Event) ([]PublishResult, error) {
	results := make([]PublishResult, 0, len(events))
	for _, event := range events {
		result, err := s.Publish(ctx, event)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func (s *Service) Recent(topic string, limit int) []Envelope {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > s.maxEvents {
		limit = 50
	}
	events := s.events[topic]
	if len(events) > limit {
		events = events[len(events)-limit:]
	}
	out := make([]Envelope, len(events))
	copy(out, events)
	return out
}

func (s *Service) Topics() []TopicSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	topics := make([]TopicSnapshot, 0, len(s.events))
	for topic, events := range s.events {
		snapshot := TopicSnapshot{Topic: topic, Events: len(events)}
		if len(events) > 0 {
			last := events[len(events)-1]
			at := time.Unix(last.TS, 0).UTC()
			snapshot.LastSeq = last.Seq
			snapshot.LastEventAt = &at
		}
		topics = append(topics, snapshot)
	}
	sort.Slice(topics, func(i, j int) bool {
		return topics[i].Topic < topics[j].Topic
	})
	return topics
}

func (s *Service) Close() {
	s.publisher.Close()
}

func (s *Service) nextEnvelope(topic string, event Event) Envelope {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	return Envelope{
		ID:          event.ID,
		Topic:       topic,
		Type:        event.Type,
		Source:      event.Source,
		AggregateID: event.AggregateID,
		UserID:      event.UserID,
		ChainID:     event.ChainID,
		Token:       event.Token,
		TS:          event.CreatedAt.Unix(),
		Seq:         s.seq,
		Payload:     event.Payload,
	}
}

func (s *Service) remember(envelope Envelope) {
	s.mu.Lock()
	defer s.mu.Unlock()
	events := append(s.events[envelope.Topic], envelope)
	if len(events) > s.maxEvents {
		events = events[len(events)-s.maxEvents:]
	}
	s.events[envelope.Topic] = events
}

func (s *Service) resolveTopics(event Event) []string {
	if event.Topic != "" {
		return []string{event.Topic}
	}
	chain := topicPart(event.ChainID)
	token := topicPart(event.Token)
	switch event.Type {
	case EventMarketTokenCreated:
		topics := []string{"public/token/new"}
		if isMemeEvent(event) {
			topics = append(topics, "public/meme/new")
		}
		return topics
	case EventMarketTokenUpdated:
		if chain == "" || token == "" {
			return nil
		}
		return []string{"public/meme/token_info/" + chain + "/" + token, "public/token_statistic/" + chain + "/" + token}
	case EventMarketStatisticUpdate:
		if chain == "" || token == "" {
			return nil
		}
		return []string{"public/token_statistic/" + chain + "/" + token}
	case EventMarketTransactionNew:
		if chain == "" || token == "" {
			return nil
		}
		return []string{"public/transaction/new/" + chain + "/" + token}
	case EventMarketOHLCUpdated:
		if token == "" {
			return nil
		}
		bucket := topicPart(event.Bucket)
		if bucket == "" {
			bucket = "1m"
		}
		return []string{"public/kline/ohlc_" + bucket + "/" + token}
	case EventNetworkFeeUpdated:
		if chain == "" {
			return nil
		}
		return []string{"public/network_fee_updated/" + chain}
	case EventTradingOrderFailed:
		if event.UserID == "" {
			return nil
		}
		return []string{"users/" + topicPart(event.UserID) + "/order_submit_failed"}
	case EventTradingOrderConfirmed:
		if event.UserID == "" {
			return nil
		}
		return []string{"users/" + topicPart(event.UserID) + "/order_confirmation"}
	case EventTradingOrderUpdated:
		if event.UserID == "" {
			return nil
		}
		return []string{"users/" + topicPart(event.UserID) + "/order_updated"}
	default:
		return nil
	}
}

func normalizeEvent(event Event, now time.Time) Event {
	event.Type = strings.TrimSpace(event.Type)
	event.Source = strings.TrimSpace(event.Source)
	event.Topic = strings.Trim(event.Topic, "/ ")
	event.AggregateID = strings.TrimSpace(event.AggregateID)
	event.UserID = strings.TrimSpace(event.UserID)
	event.ChainID = normalizeChain(event.ChainID)
	event.Token = strings.TrimSpace(event.Token)
	event.Bucket = strings.TrimSpace(event.Bucket)
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.Payload == nil {
		event.Payload = map[string]any{}
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}
	if event.QoS > 2 {
		event.QoS = 1
	}
	return event
}

func normalizeChain(chain string) string {
	chain = strings.TrimSpace(chain)
	switch strings.ToUpper(chain) {
	case "501", "SOL", "SOLANA":
		return "501"
	case "1", "ETH", "EVM", "ETHEREUM":
		return "1"
	case "56", "BSC":
		return "56"
	case "10143", "MON", "MONAD":
		return "10143"
	default:
		return chain
	}
}

func topicPart(value string) string {
	value = strings.Trim(value, "/ ")
	value = strings.ReplaceAll(value, "#", "")
	value = strings.ReplaceAll(value, "+", "")
	return value
}

func isMemeEvent(event Event) bool {
	category, _ := event.Payload["category"].(string)
	if strings.EqualFold(category, "meme") {
		return true
	}
	if value, ok := event.Payload["isMeme"].(bool); ok {
		return value
	}
	return false
}

func ChainString(chainID int) string {
	if chainID == 0 {
		return ""
	}
	return strconv.Itoa(chainID)
}
