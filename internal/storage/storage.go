package storage

import (
	"carwash-bot/internal/models"
	"context"
	"database/sql"
	"log"
	_ "log"
	_ "modernc.org/sqlite"
)

type Storage struct {
	DB *sql.DB
}

func New() *Storage {
	db, err := sql.Open("sqlite", "carwash.db")
	if err != nil {
		log.Fatal("Не удалось открыть базу данных:", err)
	}

	createUserTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		telegram_id INTEGER UNIQUE NOT NULL,
		username TEXT,
		first_name TEXT
	);`

	if _, err := db.Exec(createUserTable); err != nil {
		log.Fatal("Ошибка при создании таблицы users:", err)
	}

	return &Storage{DB: db}
}
func (s *Storage) SaveUser(telegramID int64, username, firstName string) error {
	_, err := s.DB.Exec(`
		INSERT OR IGNORE INTO users(telegram_id, username, first_name)
		VALUES (?, ?, ?)
	`, telegramID, username, firstName)

	return err
}
func NewSQLiteStorage(path string) (*Storage, error) {
	db, err := sql.Open("sqlite", "carwash.db")
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return &Storage{DB: db}, nil
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

func (s *Storage) GetUserByTelegramID(ctx context.Context, telegramID int64) (*models.User, error) {
	user := &models.User{}
	err := s.DB.QueryRowContext(ctx, `
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
