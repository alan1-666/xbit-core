package marketdata

import "time"

type Token struct {
	Address        string         `json:"address"`
	ChainID        int            `json:"chainId"`
	Symbol         string         `json:"symbol"`
	Name           string         `json:"name"`
	Decimals       int            `json:"decimals"`
	LogoURL        string         `json:"logoUrl"`
	Price          string         `json:"price"`
	Price24hChange string         `json:"price24hChange"`
	MarketCap      string         `json:"marketCap"`
	Liquidity      string         `json:"liquidity"`
	Volume24h      string         `json:"volume24h"`
	Holders        int64          `json:"holders"`
	Dexes          []string       `json:"dexes"`
	Category       string         `json:"category"`
	Metadata       map[string]any `json:"metadata"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

type TokenFilter struct {
	Query    string
	Category string
	ChainID  int
	Page     int
	Limit    int
}

type TokenList struct {
	Data  []Token `json:"data"`
	Page  int     `json:"page"`
	Limit int     `json:"limit"`
	Total int     `json:"total"`
}

type OHLC struct {
	TS          int64  `json:"ts"`
	Token       string `json:"token"`
	Open        string `json:"open"`
	High        string `json:"high"`
	Low         string `json:"low"`
	Close       string `json:"close"`
	Price       string `json:"price"`
	TokenVolume string `json:"tokenVolume"`
	USDVolume   string `json:"usdVolume"`
}

type Transaction struct {
	ID                      string         `json:"id"`
	Timestamp               int64          `json:"timestamp"`
	ChainID                 int            `json:"chainId"`
	TxHash                  string         `json:"txHash"`
	LogIndex                int            `json:"logIndex"`
	EventIndex              int            `json:"eventIndex"`
	BaseToken               string         `json:"baseToken"`
	QuoteToken              string         `json:"quoteToken"`
	Pair                    string         `json:"pair"`
	Type                    string         `json:"type"`
	Maker                   string         `json:"maker"`
	BaseAmount              string         `json:"baseAmount"`
	QuoteAmount             string         `json:"quoteAmount"`
	Price                   string         `json:"price"`
	USDAmount               string         `json:"usdAmount"`
	USDPrice                string         `json:"usdPrice"`
	Liquidity               string         `json:"liquidity"`
	HolderPct               string         `json:"holderPct"`
	IsInsider               bool           `json:"isInsider"`
	IsNativeWallet          bool           `json:"isNativeWallet"`
	IsHugeValue             bool           `json:"isHugeValue"`
	IsWhale                 bool           `json:"isWhale"`
	IsDev                   bool           `json:"isDev"`
	IsFreshWallet           bool           `json:"isFreshWallet"`
	IsNewActivity           bool           `json:"isNewActivity"`
	IsPoolContract          bool           `json:"isPoolContract"`
	IsKOL                   bool           `json:"isKOL"`
	IsSmartMoney            bool           `json:"isSmartMoney"`
	IsTopTrader             bool           `json:"isTopTrader"`
	IsSniper                bool           `json:"isSniper"`
	IsBundler               bool           `json:"isBundler"`
	TotalSupply             string         `json:"totalSupply"`
	Decimals                int            `json:"decimals"`
	Tx24h                   int            `json:"tx24h"`
	HoldingProgress         string         `json:"holdingProgress"`
	NativeAmount            string         `json:"nativeAmount"`
	NativePrice             string         `json:"nativePrice"`
	IsSingleSideTransaction bool           `json:"isSingleSideTransaction"`
	Dex                     string         `json:"dex"`
	MakerAlias              string         `json:"MakerAlias"`
	TotalFee                string         `json:"totalFee"`
	TotalFeeUSD             string         `json:"totalFeeUSD"`
	IsKlineTx               bool           `json:"isKlineTx"`
	ReasonFiltering         string         `json:"reasonFiltering"`
	Metadata                map[string]any `json:"metadata"`
}

type Pool struct {
	Address            string    `json:"address"`
	ChainID            int       `json:"chainId"`
	BaseToken          string    `json:"baseToken"`
	QuoteToken         string    `json:"quoteToken"`
	QuoteTokenPrice    string    `json:"quoteTokenPrice"`
	QuoteSymbol        string    `json:"quoteSymbol"`
	BaseSymbol         string    `json:"baseSymbol"`
	BaseTokenLiquidity string    `json:"baseTokenLiquidity"`
	QuoteLiquidity     string    `json:"quoteLiquidity"`
	USDLiquidity       string    `json:"usdLiquidity"`
	Dex                string    `json:"dex"`
	CreatedAt          time.Time `json:"createdAt"`
}

type Category struct {
	ID               string  `json:"categoryId"`
	Name             string  `json:"name"`
	MarketCap        string  `json:"marketCap"`
	Volume24h        string  `json:"volume24h"`
	Price24hChange   string  `json:"price24hChange"`
	PriceUpCount     int     `json:"priceUpCount"`
	PriceDownCount   int     `json:"priceDownCount"`
	TokensCount      int     `json:"tokensCount"`
	TopGainers       []Token `json:"topGainers"`
	Top1TokenSymbol  string  `json:"top1TokenSymbol"`
	Top1TokenAddress string  `json:"top1TokenAddress"`
	Top1TokenName    string  `json:"top1TokenName"`
	Top1TokenLogo    string  `json:"top1TokenLogo"`
	Top1TokenChange  string  `json:"top1TokenP24hChange"`
}

type Checkpoint struct {
	Source      string    `json:"source"`
	Cursor      string    `json:"cursor"`
	BlockNumber int64     `json:"blockNumber"`
	EventTS     int64     `json:"eventTs"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type UpsertTokenInput struct {
	Token
}

type AppendTransactionInput struct {
	Transaction
}
