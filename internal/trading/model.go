package trading

import "time"

const (
	OrderStatusPending   = "pending"
	OrderStatusSubmitted = "submitted"
	OrderStatusConfirmed = "confirmed"
	OrderStatusFailed    = "failed"
	OrderStatusCancelled = "cancelled"

	OrderTypeMarket = "market"
	OrderTypeLimit  = "limit"

	SideBuy  = "buy"
	SideSell = "sell"
)

type QuoteRequest struct {
	UserID      string `json:"userId"`
	ChainType   string `json:"chainType"`
	InputToken  string `json:"inputToken"`
	OutputToken string `json:"outputToken"`
	InputAmount string `json:"inputAmount"`
	SlippageBps int    `json:"slippageBps"`
}

type Quote struct {
	ID                string         `json:"id,omitempty"`
	RouteID           string         `json:"routeId"`
	UserID            string         `json:"userId,omitempty"`
	ChainType         string         `json:"chainType"`
	InputToken        string         `json:"inputToken"`
	OutputToken       string         `json:"outputToken"`
	InputAmount       string         `json:"inputAmount"`
	OutputAmount      string         `json:"outputAmount"`
	MinOutputAmount   string         `json:"minOutputAmount"`
	SlippageBps       int            `json:"slippageBps"`
	PlatformFeeAmount string         `json:"platformFeeAmount"`
	RouteSnapshot     map[string]any `json:"routeSnapshot"`
	ExpiresAt         time.Time      `json:"expiresAt"`
	CreatedAt         time.Time      `json:"createdAt"`
}

type CreateOrderInput struct {
	UserID          string         `json:"userId"`
	ChainType       string         `json:"chainType"`
	WalletAddress   string         `json:"walletAddress"`
	OrderType       string         `json:"orderType"`
	Side            string         `json:"side"`
	InputToken      string         `json:"inputToken"`
	OutputToken     string         `json:"outputToken"`
	InputAmount     string         `json:"inputAmount"`
	SlippageBps     int            `json:"slippageBps"`
	ClientRequestID string         `json:"clientRequestId"`
	RouteSnapshot   map[string]any `json:"routeSnapshot"`
}

type Order struct {
	ID                   string         `json:"id"`
	UserID               string         `json:"userId"`
	ChainType            string         `json:"chainType"`
	WalletAddress        string         `json:"walletAddress"`
	OrderType            string         `json:"orderType"`
	Side                 string         `json:"side"`
	InputToken           string         `json:"inputToken"`
	OutputToken          string         `json:"outputToken"`
	InputAmount          string         `json:"inputAmount"`
	ExpectedOutputAmount string         `json:"expectedOutputAmount"`
	MinOutputAmount      string         `json:"minOutputAmount"`
	SlippageBps          int            `json:"slippageBps"`
	RouteSnapshot        map[string]any `json:"routeSnapshot"`
	Status               string         `json:"status"`
	TxHash               string         `json:"txHash,omitempty"`
	FailureCode          string         `json:"failureCode,omitempty"`
	ClientRequestID      string         `json:"clientRequestId,omitempty"`
	CreatedAt            time.Time      `json:"createdAt"`
	UpdatedAt            time.Time      `json:"updatedAt"`
	FilledAt             *time.Time     `json:"filledAt,omitempty"`
	ExpiredAt            *time.Time     `json:"expiredAt,omitempty"`
}

type SearchOrdersInput struct {
	UserID string `json:"userId"`
	Status string `json:"status"`
	Limit  int    `json:"limit"`
}

type UpdateOrderStatusInput struct {
	Status      string         `json:"status"`
	TxHash      string         `json:"txHash"`
	FailureCode string         `json:"failureCode"`
	Payload     map[string]any `json:"payload"`
}

type NetworkFee struct {
	ChainType        string         `json:"chainType"`
	MaxComputeUnits  int64          `json:"maxComputeUnits,omitempty"`
	PriorityFeePrice map[string]any `json:"priorityFeePrice,omitempty"`
	PlatformFeeBps   int            `json:"platformFeeBps"`
	MinTipFee        string         `json:"minTipFee,omitempty"`
	AutoTipFee       string         `json:"autoTipFee,omitempty"`
	Source           string         `json:"source"`
	CreatedAt        time.Time      `json:"createdAt"`
}
