package hermes_test

import (
	"sync"
	"testing"

	"github.com/sbowman/hermes-pgx/v2"
)

func TestSessionLock(t *testing.T) {
	db, err := hermes.Connect("postgres://localhost/hermes_test?sslmode=disable")
	if err != nil {
		t.Fatalf("Unable to connect to database: %s", err)
	}

	const id uint64 = 12

	locks := 0

	lock, err := db.Lock(nil, id)
	if err != nil {
		t.Fatalf("Failed to acquire a lock: %s", err)
	}

	locks++

	// Hopefully these waits are correct to avoid deal with races...
	var wg1 sync.WaitGroup
	var wg2 sync.WaitGroup
	var wg3 sync.WaitGroup

	wg1.Add(1)
	wg2.Add(1)
	wg3.Add(1)

	go func() {
		_, err := db.TryLock(nil, id)
		if err != hermes.ErrLocked {
			t.Error("Expected the lock to be taken...")
		}
		wg1.Done()

		other, err := db.Lock(nil, id)
		if err != nil {
			t.Errorf("Failed to acquire second lock: %s", err)
			return
		}
		wg2.Done()

		locks++

		if err := other.Release(); err != nil {
			t.Errorf("Problem releasing second lock: %s", err)
		}
		wg3.Done()
	}()

	wg1.Wait()

	if err := lock.Release(); err != nil {
		t.Errorf("Failed to release first lock: %s", err)
	}
	wg2.Wait()

	if locks != 2 {
		t.Errorf("Failed to acquire and release both competing locks")
	}
	wg3.Wait()
}

func TestTransactionalLock(t *testing.T) {
	db, err := hermes.Connect("postgres://localhost/hermes_test?sslmode=disable")
	if err != nil {
		t.Fatalf("Unable to connect to database: %s", err)
	}

	tx, err := db.Begin(nil)
	if err != nil {
		t.Fatalf("Unable to connect to database: %s", err)
	}

	const id uint64 = 13

	locks := 0

	// The lock doesn't really matter in this case...
	_, err = tx.Lock(nil, id)
	if err != nil {
		t.Fatalf("Failed to acquire a lock: %s", err)
	}

	locks++

	// Hopefully these waits are correct to avoid deal with races...
	var wg1 sync.WaitGroup
	var wg2 sync.WaitGroup
	var wg3 sync.WaitGroup

	wg1.Add(1)
	wg2.Add(1)
	wg3.Add(1)

	go func() {
		tx, err := db.Begin(nil)
		if err != nil {
			t.Fatalf("Unable to connect to database: %s", err)
		}
		defer tx.Close(nil)

		_, err = tx.TryLock(nil, id)
		if err != hermes.ErrLocked {
			t.Error("Expected the lock to be taken...")
		}
		wg1.Done()

		other, err := tx.Lock(nil, id)
		if err != nil {
			t.Errorf("Failed to acquire second lock: %s", err)
			return
		}
		wg2.Done()

		locks++

		if err := other.Release(); err != nil {
			t.Errorf("Problem releasing second lock: %s", err)
		}
		wg3.Done()
	}()

	wg1.Wait()

	if err := tx.Close(nil); err != nil {
		t.Errorf("Failed to release first lock: %s", err)
	}
	wg2.Wait()

	if locks != 2 {
		t.Errorf("Failed to acquire and release both competing locks")
	}
	wg3.Wait()
}
