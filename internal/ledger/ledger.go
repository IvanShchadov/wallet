package ledger

import (
	"errors"
	"fmt"

	"github.com/IvanShchadov/wallet/internal/money"
)

// AccountID identifies an account in the ledger.
type AccountID string

// Entry is one posting: a positive amount means money comes into the account,
// a negative amount means money leaves it.
type Entry struct {
	Account AccountID
	Amount  money.Money
}

// Transaction is a checked set of entries that sum to zero in every currency.
// The only way to get one is NewTransaction.
type Transaction struct {
	entries []Entry
}

var (
	ErrTooFewEntries = errors.New("too few entries")
	ErrZeroAmount    = errors.New("zero amount")
	ErrUnbalanced    = errors.New("entries do not sum to zero")
)

// NewTransaction checks the double-entry rules and returns a transaction.
func NewTransaction(entries ...Entry) (Transaction, error) {
	if len(entries) < 2 {
		return Transaction{}, fmt.Errorf("%w: got %d, want at least 2", ErrTooFewEntries, len(entries))
	}

	currencies := make(map[money.Currency]int64)

	for _, entry := range entries {
		if entry.Amount.Amount() == 0 {
			return Transaction{}, fmt.Errorf("%w: account %s", ErrZeroAmount, entry.Account)
		}

		currencies[entry.Amount.Currency()] += entry.Amount.Amount()
	}

	for currency, value := range currencies {
		if value != 0 {
			return Transaction{}, fmt.Errorf("%w: %s is off by %d", ErrUnbalanced, currency, value)
		}
	}

	return Transaction{entries: append([]Entry{}, entries...)}, nil
}

// Entries returns a copy of the transaction's entries.
func (t Transaction) Entries() []Entry {
	return append([]Entry{}, t.entries...)
}
