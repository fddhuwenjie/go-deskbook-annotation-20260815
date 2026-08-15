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

func TestCreateCopiesInput(t *testing.T) {
	store := NewStore()
	value := reservationFixture("r1", "desk-1", "alice", 9)
	if err := store.Create(&value); err != nil {
		t.Fatal(err)
	}
	value.Owner = "mallory"
	stored, err := store.Get("r1")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Owner != "alice" {
		t.Fatalf("stored owner=%q, want alice", stored.Owner)
	}
}

func TestGetReturnsCopy(t *testing.T) {
	store := NewStore()
	value := reservationFixture("r1", "desk-1", "alice", 9)
	if err := store.Create(&value); err != nil {
		t.Fatal(err)
	}
	first, err := store.Get("r1")
	if err != nil {
		t.Fatal(err)
	}
	first.Owner = "mallory"
	second, err := store.Get("r1")
	if err != nil {
		t.Fatal(err)
	}
	if second.Owner != "alice" {
		t.Fatalf("stored owner=%q, want alice", second.Owner)
	}
}

func TestUpdateCopiesInput(t *testing.T) {
	store := NewStore()
	value := reservationFixture("r1", "desk-1", "alice", 9)
	if err := store.Create(&value); err != nil {
		t.Fatal(err)
	}
	value.Owner = "bob"
	if err := store.Update(&value); err != nil {
		t.Fatal(err)
	}
	value.Owner = "mallory"
	stored, err := store.Get("r1")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Owner != "bob" {
		t.Fatalf("stored owner=%q, want bob", stored.Owner)
	}
}

func TestListByDeskUsesIDTieBreaker(t *testing.T) {
	store := NewStore()
	for _, id := range []string{"r2", "r1"} {
		value := reservationFixture(id, "desk-1", id, 9)
		if err := store.Create(&value); err != nil {
			t.Fatal(err)
		}
	}
	list := store.ListByDesk("desk-1")
	if len(list) != 2 || list[0].ID != "r1" || list[1].ID != "r2" {
		t.Fatalf("ids=%v, want [r1 r2]", reservationIDs(list))
	}
}

func TestReserveRejectsZeroLengthInterval(t *testing.T) {
	service := NewService(NewStore())
	value := reservationFixture("r1", "desk-1", "alice", 9)
	value.End = value.Start
	if err := service.Reserve(context.Background(), value); !errors.Is(err, ErrInvalidInterval) {
		t.Fatalf("Reserve error=%v, want ErrInvalidInterval", err)
	}
}

func TestCancelledReservationDoesNotBlockTimeSlot(t *testing.T) {
	store := NewStore()
	service := NewService(store)
	first := reservationFixture("r1", "desk-1", "alice", 9)
	if err := service.Reserve(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if err := service.Cancel(context.Background(), "r1"); err != nil {
		t.Fatal(err)
	}
	second := reservationFixture("r2", "desk-1", "bob", 9)
	if err := service.Reserve(context.Background(), second); err != nil {
		t.Fatalf("replacement reservation rejected: %v", err)
	}
}

func TestConfirmAlreadyConfirmedIsIdempotent(t *testing.T) {
	store := NewStore()
	service := NewService(store)
	value := reservationFixture("r1", "desk-1", "alice", 9)
	if err := service.Reserve(context.Background(), value); err != nil {
		t.Fatal(err)
	}
	if err := service.Confirm("r1"); err != nil {
		t.Fatal(err)
	}
	if err := service.Confirm("r1"); err != nil {
		t.Fatalf("second Confirm error=%v, want nil", err)
	}
}

func TestTransferRejectsWhitespaceOwner(t *testing.T) {
	store := NewStore()
	service := NewService(store)
	value := reservationFixture("r1", "desk-1", "alice", 9)
	if err := store.Create(&value); err != nil {
		t.Fatal(err)
	}
	if err := service.Transfer(context.Background(), "r1", "   "); !errors.Is(err, ErrInvalidOwner) {
		t.Fatalf("Transfer error=%v, want ErrInvalidOwner", err)
	}
}

func TestTransferRespectsCanceledContext(t *testing.T) {
	store := NewStore()
	service := NewService(store)
	value := reservationFixture("r1", "desk-1", "alice", 9)
	if err := store.Create(&value); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := service.Transfer(ctx, "r1", "bob"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Transfer error=%v, want context.Canceled", err)
	}
	stored, err := store.Get("r1")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Owner != "alice" {
		t.Fatalf("stored owner=%q, want alice", stored.Owner)
	}
}

func TestRescheduleDoesNotConflictWithItself(t *testing.T) {
	store := NewStore()
	service := NewService(store)
	value := reservationFixture("r1", "desk-1", "alice", 9)
	if err := service.Reserve(context.Background(), value); err != nil {
		t.Fatal(err)
	}
	if err := service.Reschedule(context.Background(), "r1", value.Start.Add(15*time.Minute), value.End.Add(15*time.Minute)); err != nil {
		t.Fatalf("Reschedule error=%v, want nil", err)
	}
}

func TestRescheduleIgnoresCancelledReservations(t *testing.T) {
	store := NewStore()
	service := NewService(store)
	first := reservationFixture("r1", "desk-1", "alice", 9)
	second := reservationFixture("r2", "desk-1", "bob", 11)
	if err := service.Reserve(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if err := service.Reserve(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	if err := service.Cancel(context.Background(), "r1"); err != nil {
		t.Fatal(err)
	}
	if err := service.Reschedule(context.Background(), "r2", first.Start, first.End); err != nil {
		t.Fatalf("Reschedule into cancelled slot error=%v, want nil", err)
	}
}

func TestRescheduleRejectsZeroLengthInterval(t *testing.T) {
	store := NewStore()
	service := NewService(store)
	value := reservationFixture("r1", "desk-1", "alice", 9)
	if err := service.Reserve(context.Background(), value); err != nil {
		t.Fatal(err)
	}
	if err := service.Reschedule(context.Background(), "r1", value.Start, value.Start); !errors.Is(err, ErrInvalidInterval) {
		t.Fatalf("Reschedule error=%v, want ErrInvalidInterval", err)
	}
}

func TestListByOwnerFiltersOwner(t *testing.T) {
	store := NewStore()
	for _, value := range []Reservation{
		reservationFixture("r1", "desk-1", "alice", 9),
		reservationFixture("r2", "desk-2", "bob", 10),
	} {
		current := value
		if err := store.Create(&current); err != nil {
			t.Fatal(err)
		}
	}
	list := store.ListByOwner("alice")
	if len(list) != 1 || list[0].ID != "r1" {
		t.Fatalf("ids=%v, want [r1]", reservationIDs(list))
	}
}

func TestListByOwnerReturnsCopies(t *testing.T) {
	store := NewStore()
	value := reservationFixture("r1", "desk-1", "alice", 9)
	if err := store.Create(&value); err != nil {
		t.Fatal(err)
	}
	list := store.ListByOwner("alice")
	list[0].Owner = "mallory"
	stored, err := store.Get("r1")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Owner != "alice" {
		t.Fatalf("stored owner=%q, want alice", stored.Owner)
	}
}

func TestDeleteRespectsCanceledContext(t *testing.T) {
	store := NewStore()
	service := NewService(store)
	value := reservationFixture("r1", "desk-1", "alice", 9)
	if err := store.Create(&value); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := service.Delete(ctx, "r1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Delete error=%v, want context.Canceled", err)
	}
	if _, err := store.Get("r1"); err != nil {
		t.Fatalf("reservation was deleted: %v", err)
	}
}

func reservationFixture(id, deskID, owner string, hour int) Reservation {
	start := time.Date(2026, 8, 15, hour, 0, 0, 0, time.UTC)
	return Reservation{ID: id, DeskID: deskID, Owner: owner, Start: start, End: start.Add(time.Hour)}
}

func reservationIDs(values []*Reservation) []string {
	ids := make([]string, len(values))
	for i, value := range values {
		ids[i] = value.ID
	}
	return ids
}
