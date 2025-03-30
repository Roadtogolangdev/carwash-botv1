package services

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"carwash-bot/internal/models"
)

type ScheduleService struct {
	Bookings     []models.Booking
	BookingsLock sync.Mutex
	startTime    int
	endTime      int
}

func NewScheduleService(start, end int) *ScheduleService {
	return &ScheduleService{
		startTime: start,
		endTime:   end,
	}
}

func (s *ScheduleService) GetAvailableSlots() []models.TimeSlot {
	s.BookingsLock.Lock()
	defer s.BookingsLock.Unlock()

	var slots []models.TimeSlot

	for hour := s.startTime; hour <= s.endTime; hour++ {
		timeStr := fmt.Sprintf("%02d:00", hour)
		booked := false
		var booking models.Booking

		for _, b := range s.Bookings {
			if b.Time == timeStr {
				booked = true
				booking = b
				break
			}
		}

		slots = append(slots, models.TimeSlot{
			Time:      timeStr,
			Available: !booked,
			CarModel:  booking.CarModel,  // Убедимся, что передаем
			CarNumber: booking.CarNumber, // оба поля
		})
	}

	return slots
}

func (s *ScheduleService) BookTime(timeStr, carModel, carNumber string, userID int64) bool {
	s.BookingsLock.Lock()
	defer s.BookingsLock.Unlock()

	// Проверяем, свободно ли время
	for _, booking := range s.Bookings {
		if booking.Time == timeStr {
			return false
		}
	}

	// Добавляем новую запись (убедимся, что поля не пустые)
	if carModel == "" {
		carModel = "Не указана"
	}
	if carNumber == "" {
		carNumber = "Нет номера"
	}

	s.Bookings = append(s.Bookings, models.Booking{
		Time:      timeStr,
		CarModel:  carModel,
		CarNumber: carNumber,
		UserID:    userID,
		Created:   time.Now(),
	})

	s.sortBookings()
	return true
}

func (s *ScheduleService) CancelBooking(timeStr string) bool {
	s.BookingsLock.Lock()
	defer s.BookingsLock.Unlock()

	for i, b := range s.Bookings {
		if b.Time == timeStr {
			s.Bookings = append(s.Bookings[:i], s.Bookings[i+1:]...)
			return true
		}
	}
	return false
}

func (s *ScheduleService) sortBookings() {
	sort.Slice(s.Bookings, func(i, j int) bool {
		t1, _ := time.Parse("15:04", s.Bookings[i].Time)
		t2, _ := time.Parse("15:04", s.Bookings[j].Time)
		return t1.Before(t2)
	})
}

func formatTime(hour int) string {
	return strings.Repeat("0", 2-len(strconv.Itoa(hour))) + strconv.Itoa(hour) + ":00"
}
func (s *ScheduleService) BookDateTime(date, timeStr, carModel, carNumber string, userID int64) bool {
	s.BookingsLock.Lock()
	defer s.BookingsLock.Unlock()

	// Проверяем, свободно ли время
	for _, booking := range s.Bookings {
		if booking.Date == date && booking.Time == timeStr {
			return false
		}
	}

	s.Bookings = append(s.Bookings, models.Booking{
		Date:      date,
		Time:      timeStr,
		CarModel:  carModel,
		CarNumber: carNumber,
		UserID:    userID,
		Created:   time.Now(),
	})

	return true
}

func (s *ScheduleService) IsTimeAvailable(date, timeStr string) bool {
	s.BookingsLock.Lock()
	defer s.BookingsLock.Unlock()

	for _, booking := range s.Bookings {
		if booking.Date == date && booking.Time == timeStr {
			return false
		}
	}
	return true
}
