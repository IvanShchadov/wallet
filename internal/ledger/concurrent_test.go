package ledger_test

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IvanShchadov/wallet/internal/ledger"
	"github.com/IvanShchadov/wallet/internal/money"
)

// Run these tests with the race detector:
//
//	go test -race -timeout 30s ./internal/ledger
//
// Without -race a data race often passes silently. With it the test fails
// with "WARNING: DATA RACE" and shows the two lines of code that collided.

// inParallel starts n goroutines at the same moment and waits for all of them.
// If they are not done in 5 seconds, the test fails: most likely a lock was
// taken and never released, or taken twice.
func inParallel(t *testing.T, n int, f func(i int)) {
	t.Helper()

	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			<-start // wait for the signal, so the goroutines really collide
			f(i)
		})
	}
	close(start)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("goroutines did not finish in 5s: looks like a deadlock (a lock taken and never released, or taken twice)")
	}
}

func TestParallelDeposits(t *testing.T) {
	l := newBook(t)
	const n = 100
	tx := mustTx(t, entry(t, "external", -100, money.USD), entry(t, "alice", 100, money.USD))

	inParallel(t, n, func(i int) {
		if err := l.Post(tx); err != nil {
			t.Errorf("deposit %d: unexpected error: %v", i, err)
		}
	})

	if got := mustBalance(t, l, "alice", money.USD); got != n*100 {
		t.Errorf("alice USD = %d, want %d: some deposits were lost", got, n*100)
	}
}

func TestBalanceWhilePosting(t *testing.T) {
	l := newBook(t)
	tx := mustTx(t, entry(t, "external", -100, money.USD), entry(t, "alice", 100, money.USD))

	// Even goroutines deposit 1 USD, odd ones read the balance at the same time.
	inParallel(t, 100, func(i int) {
		if i%2 == 0 {
			if err := l.Post(tx); err != nil {
				t.Errorf("deposit %d: unexpected error: %v", i, err)
			}
			return
		}
		b, err := l.Balance("alice", money.USD)
		if err != nil {
			t.Errorf("Balance: unexpected error: %v", err)
			return
		}
		if got := b.Amount(); got < 0 || got > 5000 || got%100 != 0 {
			t.Errorf("Balance during deposits = %d: the ledger can never be in this state", got)
		}
	})

	if got := mustBalance(t, l, "alice", money.USD); got != 5000 {
		t.Errorf("alice USD = %d, want 5000", got)
	}
}

func TestParallelTransfersNeverOverdraw(t *testing.T) {
	// A race condition does not show up on every run, so try several rounds.
	for round := range 50 {
		l := newBook(t)
		deposit(t, l, "alice", 10000, money.USD)
		tx := mustTx(t, entry(t, "alice", -3000, money.USD), entry(t, "bob", 3000, money.USD))

		var passed, rejected atomic.Int64
		inParallel(t, 50, func(i int) {
			err := l.Post(tx)
			switch {
			case err == nil:
				passed.Add(1)
			case errors.Is(err, ledger.ErrInsufficientFunds):
				rejected.Add(1)
			default:
				t.Errorf("transfer %d: unexpected error: %v", i, err)
			}
		})

		// 100 USD on the account and 50 attempts to send 30 USD: exactly 3 must pass.
		if passed.Load() != 3 || rejected.Load() != 47 {
			t.Fatalf("round %d: %d transfers passed and %d were rejected, want 3 and 47: "+
				"the balance check and the append must happen under one lock",
				round, passed.Load(), rejected.Load())
		}
		if got := mustBalance(t, l, "alice", money.USD); got != 1000 {
			t.Fatalf("round %d: alice USD = %d, want 1000", round, got)
		}
		if got := mustBalance(t, l, "bob", money.USD); got != 9000 {
			t.Fatalf("round %d: bob USD = %d, want 9000", round, got)
		}
	}
}

func TestParallelOpen(t *testing.T) {
	l := ledger.New()
	var opened, existed atomic.Int64

	// 100 goroutines and 10 ids: every id is opened by 10 goroutines at once.
	inParallel(t, 100, func(i int) {
		id := ledger.AccountID(fmt.Sprintf("user-%d", i%10))
		err := l.Open(id, ledger.User)
		switch {
		case err == nil:
			opened.Add(1)
		case errors.Is(err, ledger.ErrAccountExists):
			existed.Add(1)
		default:
			t.Errorf("Open(%s): unexpected error: %v", id, err)
		}
	})

	if opened.Load() != 10 || existed.Load() != 90 {
		t.Errorf("%d accounts opened and %d got ErrAccountExists, want 10 and 90: one id can be opened only once",
			opened.Load(), existed.Load())
	}
}
