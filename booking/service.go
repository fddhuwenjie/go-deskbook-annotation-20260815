package booking

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidInterval = errors.New("invalid reservation interval")
	ErrConflict        = errors.New("reservation conflicts with an existing booking")
	ErrInvalidState    = errors.New("reservation state does not allow this operation")
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
	if value.Status != StatusPending {
		return ErrInvalidState
	}
	value.Status = StatusConfirmed
	return s.store.Update(value)
}

func overlaps(leftStart, leftEnd, rightStart, rightEnd time.Time) bool {
	return leftStart.Before(rightEnd) && rightStart.Before(leftEnd)
}
