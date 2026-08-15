package booking

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidInterval = errors.New("invalid reservation interval")
	ErrConflict        = errors.New("reservation conflicts with an existing booking")
	ErrInvalidState    = errors.New("reservation state does not allow this operation")
	ErrInvalidOwner    = errors.New("invalid reservation owner")
	ErrInvalidDesk     = errors.New("invalid reservation desk")
	ErrEmptyBatch      = errors.New("reservation batch is empty")
)

type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

func (s *Service) Reserve(ctx context.Context, value Reservation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if value.ID == "" || value.DeskID == "" || value.Owner == "" || !value.Start.Before(value.End) {
		return ErrInvalidInterval
	}
	for _, existing := range s.store.ListByDesk(value.DeskID) {
		if existing.Status != StatusCancelled && overlaps(existing.Start, existing.End, value.Start, value.End) {
			return ErrConflict
		}
	}
	value.Status = StatusPending
	return s.store.Create(&value)
}

func (s *Service) Cancel(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	value, err := s.store.Get(id)
	if err != nil {
		return err
	}
	if value.Status == StatusCancelled {
		return nil
	}
	value.Status = StatusCancelled
	return s.store.Update(value)
}

func (s *Service) Confirm(id string) error {
	value, err := s.store.Get(id)
	if err != nil {
		return err
	}
	if value.Status == StatusConfirmed {
		return nil
	}
	if value.Status != StatusPending {
		return ErrInvalidState
	}
	value.Status = StatusConfirmed
	return s.store.Update(value)
}

func (s *Service) Transfer(ctx context.Context, id, owner string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(owner) == "" {
		return ErrInvalidOwner
	}
	value, err := s.store.Get(id)
	if err != nil {
		return err
	}
	value.Owner = owner
	return s.store.Update(value)
}

func (s *Service) Reschedule(ctx context.Context, id string, start, end time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !start.Before(end) {
		return ErrInvalidInterval
	}
	value, err := s.store.Get(id)
	if err != nil {
		return err
	}
	for _, existing := range s.store.ListByDesk(value.DeskID) {
		if existing.ID != id && existing.Status != StatusCancelled && overlaps(existing.Start, existing.End, start, end) {
			return ErrConflict
		}
	}
	value.Start = start
	value.End = end
	return s.store.Update(value)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.store.Delete(id)
}

func (s *Service) Available(ctx context.Context, deskID string, start, end time.Time) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if strings.TrimSpace(deskID) == "" {
		return false, ErrInvalidDesk
	}
	if !start.Before(end) {
		return false, ErrInvalidInterval
	}
	for _, existing := range s.store.ListByDesk(deskID) {
		if existing.Status != StatusCancelled && overlaps(existing.Start, existing.End, start, end) {
			return false, nil
		}
	}
	return true, nil
}

func (s *Service) ReserveBatch(ctx context.Context, values []Reservation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(values) == 0 {
		return ErrEmptyBatch
	}

	prepared := make([]Reservation, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, input := range values {
		value := input
		value.ID = strings.TrimSpace(value.ID)
		value.DeskID = strings.TrimSpace(value.DeskID)
		value.Owner = strings.TrimSpace(value.Owner)
		if value.ID == "" || value.DeskID == "" || value.Owner == "" || !value.Start.Before(value.End) {
			return ErrInvalidInterval
		}
		if _, exists := seen[value.ID]; exists {
			return ErrAlreadyExists
		}
		seen[value.ID] = struct{}{}
		if _, err := s.store.Get(value.ID); err == nil {
			return ErrAlreadyExists
		} else if !errors.Is(err, ErrNotFound) {
			return err
		}
		value.Status = StatusPending
		value.RetryCount = 0
		for _, existing := range s.store.ListByDesk(value.DeskID) {
			if existing.Status != StatusCancelled && overlaps(existing.Start, existing.End, value.Start, value.End) {
				return ErrConflict
			}
		}
		for _, other := range prepared {
			if other.DeskID == value.DeskID && overlaps(other.Start, other.End, value.Start, value.End) {
				return ErrConflict
			}
		}
		prepared = append(prepared, value)
	}

	for index := range prepared {
		if err := s.store.Create(&prepared[index]); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) SwapOwners(ctx context.Context, leftID, rightID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	left, err := s.store.Get(leftID)
	if err != nil {
		return err
	}
	if leftID == rightID {
		return nil
	}
	right, err := s.store.Get(rightID)
	if err != nil {
		return err
	}
	if left.Status == StatusCancelled || right.Status == StatusCancelled {
		return ErrInvalidState
	}
	left.Owner, right.Owner = right.Owner, left.Owner
	if err := s.store.Update(left); err != nil {
		return err
	}
	return s.store.Update(right)
}

func (s *Service) Extend(ctx context.Context, id string, duration time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if duration < 0 {
		return ErrInvalidInterval
	}
	value, err := s.store.Get(id)
	if err != nil {
		return err
	}
	if value.Status == StatusCancelled {
		return ErrInvalidState
	}
	newEnd := value.End.Add(duration)
	for _, existing := range s.store.ListByDesk(value.DeskID) {
		if existing.ID != id && existing.Status != StatusCancelled && overlaps(value.Start, newEnd, existing.Start, existing.End) {
			return ErrConflict
		}
	}
	value.End = newEnd
	return s.store.Update(value)
}

func (s *Service) PurgeCancelled(ctx context.Context, cutoff time.Time) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return s.store.PurgeCancelledBefore(cutoff), nil
}

func overlaps(leftStart, leftEnd, rightStart, rightEnd time.Time) bool {
	return leftStart.Before(rightEnd) && rightStart.Before(leftEnd)
}
