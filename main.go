package main

import (
	"github.com/Fox1N69/bot-task/bot"
	"github.com/Fox1N69/bot-task/infra"
	"github.com/Fox1N69/bot-task/utils/logger"
)

func main() {
	// Init config
	i := infra.New("config/config.json")
	// Get logger
	log := logger.GetLogger()
	//Init bot
	token := i.Config().GetString("bot_token")
	_, err := bot.New(token)
	if err != nil {
		log.Fatal("Failed to init bot: " + err.Error())
	}
}
