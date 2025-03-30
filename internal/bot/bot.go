package bot

import (
	"log"
	"sync"

	"carwash-bot/config"
	"carwash-bot/internal/models"
	"carwash-bot/internal/services"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CarWashBot struct {
	botAPI        *tgbotapi.BotAPI
	schedule      *services.ScheduleService
	userStates    map[int64]models.UserState
	adminID       int64
	lastMessageID map[int64]int
	msgIDLock     sync.Mutex
}

func New(config *config.Config) (*CarWashBot, error) {
	botAPI, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		return nil, err
	}

	botAPI.Debug = true

	return &CarWashBot{
		botAPI:        botAPI,
		schedule:      services.NewScheduleService(config.StartTime, config.EndTime),
		userStates:    make(map[int64]models.UserState),
		adminID:       config.AdminID,
		lastMessageID: make(map[int64]int),
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

// ... остальные методы ...
