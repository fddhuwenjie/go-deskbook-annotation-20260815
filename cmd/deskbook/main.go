package main

import (
	"context"
	"fmt"
	"time"

	"example.com/deskbook/booking"
)

func main() {
	service := booking.NewService(booking.NewStore())
	start := time.Now().UTC().Truncate(time.Hour).Add(time.Hour)
	err := service.Reserve(context.Background(), booking.Reservation{
		ID: "demo", DeskID: "desk-1", Owner: "demo-user", Start: start, End: start.Add(time.Hour),
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("reservation created")
}
