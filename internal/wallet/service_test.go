package wallet_test

import (
	"errors"
	"testing"

	"github.com/IvanShchadov/wallet/internal/ledger"
	"github.com/IvanShchadov/wallet/internal/money"
	"github.com/IvanShchadov/wallet/internal/wallet"
)

// *ledger.Ledger must fit the interface the service needs.
// If it does not, this line does not compile.
var _ wallet.Ledger = (*ledger.Ledger)(nil)

// setup returns a service over a fresh ledger with two customers, alice and bob.
// The ledger comes back too, so a test can look at the system account directly.
func setup(t *testing.T) (*wallet.Service, *ledger.Ledger) {
	t.Helper()
	l := ledger.New()
	s, err := wallet.NewService(l)
	if err != nil {
		t.Fatalf("NewService: unexpected error: %v", err)
	}
	for _, id := range []ledger.AccountID{"alice", "bob"} {
		if err := s.OpenAccount(id); err != nil {
			t.Fatalf("OpenAccount(%s): unexpected error: %v", id, err)
		}
	}
	return s, l
}

// usd builds an amount in US cents.
func usd(t *testing.T, cents int64) money.Money {
	t.Helper()
	m, err := money.New(cents, money.USD)
	if err != nil {
		t.Fatalf("money.New(%d, USD): unexpected error: %v", cents, err)
	}
	return m
}

// balance reads a customer's USD balance through the service or stops the test.
func balance(t *testing.T, s *wallet.Service, id ledger.AccountID) int64 {
	t.Helper()
	b, err := s.Balance(id, money.USD)
	if err != nil {
		t.Fatalf("Balance(%s): unexpected error: %v", id, err)
	}
	return b.Amount()
}

// external reads the system account straight from the ledger: the service hides it.
func external(t *testing.T, l *ledger.Ledger) int64 {
	t.Helper()
	b, err := l.Balance("external", money.USD)
	if err != nil {
		t.Fatalf("ledger Balance(external): unexpected error: %v", err)
	}
	return b.Amount()
}

func deposit(t *testing.T, s *wallet.Service, id ledger.AccountID, cents int64) {
	t.Helper()
	if err := s.Deposit(id, usd(t, cents)); err != nil {
		t.Fatalf("Deposit(%s, %d): unexpected error: %v", id, cents, err)
	}
}

func TestNewAccountIsEmpty(t *testing.T) {
	s, _ := setup(t)
	if got := balance(t, s, "alice"); got != 0 {
		t.Errorf("new account balance = %d, want 0", got)
	}
}

func TestOpenAccountTwice(t *testing.T) {
	s, _ := setup(t)
	if err := s.OpenAccount("alice"); !errors.Is(err, ledger.ErrAccountExists) {
		t.Errorf("OpenAccount(alice) again: err = %v, want ErrAccountExists", err)
	}
}

func TestOpenAccountCannotTakeSystemID(t *testing.T) {
	s, _ := setup(t)
	if err := s.OpenAccount("external"); !errors.Is(err, ledger.ErrAccountExists) {
		t.Errorf("OpenAccount(external): err = %v, want ErrAccountExists: this id belongs to the system account", err)
	}
}

func TestNewServiceOverTheSameLedger(t *testing.T) {
	l := ledger.New()
	if _, err := wallet.NewService(l); err != nil {
		t.Fatalf("first NewService: unexpected error: %v", err)
	}
	// The system account is already there. That is fine: after a restart
	// the service starts again over the same data.
	s, err := wallet.NewService(l)
	if err != nil {
		t.Fatalf("second NewService over the same ledger: unexpected error: %v", err)
	}
	if err := s.OpenAccount("alice"); err != nil {
		t.Fatalf("OpenAccount(alice): unexpected error: %v", err)
	}
	deposit(t, s, "alice", 100)
	if got := balance(t, s, "alice"); got != 100 {
		t.Errorf("alice USD = %d, want 100", got)
	}
}

func TestDeposit(t *testing.T) {
	s, l := setup(t)
	deposit(t, s, "alice", 10000)

	if got := balance(t, s, "alice"); got != 10000 {
		t.Errorf("alice USD = %d, want 10000", got)
	}
	if got := external(t, l); got != -10000 {
		t.Errorf("external USD = %d, want -10000: a deposit takes money from the external account", got)
	}
}

func TestDepositRejectsBadAmount(t *testing.T) {
	s, l := setup(t)
	for _, amount := range []money.Money{usd(t, 0), usd(t, -500), {}} {
		if err := s.Deposit("alice", amount); !errors.Is(err, wallet.ErrInvalidAmount) {
			t.Errorf("Deposit(alice, %v): err = %v, want ErrInvalidAmount", amount, err)
		}
	}
	if got := balance(t, s, "alice"); got != 0 {
		t.Errorf("alice USD = %d, want 0: a rejected deposit must change nothing", got)
	}
	if got := external(t, l); got != 0 {
		t.Errorf("external USD = %d, want 0: a rejected deposit must change nothing", got)
	}
}

func TestDepositUnknownAccount(t *testing.T) {
	s, _ := setup(t)
	if err := s.Deposit("carol", usd(t, 100)); !errors.Is(err, ledger.ErrUnknownAccount) {
		t.Errorf("Deposit(carol): err = %v, want ErrUnknownAccount", err)
	}
}

func TestTransfer(t *testing.T) {
	s, _ := setup(t)
	deposit(t, s, "alice", 10000)

	if err := s.Transfer("alice", "bob", usd(t, 3000)); err != nil {
		t.Fatalf("Transfer(alice → bob, 3000): unexpected error: %v", err)
	}
	if got := balance(t, s, "alice"); got != 7000 {
		t.Errorf("alice USD = %d, want 7000", got)
	}
	if got := balance(t, s, "bob"); got != 3000 {
		t.Errorf("bob USD = %d, want 3000", got)
	}
}

func TestTransferInsufficientFunds(t *testing.T) {
	s, _ := setup(t)
	deposit(t, s, "alice", 1000)

	err := s.Transfer("alice", "bob", usd(t, 3000))
	if !errors.Is(err, ledger.ErrInsufficientFunds) {
		t.Errorf("Transfer(alice → bob, 3000) with 1000 on the account: err = %v, want ErrInsufficientFunds", err)
	}
	if got := balance(t, s, "alice"); got != 1000 {
		t.Errorf("alice USD = %d, want 1000", got)
	}
	if got := balance(t, s, "bob"); got != 0 {
		t.Errorf("bob USD = %d, want 0", got)
	}
}

func TestTransferRejectsBadAmount(t *testing.T) {
	s, _ := setup(t)
	deposit(t, s, "bob", 5000)

	for _, amount := range []money.Money{usd(t, 0), usd(t, -5000), {}} {
		if err := s.Transfer("alice", "bob", amount); !errors.Is(err, wallet.ErrInvalidAmount) {
			t.Errorf("Transfer(alice → bob, %v): err = %v, want ErrInvalidAmount", amount, err)
		}
	}
	// A negative transfer from alice to bob is a transfer from bob to alice
	// that bob never asked for: theft.
	if got := balance(t, s, "bob"); got != 5000 {
		t.Errorf("bob USD = %d, want 5000: a negative transfer took bob's money", got)
	}
	if got := balance(t, s, "alice"); got != 0 {
		t.Errorf("alice USD = %d, want 0", got)
	}
}

func TestTransferToSameAccount(t *testing.T) {
	s, _ := setup(t)
	deposit(t, s, "alice", 1000)

	if err := s.Transfer("alice", "alice", usd(t, 100)); !errors.Is(err, wallet.ErrSameAccount) {
		t.Errorf("Transfer(alice → alice): err = %v, want ErrSameAccount", err)
	}
}

func TestTransferUnknownAccount(t *testing.T) {
	s, _ := setup(t)
	deposit(t, s, "alice", 1000)

	for _, tc := range []struct{ from, to ledger.AccountID }{
		{"carol", "bob"},
		{"alice", "carol"},
	} {
		if err := s.Transfer(tc.from, tc.to, usd(t, 100)); !errors.Is(err, ledger.ErrUnknownAccount) {
			t.Errorf("Transfer(%s → %s): err = %v, want ErrUnknownAccount", tc.from, tc.to, err)
		}
	}
	if got := balance(t, s, "alice"); got != 1000 {
		t.Errorf("alice USD = %d, want 1000", got)
	}
}

func TestBalanceUnknownAccount(t *testing.T) {
	s, _ := setup(t)
	if _, err := s.Balance("carol", money.USD); !errors.Is(err, ledger.ErrUnknownAccount) {
		t.Errorf("Balance(carol): err = %v, want ErrUnknownAccount", err)
	}
}

func TestSystemAccountIsHidden(t *testing.T) {
	s, l := setup(t)
	deposit(t, s, "alice", 1000)

	// The system account may go below zero. If a customer could use it,
	// they could take any amount from it: money out of thin air.
	if err := s.Transfer("external", "alice", usd(t, 5000)); !errors.Is(err, ledger.ErrUnknownAccount) {
		t.Errorf("Transfer(external → alice): err = %v, want ErrUnknownAccount", err)
	}
	if err := s.Transfer("alice", "external", usd(t, 500)); !errors.Is(err, ledger.ErrUnknownAccount) {
		t.Errorf("Transfer(alice → external): err = %v, want ErrUnknownAccount", err)
	}
	if err := s.Deposit("external", usd(t, 500)); !errors.Is(err, ledger.ErrUnknownAccount) {
		t.Errorf("Deposit(external): err = %v, want ErrUnknownAccount", err)
	}
	if _, err := s.Balance("external", money.USD); !errors.Is(err, ledger.ErrUnknownAccount) {
		t.Errorf("Balance(external): err = %v, want ErrUnknownAccount", err)
	}

	if got := balance(t, s, "alice"); got != 1000 {
		t.Errorf("alice USD = %d, want 1000", got)
	}
	if got := external(t, l); got != -1000 {
		t.Errorf("external USD = %d, want -1000", got)
	}
}
