package money

import (
	"errors"
	"fmt"
)

type Currency string

const (
	USD Currency = "USD"
	EUR Currency = "EUR"
	AED Currency = "AED"
)

var (
	ErrUnknownCurrency  = errors.New("unknown currency")
	ErrCurrencyMismatch = errors.New("currency mismatch")
)

// Money is an amount in minor units of one currency
type Money struct {
	amount   int64
	currency Currency
}

func New(amount int64, currency Currency) (Money, error) {
	switch currency {
	case USD, EUR, AED:
		return Money{amount, currency}, nil
	default:
		return Money{}, fmt.Errorf("%w: %q", ErrUnknownCurrency, currency)
	}
}
func (m Money) Amount() int64 {
	return m.amount
}
func (m Money) Currency() Currency {
	return m.currency
}
func (m Money) Add(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, fmt.Errorf("%w: %s and %s", ErrCurrencyMismatch, m.currency, other.currency)
	}

	result := m.amount + other.amount

	return Money{currency: m.currency, amount: result}, nil
}
func (m Money) Sub(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, fmt.Errorf("%w: %s and %s", ErrCurrencyMismatch, m.currency, other.currency)
	}

	result := m.amount - other.amount

	return Money{currency: m.currency, amount: result}, nil
}
func (m Money) IsNegative() bool {
	return m.amount < 0
}
func (m Money) String() string {
	sign := ""
	val := m.amount

	if val < 0 {
		sign = "-"
		val = -val
	}

	major := val / 100
	minor := val % 100

	return fmt.Sprintf("%s%d.%02d %s", sign, major, minor, m.currency)
}
