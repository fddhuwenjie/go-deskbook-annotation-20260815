package booking

import (
	"errors"
	"sort"
	"sync"
	"time"
)

var (
	ErrNotFound      = errors.New("reservation not found")
	ErrAlreadyExists = errors.New("reservation already exists")
)

type Store struct {
	mu    sync.RWMutex
	items map[string]*Reservation
}

func NewStore() *Store {
	return &Store{items: make(map[string]*Reservation)}
}

func (s *Store) Create(value *Reservation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.items[value.ID]; exists {
		return ErrAlreadyExists
	}
	s.items[value.ID] = cloneReservation(value)
	return nil
}

func (s *Store) Get(id string) (*Reservation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneReservation(value), nil
}

func (s *Store) Update(value *Reservation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[value.ID]; !ok {
		return ErrNotFound
	}
	s.items[value.ID] = cloneReservation(value)
	return nil
}

func (s *Store) ListByDesk(deskID string) []*Reservation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Reservation, 0)
	for _, value := range s.items {
		if value.DeskID == deskID {
			result = append(result, cloneReservation(value))
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Start.Equal(result[j].Start) {
			return result[i].ID < result[j].ID
		}
		return result[i].Start.Before(result[j].Start)
	})
	return result
}

func (s *Store) ListByOwner(owner string) []*Reservation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Reservation, 0)
	for _, value := range s.items {
		if value.Owner == owner {
			result = append(result, cloneReservation(value))
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Start.Equal(result[j].Start) {
			return result[i].ID < result[j].ID
		}
		return result[i].Start.Before(result[j].Start)
	})
	return result
}

func (s *Store) ListByStatus(status Status) []*Reservation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Reservation, 0)
	for _, value := range s.items {
		if value.Status == status {
			result = append(result, cloneReservation(value))
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Start.Equal(result[j].Start) {
			return result[i].ID < result[j].ID
		}
		return result[i].Start.Before(result[j].Start)
	})
	return result
}

func (s *Store) PurgeCancelledBefore(cutoff time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	deleted := 0
	for id, value := range s.items {
		if value.Status == StatusCancelled && !value.End.After(cutoff) {
			delete(s.items, id)
			deleted++
		}
	}
	return deleted
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return ErrNotFound
	}
	delete(s.items, id)
	return nil
}

func (s *Store) Status(id string) (Status, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.items[id]
	if !ok {
		return "", ErrNotFound
	}
	return value.Status, nil
}
