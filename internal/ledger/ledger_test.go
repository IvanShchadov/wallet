// External test package: the tests use ledger only through its exported API.
package ledger_test

import (
	"errors"
	"testing"

	"github.com/IvanShchadov/wallet/internal/ledger"
	"github.com/IvanShchadov/wallet/internal/money"
)

// m builds Money or stops the test: it keeps the cases below short.
func m(t *testing.T, amount int64, currency money.Currency) money.Money {
	t.Helper()
	v, err := money.New(amount, currency)
	if err != nil {
		t.Fatalf("money.New(%d, %q): %v", amount, currency, err)
	}
	return v
}

func TestNewTransactionBalanced(t *testing.T) {
	tests := []struct {
		name    string
		entries []ledger.Entry
	}{
		{"deposit from outside", []ledger.Entry{
			{Account: "external", Amount: m(t, -10000, money.USD)},
			{Account: "alice", Amount: m(t, 10000, money.USD)},
		}},
		{"transfer", []ledger.Entry{
			{Account: "alice", Amount: m(t, -3000, money.USD)},
			{Account: "bob", Amount: m(t, 3000, money.USD)},
		}},
		{"transfer with a fee", []ledger.Entry{
			{Account: "alice", Amount: m(t, -10000, money.USD)},
			{Account: "bob", Amount: m(t, 9900, money.USD)},
			{Account: "fees", Amount: m(t, 100, money.USD)},
		}},
		{"exchange USD to AED", []ledger.Entry{ // balanced in each currency separately
			{Account: "alice", Amount: m(t, -10000, money.USD)},
			{Account: "exchange", Amount: m(t, 10000, money.USD)},
			{Account: "exchange", Amount: m(t, -36725, money.AED)},
			{Account: "alice", Amount: m(t, 36725, money.AED)},
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := ledger.NewTransaction(tc.entries...)
			if err != nil {
				t.Fatalf("NewTransaction: unexpected error: %v", err)
			}
			got := tx.Entries()
			if len(got) != len(tc.entries) {
				t.Fatalf("Entries() has %d entries, want %d", len(got), len(tc.entries))
			}
			for i := range got {
				if got[i] != tc.entries[i] {
					t.Errorf("Entries()[%d] = %v, want %v", i, got[i], tc.entries[i])
				}
			}
		})
	}
}

func TestNewTransactionTooFewEntries(t *testing.T) {
	cases := map[string][]ledger.Entry{
		"no entries": nil,
		"one entry":  {{Account: "alice", Amount: m(t, 3000, money.USD)}},
	}
	for name, entries := range cases {
		if _, err := ledger.NewTransaction(entries...); !errors.Is(err, ledger.ErrTooFewEntries) {
			t.Errorf("%s: err = %v, want ErrTooFewEntries", name, err)
		}
	}
}

func TestNewTransactionZeroAmount(t *testing.T) {
	cases := map[string][]ledger.Entry{
		"zero entry": {
			{Account: "alice", Amount: m(t, -3000, money.USD)},
			{Account: "bob", Amount: m(t, 3000, money.USD)},
			{Account: "carol", Amount: m(t, 0, money.USD)},
		},
		"zero value Money": { // money.Money{} has neither an amount nor a currency
			{Account: "alice", Amount: m(t, -3000, money.USD)},
			{Account: "bob", Amount: m(t, 3000, money.USD)},
			{Account: "carol", Amount: money.Money{}},
		},
	}
	for name, entries := range cases {
		if _, err := ledger.NewTransaction(entries...); !errors.Is(err, ledger.ErrZeroAmount) {
			t.Errorf("%s: err = %v, want ErrZeroAmount", name, err)
		}
	}
}

func TestNewTransactionUnbalanced(t *testing.T) {
	cases := map[string][]ledger.Entry{
		"money lost": {
			{Account: "alice", Amount: m(t, -3000, money.USD)},
			{Account: "bob", Amount: m(t, 2000, money.USD)},
		},
		"money from nowhere": {
			{Account: "alice", Amount: m(t, 3000, money.USD)},
			{Account: "bob", Amount: m(t, 3000, money.USD)},
		},
		"different currencies": { // the total is zero, but neither currency is
			{Account: "alice", Amount: m(t, -10000, money.USD)},
			{Account: "bob", Amount: m(t, 10000, money.EUR)},
		},
	}
	for name, entries := range cases {
		if _, err := ledger.NewTransaction(entries...); !errors.Is(err, ledger.ErrUnbalanced) {
			t.Errorf("%s: err = %v, want ErrUnbalanced", name, err)
		}
	}
}

func TestTransactionKeepsItsOwnCopy(t *testing.T) {
	entries := []ledger.Entry{
		{Account: "alice", Amount: m(t, -3000, money.USD)},
		{Account: "bob", Amount: m(t, 3000, money.USD)},
	}
	tx, err := ledger.NewTransaction(entries...)
	if err != nil {
		t.Fatalf("NewTransaction: unexpected error: %v", err)
	}

	// entries... hands over the caller's slice itself, not a copy.
	// If the transaction keeps that slice, this line breaks an already checked transaction.
	entries[1].Amount = m(t, 999999, money.USD)
	if got := tx.Entries()[1].Amount.Amount(); got != 3000 {
		t.Errorf("after the caller changed its slice, bob's entry = %d, want 3000: NewTransaction must copy the entries", got)
	}

	// The slice from Entries() must not be a window into the transaction either.
	out := tx.Entries()
	out[0].Amount = m(t, 1, money.USD)
	if got := tx.Entries()[0].Amount.Amount(); got != -3000 {
		t.Errorf("after changing the slice from Entries(), alice's entry = %d, want -3000: Entries must return a copy", got)
	}
}
