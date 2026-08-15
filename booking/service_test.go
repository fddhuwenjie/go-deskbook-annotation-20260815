package booking

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestReservationLifecycle(t *testing.T) {
	service := NewService(NewStore())
	start := time.Date(2026, 8, 15, 9, 0, 0, 0, time.UTC)
	if err := service.Reserve(context.Background(), Reservation{ID: "r1", DeskID: "desk-1", Owner: "alice", Start: start, End: start.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if err := service.Confirm("r1"); err != nil {
		t.Fatal(err)
	}
	if err := service.Cancel(context.Background(), "r1"); err != nil {
		t.Fatal(err)
	}
	status, err := service.store.Status("r1")
	if err != nil || status != StatusCancelled {
		t.Fatalf("status=%q err=%v", status, err)
	}
}

func TestAdjacentReservationsDoNotConflict(t *testing.T) {
	service := NewService(NewStore())
	start := time.Date(2026, 8, 15, 9, 0, 0, 0, time.UTC)
	first := Reservation{ID: "r1", DeskID: "desk-1", Owner: "alice", Start: start, End: start.Add(time.Hour)}
	second := Reservation{ID: "r2", DeskID: "desk-1", Owner: "bob", Start: first.End, End: first.End.Add(time.Hour)}
	if err := service.Reserve(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if err := service.Reserve(context.Background(), second); err != nil {
		t.Fatalf("adjacent reservation rejected: %v", err)
	}
}

func TestListByDeskReturnsCopies(t *testing.T) {
	store := NewStore()
	service := NewService(store)
	start := time.Date(2026, 8, 15, 9, 0, 0, 0, time.UTC)
	if err := service.Reserve(context.Background(), Reservation{ID: "r1", DeskID: "desk-1", Owner: "alice", Start: start, End: start.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	list := store.ListByDesk("desk-1")
	list[0].Owner = "mallory"
	stored, err := store.Get("r1")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Owner != "alice" {
		t.Fatalf("stored owner changed through list result: %q", stored.Owner)
	}
}

func TestCancelRespectsCanceledContext(t *testing.T) {
	store := NewStore()
	service := NewService(store)
	start := time.Date(2026, 8, 15, 9, 0, 0, 0, time.UTC)
	if err := service.Reserve(context.Background(), Reservation{ID: "r1", DeskID: "desk-1", Owner: "alice", Start: start, End: start.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := service.Cancel(ctx, "r1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Cancel error=%v, want context.Canceled", err)
	}
}

func TestConfirmMissingReservationReturnsError(t *testing.T) {
	service := NewService(NewStore())
	if err := service.Confirm("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Confirm error=%v, want ErrNotFound", err)
	}
}

func TestStatusConcurrentWithUpdates(t *testing.T) {
	store := NewStore()
	value := &Reservation{ID: "r1", DeskID: "desk-1", Owner: "alice", Start: time.Now(), End: time.Now().Add(time.Hour), Status: StatusPending}
	if err := store.Create(value); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 1000; i++ {
			current, err := store.Get("r1")
			if err != nil {
				return
			}
			current.Owner = "owner"
			_ = store.Update(current)
		}
	}()
	for i := 0; i < 1000; i++ {
		if _, err := store.Status("r1"); err != nil {
			t.Fatal(err)
		}
	}
	<-done
}
