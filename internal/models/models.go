package models

import "time"

type Booking struct {
	Date      string // Формат: "02.01.2006"
	Time      string
	CarModel  string
	CarNumber string
	UserID    int64
	Created   time.Time
}

type UserState struct {
	AwaitingDay     bool
	AwaitingTime    bool
	AwaitingCarInfo bool
	SelectedDate    string
	SelectedTime    string
}
type TimeSlot struct {
	Time      string
	Available bool
	BookedBy  string
	CarModel  any
	CarNumber any
}
