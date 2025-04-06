package bot

import (
	"carwash-bot/internal/storage"
	"log"
	"sync"

	"carwash-bot/config"
	"carwash-bot/internal/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type MessageHandler func(b *CarWashBot, update tgbotapi.Update, cfg *config.Config)

type CarWashBot struct {
	botAPI        *tgbotapi.BotAPI
	storage       *storage.Storage // Заменяем schedule на storage
	userStates    map[int64]models.UserState
	adminID       int64
	lastMessageID map[int64]int
	msgIDLock     sync.Mutex
	cfg           *config.Config // Добавляем конфиг в структуру бота
	handlers      map[string]MessageHandler
	store         *storage.Storage
}

func New(config *config.Config, store *storage.Storage) (*CarWashBot, error) {
	botAPI, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		return nil, err
	}

	botAPI.Debug = true

	return &CarWashBot{
		botAPI:        botAPI,
		userStates:    make(map[int64]models.UserState),
		adminID:       config.AdminID,
		lastMessageID: make(map[int64]int),
		cfg:           config, // Сохраняем конфиг в структуре
		handlers:      make(map[string]MessageHandler),
		store:         store,
	}, nil

}

func (b *CarWashBot) Start() {
	log.Printf("Авторизован как %s", b.botAPI.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.botAPI.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			b.handleMessage(update.Message)
		} else if update.CallbackQuery != nil {
			b.handleCallbackQuery(update.CallbackQuery)
		}
	}
}
func (b *CarWashBot) clearChat(chatID int64) {
	// 1. Отправляем сообщение о начале очистки
	msg := tgbotapi.NewMessage(chatID, "🧹 Начинаю очистку чата...")
	sentMsg, _ := b.botAPI.Send(msg)

	// 2. Получаем историю сообщений
	updates := tgbotapi.NewUpdate(0)
	updates.Timeout = 60

	// 3. Удаляем все сообщения бота
	b.msgIDLock.Lock()
	for msgID := range b.lastMessageID {
		if msgID == chatID {
			deleteMsg := tgbotapi.NewDeleteMessage(chatID, b.lastMessageID[msgID])
			b.botAPI.Request(deleteMsg)
		}
	}
	b.lastMessageID = make(map[int64]int) // Очищаем хранилище
	b.msgIDLock.Unlock()

	// 4. Удаляем сообщение о начале очистки
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, sentMsg.MessageID)
	b.botAPI.Request(deleteMsg)

	// 5. Отправляем новое приветственное сообщение
	b.sendWelcomeMessage(chatID)
}
