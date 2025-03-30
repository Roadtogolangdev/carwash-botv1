package services

import (
	"carwash-bot/internal/models"
	"fmt"
	"sort"
	"sync"
	"time"
)

type ScheduleService struct {
	bookings     []models.Booking
	bookingsLock sync.Mutex
	startTime    int // Начальное время (часы)
	endTime      int // Конечное время (часы)
}

func NewScheduleService(start, end int) *ScheduleService {
	return &ScheduleService{
		startTime: start,
		endTime:   end,
	}
}

// BookDateTime - добавляет новую запись с проверкой доступности
func (s *ScheduleService) BookDateTime(date, timeStr, carModel, carNumber string, userID int64) bool {
	s.bookingsLock.Lock()
	defer s.bookingsLock.Unlock()

	// Проверяем, свободно ли время
	for _, booking := range s.bookings {
		if booking.Date == date && booking.Time == timeStr {
			return false
		}
	}

	// Добавляем новую запись
	s.bookings = append(s.bookings, models.Booking{
		Date:      date,
		Time:      timeStr,
		CarModel:  carModel,
		CarNumber: carNumber,
		UserID:    userID,
		Created:   time.Now(),
	})

	return true
}

// IsTimeAvailable - проверяет доступность времени
func (s *ScheduleService) IsTimeAvailable(date, timeStr string) bool {
	s.bookingsLock.Lock()
	defer s.bookingsLock.Unlock()

	for _, booking := range s.bookings {
		if booking.Date == date && booking.Time == timeStr {
			return false
		}
	}
	return true
}

// CancelBooking - отменяет запись по времени

// GetBookingsGroupedByDate - возвращает записи сгруппированные по дате
func (s *ScheduleService) GetBookingsGroupedByDate() map[string][]models.Booking {
	s.bookingsLock.Lock()
	defer s.bookingsLock.Unlock()

	result := make(map[string][]models.Booking)
	for _, booking := range s.bookings {
		result[booking.Date] = append(result[booking.Date], booking)
	}
	return result
}

// GetAvailableTimeSlots - возвращает доступные временные слоты для даты
func (s *ScheduleService) GetAvailableTimeSlots(date string) []string {
	s.bookingsLock.Lock()
	defer s.bookingsLock.Unlock()

	var slots []string
	bookedTimes := make(map[string]bool)

	// Собираем занятые времена
	for _, booking := range s.bookings {
		if booking.Date == date {
			bookedTimes[booking.Time] = true
		}
	}

	// Генерируем все возможные слоты
	for hour := s.startTime; hour <= s.endTime; hour++ {
		timeStr := fmt.Sprintf("%02d:00", hour)
		if !bookedTimes[timeStr] {
			slots = append(slots, timeStr)
		}
	}

	return slots
}

// CancelBooking - отменяет запись по ID пользователя и времени
func (s *ScheduleService) CancelBooking(userID int64, date, timeStr string) bool {
	s.bookingsLock.Lock()
	defer s.bookingsLock.Unlock()

	for i, booking := range s.bookings {
		if booking.UserID == userID && booking.Date == date && booking.Time == timeStr {
			// Удаляем запись из slice
			s.bookings = append(s.bookings[:i], s.bookings[i+1:]...)
			return true
		}
	}
	return false
}

// GetUserBookings - возвращает записи пользователя
func (s *ScheduleService) GetUserBookings(userID int64) []models.Booking {
	s.bookingsLock.Lock()
	defer s.bookingsLock.Unlock()

	var result []models.Booking
	for _, booking := range s.bookings {
		if booking.UserID == userID {
			result = append(result, booking)
		}
	}

	// Сортируем по дате и времени
	sort.Slice(result, func(i, j int) bool {
		dateI, _ := time.Parse("02.01.2006", result[i].Date)
		dateJ, _ := time.Parse("02.01.2006", result[j].Date)
		if dateI.Equal(dateJ) {
			return result[i].Time < result[j].Time
		}
		return dateI.Before(dateJ)
	})

	return result
}
func (s *ScheduleService) GetBooking(userID int64, date, time string) *models.Booking {
	s.bookingsLock.Lock()
	defer s.bookingsLock.Unlock()

	for _, booking := range s.bookings {
		if booking.UserID == userID && booking.Date == date && booking.Time == time {
			return &booking
		}
	}
	return nil
}
