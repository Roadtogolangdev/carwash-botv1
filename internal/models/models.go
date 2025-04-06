package models

import (
	"time"
)

type Booking struct {
	ID        string    `json:"id" db:"id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	Date      string    `json:"date" db:"date"` // Формат: "02.01.2006"
	Time      string    `json:"time" db:"time"` // Формат: "15:00"
	CarModel  string    `json:"car_model" db:"car_model"`
	CarNumber string    `json:"car_number" db:"car_number"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// Дополнительные поля для JOIN-запросов
	User *User `json:"user,omitempty" db:"-"`
}

type User struct {
	ID         int64     `json:"id" db:"id"`
	TelegramID int64     `json:"telegram_id" db:"telegram_id"`
	Username   string    `json:"username" db:"username"`
	FirstName  string    `json:"first_name" db:"first_name"`
	LastName   string    `json:"last_name" db:"last_name"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`

	// Дополнительные отношения
	Bookings []*Booking `json:"bookings,omitempty" db:"-"`
}

type UserState struct {
	AwaitingDay     bool   `json:"awaiting_day"`
	AwaitingTime    bool   `json:"awaiting_time"`
	AwaitingCarInfo bool   `json:"awaiting_car_info"`
	SelectedDate    string `json:"selected_date"`
	SelectedTime    string `json:"selected_time"`
}

type TimeSlot struct {
	Time      string `json:"time"`
	Available bool   `json:"available"`
	BookedBy  string `json:"booked_by,omitempty"`
	CarModel  string `json:"car_model,omitempty"`
	CarNumber string `json:"car_number,omitempty"`
}
