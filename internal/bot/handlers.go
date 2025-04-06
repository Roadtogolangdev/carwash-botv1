package bot

import (
	"carwash-bot/internal/models"
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"sort"
	"strings"
	"time"
)

func (b *CarWashBot) handleMessage(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	userID := msg.From.ID
	text := msg.Text

	// Сначала сохраняем/обновляем пользователя
	user := &models.User{
		TelegramID: userID,
		Username:   msg.From.UserName,
		FirstName:  msg.From.FirstName,
		LastName:   msg.From.LastName,
	}
	if err := b.storage.CreateOrUpdateUser(context.Background(), user); err != nil {
		log.Printf("Ошибка сохранения пользователя: %v", err)
	}

	// Проверяем состояние пользователя
	if state, exists := b.userStates[userID]; exists {
		if state.AwaitingCarInfo {
			b.handleCarInfoInput(chatID, userID, text)
			return
		}
	}

	// Обрабатываем команды
	switch {
	case text == "/start" || text == "/menu" || text == "🏠 Главное меню":
		b.sendWelcomeMessage(chatID)

	case text == "📝 Записаться" || text == "/book":
		b.showDaySelection(chatID)

	case text == "🕒 Расписание" || text == "/schedule":
		b.showSchedule(chatID)

	case text == "❌ Мои записи" || text == "/mybookings":
		b.showUserBookings(chatID, userID)

	case text == "🧹 Очистить чат" || text == "/clear":
		b.clearChat(chatID)

	case text == "❌ Отменить запись" || text == "/cancel":
		b.handleCancelCommand(chatID, userID)

	default:
		b.sendMessage(chatID, "Я не понимаю эту команду. Используйте кнопки меню.")
	}
}

func (b *CarWashBot) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	chatID := query.Message.Chat.ID
	userID := query.From.ID
	data := query.Data

	callback := tgbotapi.NewCallback(query.ID, "")
	if _, err := b.botAPI.Request(callback); err != nil {
		log.Printf("Ошибка ответа на callback: %v", err)
	}

	switch {
	case strings.HasPrefix(data, "day_"):
		dateStr := strings.TrimPrefix(data, "day_")
		b.handleDaySelection(chatID, userID, dateStr)

	case strings.HasPrefix(data, "time_"):
		timeStr := strings.TrimPrefix(data, "time_")
		b.handleTimeSelection(chatID, userID, timeStr)

	case data == "main_menu":
		b.sendWelcomeMessage(chatID)

	case strings.HasPrefix(data, "cancel_"):
		bookingID := strings.TrimPrefix(data, "cancel_")
		b.handleBookingCancellation(chatID, userID, bookingID)

	default:
		log.Printf("Неизвестный callback: %s", data)
	}
}
func (b *CarWashBot) sendWelcomeMessage(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, `🚗 *Добро пожаловать в бота автомойки!* 🧼
    
Выберите действие:`)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📝 Записаться"),
			tgbotapi.NewKeyboardButton("🕒 Расписание"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("❌ Отменить запись"),
			tgbotapi.NewKeyboardButton("🧹 Очистить чат"),
		),
	)
	b.sendMessageWithSave(chatID, msg)
}

func (b *CarWashBot) handleTimeSelection(chatID, userID int64, timeStr string) {
	state := b.userStates[userID]

	// Проверяем доступность времени
	available, err := b.storage.IsTimeAvailable(context.Background(), state.SelectedDate, timeStr)
	if err != nil {
		log.Printf("Ошибка проверки времени: %v", err)
		b.sendMessage(chatID, "⚠️ Ошибка системы. Попробуйте позже.")
		return
	}

	if !available {
		b.sendMessage(chatID, "❌ Это время уже занято! Выберите другое время.")
		b.showTimeSlots(chatID, state.SelectedDate)
		return
	}

	b.userStates[userID] = models.UserState{
		AwaitingCarInfo: true,
		SelectedDate:    state.SelectedDate,
		SelectedTime:    timeStr,
	}

	msg := tgbotapi.NewMessage(chatID, "Введите марку и номер машины через пробел\nПример: Лада 123")
	b.sendMessageWithSave(chatID, msg)
}

func (b *CarWashBot) handleCarInfoInput(chatID, userID int64, text string) {
	b.deleteLastMessage(chatID)

	// Проверка на команды
	if b.isCommand(text) {
		msg := tgbotapi.NewMessage(chatID, "❌ Введите марку и номер машины, а не команду\nПример: Toyota CAM777")
		b.sendMessageWithSave(chatID, msg)
		return
	}

	parts := strings.SplitN(text, " ", 2)
	if len(parts) < 2 || len(parts[1]) < 3 {
		msg := tgbotapi.NewMessage(chatID, "❌ Неверный формат!\nВведите: Марка Номер\nПример: Kia ABC123")
		b.sendMessageWithSave(chatID, msg)
		return
	}

	state := b.userStates[userID]
	user, err := b.storage.GetUserByTelegramID(context.Background(), userID)
	if err != nil {
		log.Printf("Ошибка получения пользователя: %v", err)
		b.sendMessage(chatID, "⚠️ Ошибка системы. Попробуйте позже.")
		return
	}

	booking := &models.Booking{
		UserID:    user.ID,
		Date:      state.SelectedDate,
		Time:      state.SelectedTime,
		CarModel:  parts[0],
		CarNumber: parts[1],
	}

	if err := b.storage.CreateBooking(context.Background(), booking); err != nil {
		log.Printf("Ошибка создания записи: %v", err)
		b.sendMessage(chatID, "⚠️ Это время уже занято! Выберите другое.")
		b.showTimeSlots(chatID, state.SelectedDate)
		return
	}

	delete(b.userStates, userID)
	b.sendBookingConfirmation(chatID, booking)
	b.notifyAdmin(booking)
}

func (b *CarWashBot) showSchedule(chatID int64) {
	bookings, err := b.storage.GetAllBookings(context.Background())
	if err != nil {
		log.Printf("Ошибка получения расписания: %v", err)
		b.sendMessage(chatID, "⚠️ Ошибка загрузки расписания.")
		return
	}

	bookingsByDate := make(map[string][]*models.Booking)
	for _, booking := range bookings {
		bookingsByDate[booking.Date] = append(bookingsByDate[booking.Date], booking)
	}

	var sb strings.Builder
	sb.WriteString("📅 Полное расписание:\n\n")

	now := time.Now()
	today := now.Format("02.01.2006")
	tomorrow := now.AddDate(0, 0, 1).Format("02.01.2006")

	// Сортируем даты
	var dates []string
	for date := range bookingsByDate {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	for _, date := range dates {
		bookings := bookingsByDate[date]
		sort.Slice(bookings, func(i, j int) bool {
			return bookings[i].Time < bookings[j].Time
		})

		// Форматируем заголовок
		parsedDate, _ := time.Parse("02.01.2006", date)
		dayName := b.getDayName(parsedDate.Weekday())

		switch date {
		case today:
			sb.WriteString("=== Сегодня (" + dayName + ") ===\n")
		case tomorrow:
			sb.WriteString("=== Завтра (" + dayName + ") ===\n")
		default:
			sb.WriteString(fmt.Sprintf("=== %s, %s ===\n", dayName, date))
		}

		for _, booking := range bookings {
			sb.WriteString(fmt.Sprintf("🕒 %s - %s %s\n",
				booking.Time,
				booking.CarModel,
				booking.CarNumber))
		}
		sb.WriteString("\n")
	}

	if len(dates) == 0 {
		sb.WriteString("Нет записей\n")
	}

	msg := tgbotapi.NewMessage(chatID, sb.String())
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📝 Записаться"),
			tgbotapi.NewKeyboardButton("🏠 Главное меню"),
		),
	)
	b.sendMessageWithSave(chatID, msg)
}

func (b *CarWashBot) notifyAdminAboutNewBooking(timeStr, carModel, carNumber string) {
	msgText := fmt.Sprintf(`🆕 Новая запись:
Время: %s
Авто: %s %s`, timeStr, carModel, carNumber)

	msg := tgbotapi.NewMessage(b.adminID, msgText)
	b.botAPI.Send(msg)
}

func (b *CarWashBot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := b.botAPI.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}
func (b *CarWashBot) sendMessageWithSave(chatID int64, msg tgbotapi.MessageConfig) {
	sentMsg, err := b.botAPI.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
		return
	}

	b.msgIDLock.Lock()
	b.lastMessageID[chatID] = sentMsg.MessageID
	b.msgIDLock.Unlock()
}
func (b *CarWashBot) deleteLastMessage(chatID int64) {
	b.msgIDLock.Lock()
	msgID := b.lastMessageID[chatID]
	b.msgIDLock.Unlock()

	if msgID != 0 {
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, msgID)
		b.botAPI.Request(deleteMsg)
	}
}

func (b *CarWashBot) showTimeSlots(chatID int64, dateStr string) {
	date, _ := time.Parse("02.01.2006", dateStr)
	dayName := b.getDayName(date.Weekday())

	var rows [][]tgbotapi.InlineKeyboardButton
	for hour := 8; hour <= 20; hour++ {
		timeStr := fmt.Sprintf("%02d:00", hour)

		available, err := b.IsTimeAvailable(dateStr, timeStr)
		if err != nil {
			log.Printf("Ошибка проверки времени: %v", err)
			continue
		}

		btnText := fmt.Sprintf("🕒 %s", timeStr)
		if !available {
			btnText = "🔴 " + timeStr + " (Занято)"
		} else {
			btnText = "🔵 " + timeStr + " (Свободно)"
		}

		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(btnText, "time_"+timeStr),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 В главное меню", "main_menu"),
	))

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Выберите время на %s, %s:", dayName, dateStr))
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	b.sendMessageWithSave(chatID, msg)
}
func (b *CarWashBot) IsTimeAvailable(date, time string) (bool, error) {
	return b.storage.IsTimeAvailable(context.Background(), date, time)
}
func (b *CarWashBot) showDaySelection(chatID int64) {
	now := time.Now()
	var buttons [][]tgbotapi.InlineKeyboardButton

	for i := 0; i < 7; i++ {
		date := now.AddDate(0, 0, i)
		dateStr := date.Format("02.01.2006")
		dayName := b.getDayName(date.Weekday())

		btnText := fmt.Sprintf("📅 %s, %s", dayName, date.Format("02.01"))
		if i == 0 {
			btnText = "🔵 Сегодня, " + date.Format("02.01")
		} else if i == 1 {
			btnText = "🔵 Завтра, " + date.Format("02.01")
		}

		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(btnText, "day_"+dateStr),
		))
	}

	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 В главное меню", "main_menu"),
	))

	msg := tgbotapi.NewMessage(chatID, "Выберите день для записи:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	b.sendMessageWithSave(chatID, msg)
}
func (b *CarWashBot) handleDaySelection(chatID, userID int64, dateStr string) {
	selectedDate, err := time.Parse("02.01.2006", dateStr)
	if err != nil || selectedDate.Before(time.Now().Truncate(24*time.Hour)) {
		b.sendMessage(chatID, "❌ Нельзя записаться на прошедшую дату")
		b.showDaySelection(chatID)
		return
	}

	b.userStates[userID] = models.UserState{
		AwaitingTime: true,
		SelectedDate: dateStr,
	}

	b.showTimeSlots(chatID, dateStr)
}

func (b *CarWashBot) showUserBookings(chatID, userID int64) {
	user, err := b.storage.GetUserByTelegramID(context.Background(), userID)
	if err != nil || user == nil {
		log.Printf("Ошибка получения пользователя: %v", err)
		b.sendMessage(chatID, "⚠️ Ошибка загрузки записей.")
		return
	}

	bookings, err := b.storage.GetUserBookings(context.Background(), user.ID)
	if err != nil {
		log.Printf("Ошибка получения записей: %v", err)
		b.sendMessage(chatID, "⚠️ Ошибка загрузки записей.")
		return
	}

	if len(bookings) == 0 {
		b.sendMessage(chatID, "У вас нет активных записей.")
		return
	}

	var sb strings.Builder
	sb.WriteString("📋 Ваши записи:\n\n")

	var buttons [][]tgbotapi.InlineKeyboardButton
	for _, booking := range bookings {
		sb.WriteString(fmt.Sprintf(
			"📅 %s\n🕒 %s\n🚗 %s %s\n\n",
			booking.Date,
			booking.Time,
			booking.CarModel,
			booking.CarNumber))

		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отменить", "cancel_"+booking.ID),
		))
	}

	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🏠 В главное меню", "main_menu"),
	))

	msg := tgbotapi.NewMessage(chatID, sb.String())
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	b.sendMessageWithSave(chatID, msg)
}

func (b *CarWashBot) handleCancelCommand(chatID, userID int64) {
	user, err := b.storage.GetUserByTelegramID(context.Background(), userID)
	if err != nil || user == nil {
		b.sendMessage(chatID, "⚠️ Ошибка загрузки записей")
		return
	}

	bookings, err := b.storage.GetUserBookings(context.Background(), user.ID)
	if err != nil {
		b.sendMessage(chatID, "⚠️ Ошибка загрузки записей")
		return
	}

	if len(bookings) == 0 {
		b.sendMessage(chatID, "У вас нет активных записей.")
		return
	}

	var buttons [][]tgbotapi.InlineKeyboardButton
	for _, booking := range bookings {
		btnText := fmt.Sprintf("%s %s - %s %s",
			booking.Date, booking.Time, booking.CarModel, booking.CarNumber)
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(btnText, "cancel_"+booking.ID),
		))
	}

	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "main_menu"),
	))

	msg := tgbotapi.NewMessage(chatID, "Выберите запись для отмены:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	b.sendMessageWithSave(chatID, msg)
}
func (b *CarWashBot) handleBookingCancellation(chatID, userID int64, bookingID string) {
	user, err := b.storage.GetUserByTelegramID(context.Background(), userID)
	if err != nil || user == nil {
		log.Printf("Ошибка получения пользователя: %v", err)
		b.sendMessage(chatID, "⚠️ Ошибка системы.")
		return
	}

	booking, err := b.storage.CancelBooking(context.Background(), bookingID, user.ID)
	if err != nil {
		log.Printf("Ошибка отмены брони: %v", err)
		b.sendMessage(chatID, "❌ Не удалось отменить запись.")
		return
	}

	msg := fmt.Sprintf("✅ Запись отменена:\n%s %s - %s %s",
		booking.Date, booking.Time, booking.CarModel, booking.CarNumber)
	b.sendMessage(chatID, msg)
}
func (b *CarWashBot) createDayButtons() tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	days := b.cfg.Days // Используем дни из конфига

	// Создаем кнопки для каждого дня
	for engDay, ruDay := range days {
		btn := tgbotapi.NewInlineKeyboardButtonData(ruDay, "day_"+engDay)
		row := []tgbotapi.InlineKeyboardButton{btn}
		rows = append(rows, row)
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// Добавляем метод для получения русских названий дней
func (b *CarWashBot) getDayName(day time.Weekday) string {
	daysMap := map[time.Weekday]string{
		time.Monday:    "Понедельник",
		time.Tuesday:   "Вторник",
		time.Wednesday: "Среда",
		time.Thursday:  "Четверг",
		time.Friday:    "Пятница",
		time.Saturday:  "Суббота",
		time.Sunday:    "Воскресенье",
	}
	return daysMap[day]
}

func (b *CarWashBot) isCommand(text string) bool {
	commands := []string{
		"📝 Записаться", "🕒 Расписание", "❌ Отменить запись",
		"🏠 Главное меню", "/start", "/menu", "/book", "/schedule", "/cancel",
	}

	for _, cmd := range commands {
		if strings.EqualFold(text, cmd) {
			return true
		}
	}
	return false
}

func (b *CarWashBot) sendBookingConfirmation(chatID int64, booking *models.Booking) {
	msgText := fmt.Sprintf(`✅ Запись подтверждена!
📅 Дата: %s
🕒 Время: %s
🚗 Авто: %s %s`,
		booking.Date,
		booking.Time,
		booking.CarModel,
		booking.CarNumber)

	msg := tgbotapi.NewMessage(chatID, msgText)
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🏠 Главное меню"),
		),
	)
	b.sendMessageWithSave(chatID, msg)
}

func (b *CarWashBot) notifyAdmin(booking *models.Booking) {
	msgText := fmt.Sprintf(`🆕 Новая запись:
📅 %s в %s
🚗 %s %s`,
		booking.Date,
		booking.Time,
		booking.CarModel,
		booking.CarNumber)

	msg := tgbotapi.NewMessage(b.adminID, msgText)
	if _, err := b.botAPI.Send(msg); err != nil {
		log.Printf("Ошибка уведомления админа: %v", err)
	}
}
func (b *CarWashBot) GetAllBookings() ([]*models.Booking, error) {
	return b.storage.GetAllBookings(context.Background())
}
