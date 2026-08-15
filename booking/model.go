package booking

import "time"

type Status string

const (
	StatusPending   Status = "pending"
	StatusConfirmed Status = "confirmed"
	StatusCancelled Status = "cancelled"
)

type Reservation struct {
	ID         string
	DeskID     string
	Owner      string
	Start      time.Time
	End        time.Time
	Status     Status
	RetryCount int
}

func cloneReservation(value *Reservation) *Reservation {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
