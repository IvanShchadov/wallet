// External test package: the tests use money only through its exported API,
// the same way the rest of the wallet will.
package money_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/IvanShchadov/wallet/internal/money"
)

// mustNew creates Money or stops the test: it keeps the cases below short.
func mustNew(t *testing.T, amount int64, currency money.Currency) money.Money {
	t.Helper()
	m, err := money.New(amount, currency)
	if err != nil {
		t.Fatalf("New(%d, %q): unexpected error: %v", amount, currency, err)
	}
	return m
}

func TestNew(t *testing.T) {
	for _, c := range []money.Currency{money.USD, money.EUR, money.AED} {
		m, err := money.New(1050, c)
		if err != nil {
			t.Fatalf("New(1050, %q): unexpected error: %v", c, err)
		}
		if m.Amount() != 1050 || m.Currency() != c {
			t.Errorf("New(1050, %q) = %d %s, want 1050 %s", c, m.Amount(), m.Currency(), c)
		}
	}
}

func TestNewUnknownCurrency(t *testing.T) {
	for _, c := range []money.Currency{"", "usd", "XYZ"} {
		if _, err := money.New(100, c); !errors.Is(err, money.ErrUnknownCurrency) {
			t.Errorf("New(100, %q): err = %v, want ErrUnknownCurrency", c, err)
		}
	}
}

func TestAdd(t *testing.T) {
	a := mustNew(t, 1050, money.USD)
	b := mustNew(t, 250, money.USD)

	sum, err := a.Add(b)
	if err != nil {
		t.Fatalf("Add: unexpected error: %v", err)
	}
	if sum.Amount() != 1300 || sum.Currency() != money.USD {
		t.Errorf("1050 USD + 250 USD = %d %s, want 1300 USD", sum.Amount(), sum.Currency())
	}
	// Money is a value object: Add returns a new value and leaves both operands untouched.
	if a.Amount() != 1050 || b.Amount() != 250 {
		t.Errorf("Add changed its operands: a = %d, b = %d, want 1050 and 250", a.Amount(), b.Amount())
	}
}

func TestSub(t *testing.T) {
	tests := []struct{ a, b, want int64 }{
		{1050, 250, 800},
		{100, 250, -150}, // Money may go below zero; whether an account may is the account's rule
		{0, 0, 0},
	}
	for _, tc := range tests {
		a := mustNew(t, tc.a, money.EUR)
		b := mustNew(t, tc.b, money.EUR)

		diff, err := a.Sub(b)
		if err != nil {
			t.Fatalf("Sub: unexpected error: %v", err)
		}
		if diff.Amount() != tc.want || diff.Currency() != money.EUR {
			t.Errorf("%d EUR - %d EUR = %d %s, want %d EUR", tc.a, tc.b, diff.Amount(), diff.Currency(), tc.want)
		}
	}
}

func TestCurrencyMismatch(t *testing.T) {
	usd := mustNew(t, 100, money.USD)
	eur := mustNew(t, 100, money.EUR)

	if _, err := usd.Add(eur); !errors.Is(err, money.ErrCurrencyMismatch) {
		t.Errorf("USD.Add(EUR): err = %v, want ErrCurrencyMismatch", err)
	}
	if _, err := usd.Sub(eur); !errors.Is(err, money.ErrCurrencyMismatch) {
		t.Errorf("USD.Sub(EUR): err = %v, want ErrCurrencyMismatch", err)
	}
}

func TestIsNegative(t *testing.T) {
	tests := []struct {
		amount int64
		want   bool
	}{{-1, true}, {0, false}, {1, false}}
	for _, tc := range tests {
		m := mustNew(t, tc.amount, money.AED)
		if got := m.IsNegative(); got != tc.want {
			t.Errorf("IsNegative() for %d = %v, want %v", tc.amount, got, tc.want)
		}
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		amount   int64
		currency money.Currency
		want     string
	}{
		{1050, money.USD, "10.50 USD"},
		{5, money.EUR, "0.05 EUR"},
		{0, money.AED, "0.00 AED"},
		{100000, money.USD, "1000.00 USD"},
		{-5, money.EUR, "-0.05 EUR"},
		{-150, money.USD, "-1.50 USD"},
		// 2^53 + 1 cents: float64 cannot hold this number exactly and rounds it
		// to ...992, so formatting through float64 prints .92 instead of .93.
		{9_007_199_254_740_993, money.USD, "90071992547409.93 USD"},
	}
	for _, tc := range tests {
		m := mustNew(t, tc.amount, tc.currency)
		if got := m.String(); got != tc.want {
			t.Errorf("String() for %d %s = %q, want %q", tc.amount, tc.currency, got, tc.want)
		}
	}

	// fmt calls String() only if Money (not *Money) implements fmt.Stringer.
	m := mustNew(t, 1050, money.USD)
	if got := fmt.Sprint(m); got != "10.50 USD" {
		t.Errorf("fmt.Sprint(Money) = %q, want %q: is String() defined on *Money instead of Money?", got, "10.50 USD")
	}
}
