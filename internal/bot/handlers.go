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

	case strings.HasPrefix(text, "/cancel ") && userID == b.adminID:
		timeStr := strings.TrimPrefix(text, "/cancel ")
		b.handleCancelBooking(chatID, timeStr)

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
	case strings.HasPrefix(data, "book_"):
		timeStr := strings.TrimPrefix(data, "book_")
		b.handleTimeSelection(chatID, userID, timeStr)

	case data == "main_menu":
		b.sendWelcomeMessage(chatID)

	case strings.HasPrefix(data, "day_"):
		dateStr := strings.TrimPrefix(data, "day_")
		b.handleDaySelection(chatID, userID, dateStr)

	case strings.HasPrefix(data, "time_"):
		timeStr := strings.TrimPrefix(data, "time_")
		b.handleTimeSelection(chatID, userID, timeStr)
	case data == "day_selection":
		b.showDaySelection(chatID) // Просто вызываем метод заново

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
			tgbotapi.NewKeyboardButton("ℹ️ Помощь"),
		),
	)
	// Для Reply-кнопок можно задать цвет через параметры
	// Но в текущей версии API это делается через web_app
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
	if !b.schedule.BookTime(state.SelectedTime, carModel, carNumber, userID) {
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
	if !b.schedule.BookDateTime(state.SelectedDate, state.SelectedTime, carModel, carNumber, userID) {
		b.sendMessage(chatID, "❌ Время стало занято! Начните запись заново.")
		b.showDaySelection(chatID)
		return
	}

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
	// Получаем все записи, сгруппированные по датам
	bookingsByDate := b.groupBookingsByDate()

	// Сортируем даты в хронологическом порядке
	var dates []string
	for dateStr := range bookingsByDate {
		dates = append(dates, dateStr)
	}
	sort.Strings(dates)

	// Формируем сообщение с расписанием
	var sb strings.Builder
	sb.WriteString("📅 Полное расписание:\n\n")

	now := time.Now()
	todayStr := now.Format("02.01.2006")
	tomorrowStr := now.AddDate(0, 0, 1).Format("02.01.2006")

	for _, dateStr := range dates {
		date, _ := time.Parse("02.01.2006", dateStr)

		// Форматируем заголовок даты
		switch dateStr {
		case todayStr:
			sb.WriteString("=== Сегодня ===\n")
		case tomorrowStr:
			sb.WriteString("=== Завтра ===\n")
		default:
			sb.WriteString(fmt.Sprintf("=== %s ===\n", date.Format("Monday, 02.01")))
		}

		// Добавляем записи для этой даты
		for _, booking := range bookingsByDate[dateStr] {
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

	// Создаем сообщение с кнопками
	msg := tgbotapi.NewMessage(chatID, sb.String())
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📝 Записаться"),
			tgbotapi.NewKeyboardButton("🏠 Главное меню"),
		),
	)
	b.sendMessageWithSave(chatID, msg)
}

func (b *CarWashBot) handleCancelBooking(chatID int64, timeStr string) {
	if b.schedule.CancelBooking(timeStr) {
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Запись на %s отменена.", timeStr))
		b.sendMessageWithSave(chatID, msg)
		b.showSchedule(chatID)
	} else {
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Запись на %s не найдена.", timeStr))
		b.sendMessageWithSave(chatID, msg)
	}
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
	// ... проверка даты ...

	var rows [][]tgbotapi.InlineKeyboardButton
	for hour := 8; hour <= 20; hour++ {
		timeStr := fmt.Sprintf("%02d:00", hour)
		available := b.schedule.IsTimeAvailable(dateStr, timeStr)

		// Синие кнопки с разными emoji для статуса
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

	// Синяя кнопка "Назад"
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Выбрать другой день", "day_selection"),
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

		// Синие кнопки с emoji
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

	// Синяя кнопка "Назад"
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "main_menu"),
	))

	msg := tgbotapi.NewMessage(chatID, "Выберите день для записи:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	b.sendMessageWithSave(chatID, msg)
}
func (b *CarWashBot) handleDaySelection(chatID, userID int64, dateStr string) {
	// Проверяем, не прошла ли дата
	selectedDate, err := time.Parse("02.01.2006", dateStr)
	if err != nil {
		b.sendMessage(chatID, "Ошибка формата даты")
		return
	}

	today := time.Now().Truncate(24 * time.Hour)
	if selectedDate.Before(today) {
		b.sendMessage(chatID, "❌ Нельзя записаться на прошедшую дату")
		b.showDaySelection(chatID)
		return
	}

	// Сохраняем выбранную дату
	b.userStates[userID] = models.UserState{
		AwaitingTime: true,
		SelectedDate: dateStr,
	}

	// Показываем выбор времени для этой даты
	b.showTimeSlots(chatID, dateStr)
}
func (b *CarWashBot) groupBookingsByDate() map[string][]models.Booking {
	b.schedule.BookingsLock.Lock()
	defer b.schedule.BookingsLock.Unlock()

	result := make(map[string][]models.Booking)
	for _, booking := range b.schedule.Bookings {
		result[booking.Date] = append(result[booking.Date], booking)
	}
	return result
}
