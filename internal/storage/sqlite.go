package storage

import (
	"carwash-bot/internal/models"
	"context"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

func (s *SQLiteStorage) Init() error {
	_, err := s.db.Exec(`
    CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        telegram_id INTEGER UNIQUE NOT NULL,
        username TEXT,
        first_name TEXT,
        last_name TEXT,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
    
    CREATE TABLE IF NOT EXISTS bookings (
        id TEXT PRIMARY KEY,
        user_id INTEGER NOT NULL,
        date TEXT NOT NULL,
        time TEXT NOT NULL,
        car_model TEXT NOT NULL,
        car_number TEXT NOT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
    );
    
    CREATE INDEX IF NOT EXISTS idx_bookings_user ON bookings(user_id);
    CREATE INDEX IF NOT EXISTS idx_bookings_date ON bookings(date);
    `)
	return err
}

func (s *SQLiteStorage) CreateBooking(ctx context.Context, booking *models.Booking) error {
	booking.ID = uuid.New().String()
	booking.CreatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
	INSERT INTO bookings (id, user_id, date, time, car_model, car_number)
	VALUES (?, ?, ?, ?, ?, ?)`,
		booking.ID,
		booking.UserID,
		booking.Date,
		booking.Time,
		booking.CarModel,
		booking.CarNumber)

	return err
}

func (s *SQLiteStorage) GetUserBookings(ctx context.Context, userID int64) ([]*models.Booking, error) {
	rows, err := s.db.QueryContext(ctx, `
	SELECT id, date, time, car_model, car_number, created_at
	FROM bookings WHERE user_id = ? ORDER BY date, time`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []*models.Booking
	for rows.Next() {
		var b models.Booking
		if err := rows.Scan(&b.ID, &b.Date, &b.Time, &b.CarModel, &b.CarNumber, &b.CreatedAt); err != nil {
			return nil, err
		}
		bookings = append(bookings, &b)
	}
	return bookings, nil
}
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
func (s *SQLiteStorage) IsTimeAvailable(ctx context.Context, date, time string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM bookings WHERE date = ? AND time = ?`,
		date, time).Scan(&count)
	return count == 0, err
}

func (s *SQLiteStorage) GetAllBookings(ctx context.Context) ([]*models.Booking, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, date, time, car_model, car_number 
		FROM bookings ORDER BY date, time`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []*models.Booking
	for rows.Next() {
		var b models.Booking
		if err := rows.Scan(&b.ID, &b.Date, &b.Time, &b.CarModel, &b.CarNumber); err != nil {
			return nil, err
		}
		bookings = append(bookings, &b)
	}
	return bookings, nil
}
func (s *SQLiteStorage) CancelBooking(ctx context.Context, bookingID string, userID int64) (*models.Booking, error) {
	// Начинаем транзакцию
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Получаем данные брони перед удалением
	var booking models.Booking
	err = tx.QueryRowContext(ctx, `
        SELECT id, date, time, car_model, car_number 
        FROM bookings 
        WHERE id = ? AND user_id = ?`,
		bookingID, userID).Scan(
		&booking.ID,
		&booking.Date,
		&booking.Time,
		&booking.CarModel,
		&booking.CarNumber)

	if err != nil {
		return nil, err
	}

	// Удаляем бронь
	_, err = tx.ExecContext(ctx, `
        DELETE FROM bookings 
        WHERE id = ? AND user_id = ?`,
		bookingID, userID)
	if err != nil {
		return nil, err
	}

	// Фиксируем транзакцию
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &booking, nil
}
func (s *SQLiteStorage) CreateOrUpdateUser(ctx context.Context, user *models.User) error {
	_, err := s.db.ExecContext(ctx, `
    INSERT INTO users (telegram_id, username, first_name, last_name)
    VALUES (?, ?, ?, ?)
    ON CONFLICT(telegram_id) DO UPDATE SET
        username = excluded.username,
        first_name = excluded.first_name,
        last_name = excluded.last_name`,
		user.TelegramID, user.Username, user.FirstName, user.LastName)
	return err
}
