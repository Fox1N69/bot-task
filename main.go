package main

import (
	"github.com/Fox1N69/bot-task/bot"
	"github.com/Fox1N69/bot-task/infra"
	"github.com/Fox1N69/bot-task/storage"
	"github.com/Fox1N69/bot-task/storage/models"
	"github.com/Fox1N69/bot-task/utils/logger"
)

func main() {
	// Init config
	i := infra.New("config/config.json")

	// Get logger
	log := logger.GetLogger()

	// Connect to database
	storage := storage.NewSotrage(i.GormDB())
	log.Info("Connect to database")

	// Migrate database
	i.Migrate(
		&models.User{},
		&models.Button{},
		&models.WalletConnection{},
		&models.ButtonPress{},
	)
	log.Info("Database migrate success")

	// Init bot
	token := i.Config().GetString("bot_token")
	bot, err := bot.New(token, storage)
	if err != nil {
		log.Fatal("Failed to init bot: " + err.Error())
	}

	// Start bot
	if err := bot.Start(); err != nil {
		log.Fatal("Failed to start bot: " + err.Error())
	}
}
