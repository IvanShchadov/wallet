// Package wallet holds what a customer can do with money: open an account,
// deposit, transfer and check a balance.
package wallet

import (
	"errors"
	"fmt"

	"github.com/IvanShchadov/wallet/internal/ledger"
	"github.com/IvanShchadov/wallet/internal/money"
)

const externalAccount ledger.AccountID = "external" // системный счёт, откуда приходят пополнения

var (
	ErrInvalidAmount = errors.New("amount must be positive")
	ErrSameAccount   = errors.New("cannot transfer to the same account")
)

// Ledger is what the service needs from the book of transactions.
type Ledger interface {
	Open(id ledger.AccountID, kind ledger.AccountKind) error
	Post(tx ledger.Transaction) error
	Balance(id ledger.AccountID, currency money.Currency) (money.Money, error)
}

// Service runs the wallet's use cases on top of a ledger.
type Service struct {
	ledger Ledger
}

// NewService opens the system account and returns the service.
func NewService(l Ledger) (*Service, error) {
	err := l.Open(externalAccount, ledger.System)

	if err != nil && !errors.Is(err, ledger.ErrAccountExists) {
		return nil, err
	}

	return &Service{ledger: l}, nil
}

// OpenAccount opens a customer account.
func (s *Service) OpenAccount(id ledger.AccountID) error {
	return s.ledger.Open(id, ledger.User) // сервис сам ничего не хранит — просит книгу
}

// Balance returns a customer's balance in the currency.
func (s *Service) Balance(id ledger.AccountID, currency money.Currency) (money.Money, error) {
	if err := customer(id); err != nil {
		return money.Money{}, err
	}

	return s.ledger.Balance(id, currency)
}

// move posts one transaction: amount leaves from and comes to to.
func (s *Service) move(from, to ledger.AccountID, amount money.Money) error {
	out, err := money.New(-amount.Amount(), amount.Currency())

	if err != nil {
		return err
	}

	tx, err := ledger.NewTransaction(
		ledger.Entry{Account: from, Amount: out},
		ledger.Entry{Account: to, Amount: amount},
	)

	if err != nil {
		return err
	}

	return s.ledger.Post(tx)
}

// Deposit brings money from the outside world to a customer account.
func (s *Service) Deposit(id ledger.AccountID, amount money.Money) error {
	if amount.Amount() <= 0 {
		return ErrInvalidAmount
	}

	if err := customer(id); err != nil {
		return err
	}

	return s.move(externalAccount, id, amount)
}

// Transfer moves money from one customer account to another.
func (s *Service) Transfer(from, to ledger.AccountID, amount money.Money) error {
	if amount.Amount() <= 0 {
		return ErrInvalidAmount
	}

	if from == to {
		return ErrSameAccount
	}

	if err := customer(from); err != nil {
		return err
	}

	if err := customer(to); err != nil {
		return err
	}

	err := s.move(from, to, amount)

	return err
}

// customer hides the system account: for a customer it does not exist.
func customer(id ledger.AccountID) error {
	if id == externalAccount {
		return fmt.Errorf("%w: %s", ledger.ErrUnknownAccount, id)
	}

	return nil
}
