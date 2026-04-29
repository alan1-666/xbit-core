package streambridge

import "time"

const (
	EventMarketTokenCreated    = "market.token.created"
	EventMarketTokenUpdated    = "market.token.updated"
	EventMarketStatisticUpdate = "market.token.statistic_updated"
	EventMarketTransactionNew  = "market.transaction.created"
	EventMarketOHLCUpdated     = "market.ohlc.updated"
	EventNetworkFeeUpdated     = "network.fee.updated"
	EventTradingOrderUpdated   = "trading.order.updated"
	EventTradingOrderFailed    = "trading.order.submit_failed"
	EventTradingOrderConfirmed = "trading.order.confirmed"

	EventHypertraderOrderUpdated    = "hypertrader.order.updated"
	EventHypertraderFillCreated     = "hypertrader.fill.created"
	EventHypertraderOpenOrders      = "hypertrader.open_orders.snapshot"
	EventHypertraderAccountUpdated  = "hypertrader.account.updated"
	EventHypertraderPositionUpdated = "hypertrader.position.updated"
	EventHypertraderFundingUpdated  = "hypertrader.funding.updated"
	EventHypertraderLedgerUpdated   = "hypertrader.ledger.updated"
	EventHypertraderRawEvent        = "hypertrader.event"
)

type Event struct {
	ID          string         `json:"id,omitempty"`
	Type        string         `json:"type"`
	Source      string         `json:"source,omitempty"`
	Topic       string         `json:"topic,omitempty"`
	AggregateID string         `json:"aggregateId,omitempty"`
	UserID      string         `json:"userId,omitempty"`
	ChainID     string         `json:"chainId,omitempty"`
	Token       string         `json:"token,omitempty"`
	Bucket      string         `json:"bucket,omitempty"`
	Retain      bool           `json:"retain,omitempty"`
	QoS         byte           `json:"qos,omitempty"`
	Payload     map[string]any `json:"payload"`
	CreatedAt   time.Time      `json:"createdAt,omitempty"`
}

type Envelope struct {
	ID          string         `json:"id"`
	Topic       string         `json:"topic"`
	Type        string         `json:"type"`
	Source      string         `json:"source,omitempty"`
	AggregateID string         `json:"aggregateId,omitempty"`
	UserID      string         `json:"userId,omitempty"`
	ChainID     string         `json:"chainId,omitempty"`
	Token       string         `json:"token,omitempty"`
	TS          int64          `json:"ts"`
	Seq         int64          `json:"seq"`
	Payload     map[string]any `json:"payload"`
}

type PublishResult struct {
	EventID   string     `json:"eventId"`
	Topics    []string   `json:"topics"`
	Envelopes []Envelope `json:"envelopes,omitempty"`
}

type TopicSnapshot struct {
	Topic       string     `json:"topic"`
	LastSeq     int64      `json:"lastSeq"`
	LastEventAt *time.Time `json:"lastEventAt,omitempty"`
	Events      int        `json:"events"`
}

type Config struct {
	BrokerURL string
	ClientID  string
	Username  string
	Password  string
	Enabled   bool
}
