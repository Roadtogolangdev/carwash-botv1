package storage

import (
	"carwash-bot/internal/models"
	"context"
	"database/sql"
	_ "log"
	_ "modernc.org/sqlite"
)

type SQLiteStorage struct {
	db *sql.DB
}

func NewSQLiteStorage(path string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return &SQLiteStorage{db: db}, nil
}

type BookingRepository interface {
	CreateBooking(ctx context.Context, booking *models.Booking) error
	GetUserBookings(ctx context.Context, userID int64) ([]models.Booking, error)
	CancelBooking(ctx context.Context, bookingID int64) error
	// Другие методы...
}

type UserRepository interface {
	CreateOrUpdateUser(ctx context.Context, user *models.User) error
	GetUserByID(ctx context.Context, userID int64) (*models.User, error)
}

func (s *SQLiteStorage) GetUserByTelegramID(ctx context.Context, telegramID int64) (*models.User, error) {
	user := &models.User{}
	err := s.db.QueryRowContext(ctx, `
	SELECT id, telegram_id, username, first_name, last_name, created_at
	FROM users WHERE telegram_id = ?`, telegramID).Scan(
		&user.ID,
		&user.TelegramID,
		&user.Username,
		&user.FirstName,
		&user.LastName,
		&user.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return user, err
}
