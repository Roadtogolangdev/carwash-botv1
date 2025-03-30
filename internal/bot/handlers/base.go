package handlers

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *CarWashBot) handleStart(msg *tgbotapi.Message) {
	msgText := `🚗 *Добро пожаловать в бота автомойки!* 🧼
    
Выберите действие:`

	replyMarkup := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📝 Записаться"),
			tgbotapi.NewKeyboardButton("🕒 Расписание"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("ℹ️ Помощь"),
		),
	)

	b.sendWelcomeMessage(msg.Chat.ID, msgText, replyMarkup)
}

func (b *CarWashBot) handleUnknownCommand(chatID int64) {
	b.sendMessage(chatID, "Я не понимаю эту команду. Используйте кнопки меню.")
}
