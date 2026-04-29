package money

import "testing"

func TestMultiplyBps(t *testing.T) {
	got, err := MultiplyBps("100.00", 30)
	if err != nil {
		t.Fatal(err)
	}
	if got != "0.3" {
		t.Fatalf("got %s", got)
	}
}

func TestSub(t *testing.T) {
	got, err := Sub("100", "0.3")
	if err != nil {
		t.Fatal(err)
	}
	if got != "99.7" {
		t.Fatalf("got %s", got)
	}
}

func TestParseRejectsNegative(t *testing.T) {
	if _, err := Parse("-1"); err == nil {
		t.Fatalf("expected error")
	}
}
