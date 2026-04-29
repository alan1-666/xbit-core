package marketdata

import (
	"context"
	"testing"
)

func TestMarketDataServiceReadModel(t *testing.T) {
	svc := NewService(nil)
	ctx := context.Background()

	list, err := svc.ListTokens(ctx, TokenFilter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Data) == 0 || list.Total == 0 {
		t.Fatalf("list = %+v", list)
	}

	token, err := svc.GetToken(ctx, list.Data[0].ChainID, list.Data[0].Address)
	if err != nil {
		t.Fatal(err)
	}
	if token.Symbol == "" || token.Price == "" {
		t.Fatalf("token = %+v", token)
	}

	results, err := svc.SearchTokens(ctx, token.Symbol, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatalf("search returned no tokens")
	}

	points, err := svc.OHLC(ctx, token.ChainID, token.Address, "1m", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 5 || points[0].Close == "" {
		t.Fatalf("ohlc = %+v", points)
	}

	txs, err := svc.Transactions(ctx, token.ChainID, token.Address, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(txs) != 3 || txs[0].TxHash == "" {
		t.Fatalf("txs = %+v", txs)
	}
}

func TestMarketDataIngest(t *testing.T) {
	svc := NewService(NewMemoryStore())
	ctx := context.Background()

	token, err := svc.UpsertToken(ctx, UpsertTokenInput{Token: Token{
		Address:   "new-token",
		ChainID:   501,
		Symbol:    "NEW",
		Name:      "New Token",
		Decimals:  6,
		Price:     "1.23",
		MarketCap: "1230000",
		Liquidity: "100000",
		Volume24h: "50000",
		Category:  "meme",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if token.Address != "new-token" {
		t.Fatalf("token = %+v", token)
	}

	tx, err := svc.AppendTransaction(ctx, AppendTransactionInput{Transaction: Transaction{
		ChainID:     501,
		BaseToken:   "new-token",
		QuoteToken:  "USDC",
		TxHash:      "0xnew",
		Type:        "buy",
		Maker:       "wallet",
		BaseAmount:  "1",
		QuoteAmount: "1.23",
		Price:       "1.23",
		USDAmount:   "1.23",
		USDPrice:    "1.23",
		Liquidity:   "100000",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if tx.ID == "" {
		t.Fatalf("tx id missing")
	}

	checkpoint, err := svc.SaveCheckpoint(ctx, Checkpoint{Source: "solana", Cursor: "abc", BlockNumber: 123})
	if err != nil {
		t.Fatal(err)
	}
	if checkpoint.Source != "solana" {
		t.Fatalf("checkpoint = %+v", checkpoint)
	}
}
