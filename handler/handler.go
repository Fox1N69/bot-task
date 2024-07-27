package handler

import (
	"time"

	"github.com/Fox1N69/bot-task/storage"
	"github.com/Fox1N69/bot-task/storage/models"
	"github.com/Fox1N69/bot-task/ton"
	"github.com/Fox1N69/bot-task/utils/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"gorm.io/gorm"
)

type Handler struct {
	bot           *tgbotapi.BotAPI
	storageClient storage.Storage
	log           logger.Logger
}

func NewHandler(bot *tgbotapi.BotAPI, storageClient storage.Storage) *Handler {
	return &Handler{
		bot:           bot,
		storageClient: storageClient,
		log:           logger.GetLogger(),
	}
}

func (h *Handler) HandleStart(msg *tgbotapi.Message) {
	telegramID := msg.From.ID
	user, err := h.storageClient.GetUser(telegramID)

	if err != nil && err == gorm.ErrRecordNotFound {
		// Создание нового пользователя
		now := time.Now().Format(time.RFC3339)
		user = &models.User{
			TelegramID:        telegramID,
			JoinDate:          now,
			TGSubscribed:      false,
			TwitterSubscribed: false,
		}
		if err := h.storageClient.CreateUser(user); err != nil {
			h.log.Printf("Failed to create user: %v", err)
			return
		}
		h.sendWelcomeMessage(msg)
	}

	if err != nil && err != gorm.ErrRecordNotFound {
		h.log.Printf("Failed to get user: %v", err)
		return
	}

	// Проверка подписок и отправка напоминаний
	h.checkSubscriptions(msg, user)

	var buttons [][]tgbotapi.InlineKeyboardButton

	if !user.TGSubscribed || !user.TwitterSubscribed {
		// Отправка кнопок для подписки на Telegram и Twitter
		buttons = h.getInlineButtonsForSubscriptions()
		msgConfig := tgbotapi.NewMessage(msg.Chat.ID, "Выберите действие:")
		msgConfig.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)

		if _, err := h.bot.Send(msgConfig); err != nil {
			h.log.Printf("Failed to send message: %v", err)
		}
	} else {
		// Отправка всех кнопок, кроме кнопок для подписки на Telegram и Twitter
		buttons = h.getInlineButtonsForConnectedUsers()
		msgConfig := tgbotapi.NewMessage(msg.Chat.ID, "Поздравляем! Вы подписаны на все каналы. Выберите действие:")
		msgConfig.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)

		if _, err := h.bot.Send(msgConfig); err != nil {
			h.log.Printf("Failed to send message: %v", err)
		}
	}
}

func (h *Handler) getInlineButtonsForSubscriptions() [][]tgbotapi.InlineKeyboardButton {
	var buttons []tgbotapi.InlineKeyboardButton

	// Добавление кнопок для подписки на Telegram и Twitter
	buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData("Подписка на Telegram", "Telegram"))
	buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData("Подписка на Twitter", "Twitter"))

	return [][]tgbotapi.InlineKeyboardButton{buttons}
}

func (h *Handler) getInlineButtonsForConnectedUsers() [][]tgbotapi.InlineKeyboardButton {
	var buttons []tgbotapi.InlineKeyboardButton

	buttonsFromDB, err := h.storageClient.GetAllButtons()
	if err != nil {
		h.log.Printf("Failed to get buttons from database: %v", err)
		return nil
	}

	for _, button := range buttonsFromDB {
		if button.Name != "Telegram" && button.Name != "Twitter" {
			buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(button.Name, button.Name))
		}
	}

	var inlineButtons [][]tgbotapi.InlineKeyboardButton
	if len(buttons) > 0 {
		inlineButtons = append(inlineButtons, buttons)
	}

	return inlineButtons
}

// HandleCallback
func (h *Handler) HandleCallback(callback *tgbotapi.CallbackQuery) {
	data := callback.Data

	button, err := h.storageClient.GetButton(data)
	if err != nil {
		h.log.Printf("Failed to get button: %v", err)
		return
	}

	if button.Flag {
		msgText := "Действие уже выполнено."
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, msgText)
		if _, err := h.bot.Send(msg); err != nil {
			h.log.Errorf("Failed to send message: %v", err)
		}
		return
	}

	// Записываем нажатие кнопки
	if err := h.storageClient.RecordButtonPress(callback.From.ID, data); err != nil {
		h.log.Errorf("Failed to record button press: %v", err)
	}

	switch data {
	case "Telegram":
		h.handleTelegramSubscription(callback)
	case "Twitter":
		h.handleTwitterSubscription(callback)
	case "Wallet":
		h.handleWalletConnection(callback)
	default:
		h.log.Printf("Unknown callback data: %s", data)
	}
}

func (h *Handler) handleTelegramSubscription(callback *tgbotapi.CallbackQuery) {
	userID := callback.From.ID

	user, err := h.storageClient.GetUser(userID)
	if err != nil {
		h.log.Errorf("Failed to get user: %v", err)
		return
	}

	if user.TGSubscribed {
		msgText := "Вы уже подписаны на канал!"
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, msgText)
		if _, err := h.bot.Send(msg); err != nil {
			h.log.Errorf("Failed to send message: %v", err)
		}
	} else {
		msgText := "Пожалуйста, подпишитесь на наш канал и нажмите кнопку снова."
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, msgText)
		if _, err := h.bot.Send(msg); err != nil {
			h.log.Errorf("Failed to send message: %v", err)
		}

		channelLink := "https://web.telegram.org/k/#-2214927764"
		msgText = "Перейдите по ссылке для подписки: " + channelLink
		msg = tgbotapi.NewMessage(callback.Message.Chat.ID, msgText)
		if _, err := h.bot.Send(msg); err != nil {
			h.log.Errorf("Failed to send message: %v", err)
		}
	}
}

func (h *Handler) handleTwitterSubscription(callback *tgbotapi.CallbackQuery) {
	userID := callback.From.ID

	user, err := h.storageClient.GetUser(userID)
	if err != nil {
		h.log.Errorf("Failed to get user: %v", err)
		return
	}

	if user.TwitterSubscribed {
		msgText := "Вы уже подписаны на Twitter!"
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, msgText)
		if _, err := h.bot.Send(msg); err != nil {
			h.log.Errorf("Failed to send message: %v", err)
		}
	} else {
		msgText := "Пожалуйста, подпишитесь на наш Twitter и нажмите кнопку снова."
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, msgText)
		if _, err := h.bot.Send(msg); err != nil {
			h.log.Errorf("Failed to send message: %v", err)
		}

		twitterLink := "https://twitter.com/yourTwitterHandle"
		msgText = "Перейдите по ссылке для подписки: " + twitterLink
		msg = tgbotapi.NewMessage(callback.Message.Chat.ID, msgText)
		if _, err := h.bot.Send(msg); err != nil {
			h.log.Errorf("Failed to send message: %v", err)
		}
	}
}

// handleWalletConnection
func (h *Handler) handleWalletConnection(callback *tgbotapi.CallbackQuery) {
	userID := callback.From.ID

	if err := h.processWalletConnection(userID); err != nil {
		h.log.Errorf("Failed to process wallet connection: %v", err)
		return
	}

	if err := h.storageClient.RecordWalletConnection(userID, "wallet_id_placeholder"); err != nil {
		h.log.Errorf("Failed to record wallet connection: %v", err)
	}

	button, err := h.storageClient.GetButton("Wallet")
	if err != nil {
		h.log.Printf("Failed to get button: %v", err)
		return
	}
	button.Flag = true
	if err := h.storageClient.UpdateButton(button); err != nil {
		h.log.Printf("Failed to update button: %v", err)
		return
	}

	user, err := h.storageClient.GetUser(userID)
	if err != nil {
		h.log.Printf("Failed to get user: %v", err)
		return
	}
	user.WalletConnected = true
	if err := h.storageClient.UpdateUser(user); err != nil {
		h.log.Printf("Failed to update user: %v", err)
		return
	}

	if user.TGSubscribed && user.TwitterSubscribed && user.WalletConnected {
		msgText := "Поздравляем! Вы подписаны на все каналы и подключили кошелек. Выберите другую кнопку."
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, msgText)

		buttons := h.getInlineButtonsForConnectedUsers()
		inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
		msg.ReplyMarkup = inlineKeyboard

		if _, err := h.bot.Send(msg); err != nil {
			h.log.Printf("Failed to send message: %v", err)
		}
	} else {
		msgText := "Подключение завершено! Проверьте подписки и подключение кошелька."
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, msgText)
		if _, err := h.bot.Send(msg); err != nil {
			h.log.Printf("Failed to send message: %v", err)
		}
	}
}

func (h *Handler) processWalletConnection(userID int) error {
	tonClient := ton.NewTonConnectClient("https://example.com/tonconnect-manifest.json", "")

	if err := tonClient.ConnectWallet(userID); err != nil {
		h.log.Errorf("Failed connect to the wallet: %v", err)
		return err
	}

	return nil
}

func (h *Handler) HandleUnknownCommand(msg *tgbotapi.Message) {
	msgText := "Команда не распознана. Пожалуйста, используйте команду /start."
	msgConfig := tgbotapi.NewMessage(msg.Chat.ID, msgText)
	h.bot.Send(msgConfig)
}

func (h *Handler) sendWelcomeMessage(msg *tgbotapi.Message) {
	welcomeText := "Добро пожаловать! Пожалуйста, проверьте подписки и подключите кошелек."
	msgConfig := tgbotapi.NewMessage(msg.Chat.ID, welcomeText)
	if _, err := h.bot.Send(msgConfig); err != nil {
		h.log.Printf("Failed to send welcome message: %v", err)
	}
}

func (h *Handler) checkSubscriptions(msg *tgbotapi.Message, user *models.User) {
	if !user.TGSubscribed {
		h.sendTelegramSubscriptionReminder(msg)
	}

	if !user.TwitterSubscribed {
		h.sendTwitterSubscriptionReminder(msg)
	}
}

func (h *Handler) sendTelegramSubscriptionReminder(msg *tgbotapi.Message) {
	msgText := "Пожалуйста, подпишитесь на наш Telegram канал."
	if _, err := h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, msgText)); err != nil {
		h.log.Errorf("Failed to send Telegram subscription reminder: %v", err)
	}
}

func (h *Handler) sendTwitterSubscriptionReminder(msg *tgbotapi.Message) {
	msgText := "Пожалуйста, подпишитесь на наш Twitter."
	if _, err := h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, msgText)); err != nil {
		h.log.Errorf("Failed to send Twitter subscription reminder: %v", err)
	}
}
