package money

import (
	"fmt"
	"math/big"
	"strings"
)

func Parse(amount string) (*big.Rat, error) {
	amount = strings.TrimSpace(amount)
	if amount == "" {
		return nil, fmt.Errorf("amount is required")
	}
	value, ok := new(big.Rat).SetString(amount)
	if !ok {
		return nil, fmt.Errorf("invalid decimal amount %q", amount)
	}
	if value.Sign() < 0 {
		return nil, fmt.Errorf("amount must be non-negative")
	}
	return value, nil
}

func MultiplyBps(amount string, bps int64) (string, error) {
	value, err := Parse(amount)
	if err != nil {
		return "", err
	}
	value.Mul(value, big.NewRat(bps, 10_000))
	return Format(value), nil
}

func Sub(a string, b string) (string, error) {
	left, err := Parse(a)
	if err != nil {
		return "", err
	}
	right, err := Parse(b)
	if err != nil {
		return "", err
	}
	left.Sub(left, right)
	if left.Sign() < 0 {
		return "", fmt.Errorf("result would be negative")
	}
	return Format(left), nil
}

func Format(value *big.Rat) string {
	if value == nil {
		return "0"
	}
	s := value.FloatString(18)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" || s == "-0" {
		return "0"
	}
	return s
}
