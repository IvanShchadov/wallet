package ledger_test

import (
	"errors"
	"testing"

	"github.com/IvanShchadov/wallet/internal/ledger"
	"github.com/IvanShchadov/wallet/internal/money"
)

// newBook returns a ledger with the accounts used below:
// external and exchange belong to the system, alice and bob are customers.
func newBook(t *testing.T) *ledger.Ledger {
	t.Helper()
	l := ledger.New()
	accounts := []struct {
		id   ledger.AccountID
		kind ledger.AccountKind
	}{
		{"external", ledger.System},
		{"exchange", ledger.System},
		{"alice", ledger.User},
		{"bob", ledger.User},
	}
	for _, a := range accounts {
		if err := l.Open(a.id, a.kind); err != nil {
			t.Fatalf("Open(%s): unexpected error: %v", a.id, err)
		}
	}
	return l
}

// entry is a short way to write one entry.
func entry(t *testing.T, account ledger.AccountID, amount int64, currency money.Currency) ledger.Entry {
	t.Helper()
	return ledger.Entry{Account: account, Amount: m(t, amount, currency)}
}

// mustTx builds a checked transaction or stops the test.
func mustTx(t *testing.T, entries ...ledger.Entry) ledger.Transaction {
	t.Helper()
	tx, err := ledger.NewTransaction(entries...)
	if err != nil {
		t.Fatalf("NewTransaction: unexpected error: %v", err)
	}
	return tx
}

// mustBalance reads a balance in minor units or stops the test.
func mustBalance(t *testing.T, l *ledger.Ledger, id ledger.AccountID, currency money.Currency) int64 {
	t.Helper()
	b, err := l.Balance(id, currency)
	if err != nil {
		t.Fatalf("Balance(%s, %s): unexpected error: %v", id, currency, err)
	}
	if b.Currency() != currency {
		t.Fatalf("Balance(%s, %s) came back in %s", id, currency, b.Currency())
	}
	return b.Amount()
}

// deposit brings money to a customer from the outside world.
func deposit(t *testing.T, l *ledger.Ledger, id ledger.AccountID, amount int64, currency money.Currency) {
	t.Helper()
	tx := mustTx(t, entry(t, "external", -amount, currency), entry(t, id, amount, currency))
	if err := l.Post(tx); err != nil {
		t.Fatalf("deposit of %d %s to %s: unexpected error: %v", amount, currency, id, err)
	}
}

func TestOpenTwice(t *testing.T) {
	l := ledger.New()
	if err := l.Open("alice", ledger.User); err != nil {
		t.Fatalf("first Open: unexpected error: %v", err)
	}
	if err := l.Open("alice", ledger.User); !errors.Is(err, ledger.ErrAccountExists) {
		t.Errorf("second Open: err = %v, want ErrAccountExists", err)
	}
}

func TestDeposit(t *testing.T) {
	l := newBook(t)
	deposit(t, l, "alice", 10000, money.USD)

	if got := mustBalance(t, l, "alice", money.USD); got != 10000 {
		t.Errorf("alice USD = %d, want 10000", got)
	}
	// external is a system account: it may go below zero, this is where outside money comes from.
	if got := mustBalance(t, l, "external", money.USD); got != -10000 {
		t.Errorf("external USD = %d, want -10000", got)
	}
	if got := mustBalance(t, l, "alice", money.EUR); got != 0 {
		t.Errorf("alice EUR = %d, want 0: each currency has its own balance", got)
	}
}

func TestTransfer(t *testing.T) {
	l := newBook(t)
	deposit(t, l, "alice", 10000, money.USD)

	if err := l.Post(mustTx(t, entry(t, "alice", -3000, money.USD), entry(t, "bob", 3000, money.USD))); err != nil {
		t.Fatalf("transfer: unexpected error: %v", err)
	}
	if got := mustBalance(t, l, "alice", money.USD); got != 7000 {
		t.Errorf("alice USD = %d, want 7000", got)
	}
	if got := mustBalance(t, l, "bob", money.USD); got != 3000 {
		t.Errorf("bob USD = %d, want 3000", got)
	}
}

func TestInsufficientFunds(t *testing.T) {
	l := newBook(t)
	deposit(t, l, "alice", 10000, money.USD)

	err := l.Post(mustTx(t, entry(t, "alice", -15000, money.USD), entry(t, "bob", 15000, money.USD)))
	if !errors.Is(err, ledger.ErrInsufficientFunds) {
		t.Fatalf("sending 150 with 100 on the account: err = %v, want ErrInsufficientFunds", err)
	}
	// All or nothing: a rejected transaction leaves no trace.
	if got := mustBalance(t, l, "alice", money.USD); got != 10000 {
		t.Errorf("alice USD = %d, want 10000", got)
	}
	if got := mustBalance(t, l, "bob", money.USD); got != 0 {
		t.Errorf("bob USD = %d, want 0: bob got money from a rejected transaction", got)
	}
}

func TestSameAccountTwiceInOneTransaction(t *testing.T) {
	l := newBook(t)
	deposit(t, l, "alice", 10000, money.USD)

	// Each entry alone fits into 100, together they take 160.
	err := l.Post(mustTx(t,
		entry(t, "alice", -8000, money.USD),
		entry(t, "alice", -8000, money.USD),
		entry(t, "bob", 16000, money.USD),
	))
	if !errors.Is(err, ledger.ErrInsufficientFunds) {
		t.Fatalf("err = %v, want ErrInsufficientFunds: check the total change of each account, not every entry alone", err)
	}
	if got := mustBalance(t, l, "alice", money.USD); got != 10000 {
		t.Errorf("alice USD = %d, want 10000", got)
	}
}

func TestFundsAreCheckedPerCurrency(t *testing.T) {
	l := newBook(t)
	deposit(t, l, "alice", 10000, money.USD)

	// alice exchanges all her dollars to dirhams
	exchange := mustTx(t,
		entry(t, "alice", -10000, money.USD),
		entry(t, "exchange", 10000, money.USD),
		entry(t, "exchange", -36725, money.AED),
		entry(t, "alice", 36725, money.AED),
	)
	if err := l.Post(exchange); err != nil {
		t.Fatalf("exchange: unexpected error: %v", err)
	}
	if got := mustBalance(t, l, "alice", money.AED); got != 36725 {
		t.Errorf("alice AED = %d, want 36725", got)
	}

	// Now she has dirhams, but not a single cent.
	err := l.Post(mustTx(t, entry(t, "alice", -100, money.USD), entry(t, "bob", 100, money.USD)))
	if !errors.Is(err, ledger.ErrInsufficientFunds) {
		t.Errorf("sending 1 USD with 0 USD and 367.25 AED: err = %v, want ErrInsufficientFunds", err)
	}
}

func TestUnknownAccount(t *testing.T) {
	l := newBook(t)
	deposit(t, l, "alice", 10000, money.USD)

	err := l.Post(mustTx(t, entry(t, "alice", -3000, money.USD), entry(t, "ghost", 3000, money.USD)))
	if !errors.Is(err, ledger.ErrUnknownAccount) {
		t.Errorf("Post to ghost: err = %v, want ErrUnknownAccount", err)
	}
	if got := mustBalance(t, l, "alice", money.USD); got != 10000 {
		t.Errorf("alice USD = %d, want 10000: a rejected transaction changed the balance", got)
	}
	if _, err := l.Balance("ghost", money.USD); !errors.Is(err, ledger.ErrUnknownAccount) {
		t.Errorf("Balance(ghost): err = %v, want ErrUnknownAccount", err)
	}
}

func TestBalanceUnknownCurrency(t *testing.T) {
	l := newBook(t)
	if _, err := l.Balance("alice", "RUB"); !errors.Is(err, money.ErrUnknownCurrency) {
		t.Errorf("Balance(alice, RUB): err = %v, want money.ErrUnknownCurrency", err)
	}
}

func TestPostZeroValueTransaction(t *testing.T) {
	l := newBook(t)
	// Any Go type can be made without its constructor, as a zero value.
	if err := l.Post(ledger.Transaction{}); !errors.Is(err, ledger.ErrTooFewEntries) {
		t.Errorf("Post(Transaction{}): err = %v, want ErrTooFewEntries", err)
	}
}

func TestZeroKindIsUser(t *testing.T) {
	l := newBook(t)
	// A forgotten kind must be the safe one: the zero value of AccountKind is User.
	var kind ledger.AccountKind
	if err := l.Open("carol", kind); err != nil {
		t.Fatalf("Open(carol): unexpected error: %v", err)
	}
	err := l.Post(mustTx(t, entry(t, "carol", -100, money.USD), entry(t, "bob", 100, money.USD)))
	if !errors.Is(err, ledger.ErrInsufficientFunds) {
		t.Errorf("carol, opened with the zero kind, went below zero: err = %v, want ErrInsufficientFunds", err)
	}
}
