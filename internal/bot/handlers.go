package bot

import (
	"carwash-bot/internal/models"
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

	// Проверяем состояние пользователя (ожидание данных авто)
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
		b.showUserBookings(msg.Chat.ID, msg.From.ID)

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

	// Отвечаем на callback (убираем "часы ожидания")
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
	msgText := `🚗 *Добро пожаловать в бота автомойки!* 🧼
    
Выберите действие:`

	msg := tgbotapi.NewMessage(chatID, msgText)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📝 Записаться"),
			tgbotapi.NewKeyboardButton("🕒 Расписание"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("❌ Отменить запись"),
			tgbotapi.NewKeyboardButton("ℹ️ Помощь"),
		),
	)
	b.sendMessageWithSave(chatID, msg)
}

func (b *CarWashBot) handleTimeSelection(chatID, userID int64, timeStr string) {
	state := b.userStates[userID]

	// Проверяем доступность времени
	if !b.schedule.IsTimeAvailable(state.SelectedDate, timeStr) {
		b.sendMessage(chatID, "❌ Это время уже занято! Выберите другое время.")
		b.showTimeSlots(chatID, state.SelectedDate)
		return
	}

	// Сохраняем время и запрашиваем данные авто
	b.userStates[userID] = models.UserState{
		AwaitingCarInfo: true,
		SelectedDate:    state.SelectedDate,
		SelectedTime:    timeStr,
	}

	msg := tgbotapi.NewMessage(chatID, "Введите марку и номер машины через пробел\nПример: Лада 123")
	b.sendMessageWithSave(chatID, msg)
}

func (b *CarWashBot) handleCarInfoInput(chatID, userID int64, text string) {
	// Удаляем предыдущее сообщение
	b.deleteLastMessage(chatID)

	// Упрощенная проверка - просто разделяем по первому пробелу
	parts := strings.SplitN(text, " ", 2)
	if len(parts) < 2 {
		msg := tgbotapi.NewMessage(chatID, "Нужно ввести и марку, и номер!\nПример: Газель 123")
		b.sendMessageWithSave(chatID, msg)
		return
	}

	carModel := parts[0]
	carNumber := parts[1]
	state := b.userStates[userID]

	// Записываем в расписание
	if !b.schedule.BookDateTime(state.SelectedDate, state.SelectedTime, carModel, carNumber, userID) {
		msg := tgbotapi.NewMessage(chatID, "⚠️ Это время уже занято! Выберите другое время.")
		msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("📝 Записаться"),
				tgbotapi.NewKeyboardButton("🏠 Главное меню"),
			),
		)
		b.sendMessageWithSave(chatID, msg)
		return
	}

	// Удаляем состояние пользователя
	delete(b.userStates, userID)

	// Отправляем подтверждение
	confirmMsg := fmt.Sprintf(`✅ Вы записаны!
📅 %s в %s
🚗 %s %s`,
		state.SelectedDate, state.SelectedTime, carModel, carNumber)

	msg := tgbotapi.NewMessage(chatID, confirmMsg)
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🏠 Главное меню"),
		),
	)
	b.sendMessageWithSave(chatID, msg)

	// Уведомляем админа
	b.notifyAdminAboutNewBooking(state.SelectedTime, carModel, carNumber)
}

func (b *CarWashBot) showSchedule(chatID int64) {
	// Получаем записи через метод сервиса
	bookingsByDate := b.schedule.GetBookingsGroupedByDate()

	// Сортируем даты в хронологическом порядке
	var dates []time.Time
	for dateStr := range bookingsByDate {
		date, _ := time.Parse("02.01.2006", dateStr)
		dates = append(dates, date)
	}
	sort.Slice(dates, func(i, j int) bool {
		return dates[i].Before(dates[j])
	})

	var sb strings.Builder
	sb.WriteString("📅 Полное расписание:\n\n")

	now := time.Now()
	today := now.Format("02.01.2006")
	tomorrow := now.AddDate(0, 0, 1).Format("02.01.2006")

	for _, date := range dates {
		dateStr := date.Format("02.01.2006")

		// Форматируем заголовок
		switch dateStr {
		case today:
			sb.WriteString("=== Сегодня ===\n")
		case tomorrow:
			sb.WriteString("=== Завтра ===\n")
		default:
			sb.WriteString(fmt.Sprintf("=== %s ===\n", date.Format("Monday, 02.01")))
		}

		// Сортируем записи по времени
		bookings := bookingsByDate[dateStr]
		sort.Slice(bookings, func(i, j int) bool {
			return bookings[i].Time < bookings[j].Time
		})

		// Добавляем записи
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
	var rows [][]tgbotapi.InlineKeyboardButton
	for hour := 8; hour <= 20; hour++ {
		timeStr := fmt.Sprintf("%02d:00", hour)
		available := b.schedule.IsTimeAvailable(dateStr, timeStr)

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

	date, _ := time.Parse("02.01.2006", dateStr)
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Выберите время на %s:", date.Format("Monday, 02.01")))
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	b.sendMessageWithSave(chatID, msg)
}

func (b *CarWashBot) showDaySelection(chatID int64) {
	now := time.Now()
	var buttons [][]tgbotapi.InlineKeyboardButton

	for i := 0; i < 7; i++ {
		date := now.AddDate(0, 0, i)
		dateStr := date.Format("02.01.2006")

		btnText := fmt.Sprintf("📅 %s", date.Format("Mon 02.01"))
		if i == 0 {
			btnText = "🔵 Сегодня " + date.Format("02.01")
		} else if i == 1 {
			btnText = "🔵 Завтра " + date.Format("02.01")
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
	bookings := b.schedule.GetUserBookings(userID)

	if len(bookings) == 0 {
		b.sendMessage(chatID, "У вас нет активных записей.")
		return
	}

	var sb strings.Builder
	sb.WriteString("📋 *Ваши записи:*\n\n")

	var buttons [][]tgbotapi.InlineKeyboardButton

	for _, booking := range bookings {
		sb.WriteString(fmt.Sprintf(
			"📅 %s\n🕒 %s\n🚗 %s %s\n\n",
			booking.Date,
			booking.Time,
			booking.CarModel,
			booking.CarNumber,
		))

		// Добавляем кнопку отмены для каждой записи
		btnData := fmt.Sprintf("cancel_%s_%s", booking.Date, booking.Time)
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отменить эту запись", btnData),
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
	userBookings := b.schedule.GetUserBookings(userID)
	if len(userBookings) == 0 {
		b.sendMessage(chatID, "У вас нет активных записей.")
		return
	}

	var buttons [][]tgbotapi.InlineKeyboardButton
	for _, booking := range userBookings {
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
	success, booking := b.schedule.CancelBooking(bookingID, userID)
	if !success {
		b.sendMessage(chatID, "Не удалось отменить запись.")
		return
	}

	msg := fmt.Sprintf("✅ Запись отменена:\n%s %s - %s %s",
		booking.Date, booking.Time, booking.CarModel, booking.CarNumber)
	b.sendMessage(chatID, msg)
}
