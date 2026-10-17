package ledger

import (
	"errors"
	"fmt"
	"sync"

	"github.com/IvanShchadov/wallet/internal/money"
)

// AccountKind says whether an account may go below zero.
type AccountKind int

const (
	User   AccountKind = iota // a customer: the balance never goes below zero
	System                    // the wallet's own account (external, exchange, fees): may go below zero
)

var (
	ErrAccountExists     = errors.New("account already exists")
	ErrUnknownAccount    = errors.New("unknown account")
	ErrInsufficientFunds = errors.New("insufficient funds")
)

// Ledger is an append-only book of transactions.
type Ledger struct {
	mu           sync.RWMutex // guards accounts and transactions
	accounts     map[AccountID]AccountKind
	transactions []Transaction
}

// balanceKey identifies one balance: an account in one currency.
type balanceKey struct {
	account  AccountID
	currency money.Currency
}

// New returns an empty ledger.
func New() *Ledger {
	ledger := &Ledger{
		accounts: make(map[AccountID]AccountKind),
	}

	return ledger
}

// Open adds an account to the ledger.
func (l *Ledger) Open(id AccountID, kind AccountKind) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, ok := l.accounts[id]; ok {
		return fmt.Errorf("%w: %s", ErrAccountExists, id)
	}

	l.accounts[id] = kind

	return nil
}

// sum adds up the account's entries in the currency.
// Call it only while holding l.mu.
func (l *Ledger) sum(id AccountID, currency money.Currency) int64 {
	var amount int64

	for _, transaction := range l.transactions {
		for _, entry := range transaction.entries {
			if entry.Account == id && entry.Amount.Currency() == currency {
				amount += entry.Amount.Amount()
			}
		}
	}

	return amount
}

// Post appends a transaction. If it is rejected, the ledger does not change.
func (l *Ledger) Post(tx Transaction) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(tx.entries) < 2 {
		return fmt.Errorf("%w: got %d", ErrTooFewEntries, len(tx.entries))
	}

	changes := make(map[balanceKey]int64)
	for _, entry := range tx.entries {
		if _, ok := l.accounts[entry.Account]; !ok {
			return fmt.Errorf("%w: %s", ErrUnknownAccount, entry.Account)
		}

		changes[balanceKey{entry.Account, entry.Amount.Currency()}] += entry.Amount.Amount()
	}

	for key, change := range changes {
		if l.accounts[key.account] == System {
			continue
		}
		if l.sum(key.account, key.currency)+change < 0 {
			return fmt.Errorf("%w: %s has not enough %s", ErrInsufficientFunds, key.account, key.currency)
		}
	}

	l.transactions = append(l.transactions, tx)
	return nil
}

// Balance returns the sum of the account's entries in the currency.
func (l *Ledger) Balance(id AccountID, currency money.Currency) (money.Money, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if _, ok := l.accounts[id]; !ok {
		return money.Money{}, fmt.Errorf("%w: %s", ErrUnknownAccount, id)
	}

	amount := l.sum(id, currency)

	return money.New(amount, currency)
}
