package bot

import (
	"github.com/Fox1N69/bot-task/utils/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type Bot interface {
}

type bot struct {
	log   logger.Logger
	token string
	api   *tgbotapi.BotAPI
	chats map[int64]bool
}

func New(token string) (Bot, error) {
	botAPI, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return &bot{
		log:   logger.GetLogger(),
		token: token,
		api:   botAPI,
		chats: make(map[int64]bool),
	}, nil
}
