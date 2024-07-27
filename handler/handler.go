package handler

import (
	"time"

	"github.com/Fox1N69/bot-task/storage"
	"github.com/Fox1N69/bot-task/storage/models"
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
		// Пользователь не найден, создаем нового пользователя
		user = &models.User{
			TelegramID: telegramID,
			JoinDate:   time.Now().Format(time.RFC3339),
		}
		if err := h.storageClient.CreateUser(user); err != nil {
			h.log.Printf("Failed to create user: %v", err)
			return
		}
	} else if err != nil {
		h.log.Printf("Failed to get user: %v", err)
		return
	} else {
		// Обновляем статус подписки пользователя
		user.TGSubscribed = true
		user.TwitterSubscribed = true
		if err := h.storageClient.UpdateUser(user); err != nil {
			h.log.Printf("Failed to update user: %v", err)
			return
		}
	}

	// Формируем и отправляем сообщение с кнопками
	msgText := "Добро пожаловать! Проверьте вашу подписку и выберите кнопку."
	msgConfig := tgbotapi.NewMessage(msg.Chat.ID, msgText)

	buttons := h.getInlineButtons()
	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msgConfig.ReplyMarkup = inlineKeyboard

	if _, err := h.bot.Send(msgConfig); err != nil {
		h.log.Printf("Failed to send message: %v", err)
	}
}

func (h *Handler) HandleCallback(callback *tgbotapi.CallbackQuery) {
	data := callback.Data

	// Проверка флага кнопки
	button, err := h.storageClient.GetButton(data)
	if err != nil {
		h.log.Printf("Failed to get button: %v", err)
		return
	}

	if button.Flag {
		msgText := "Задача уже выполнена!"
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, msgText)
		if _, err := h.bot.Send(msg); err != nil {
			h.log.Printf("Failed to send message: %v", err)
		}
		return
	}

	if data == "Telegramm" {
		channelID := int64(-2214927764)
		isSubscribed, err := h.isUserSubscribedToChannel(callback.From.ID, channelID)
		if err != nil {
			h.log.Errorf("Failed to check subscription: %v", err)
			h.log.Info(channelID, callback.From.ID)
			return
		}

		if isSubscribed {
			// Обновление статуса подписки в базе данных
			user, err := h.storageClient.GetUser(callback.From.ID)
			if err != nil {
				h.log.Errorf("Failed to get user: %v", err)
				return
			}
			user.TGSubscribed = true
			if err := h.storageClient.UpdateUser(user); err != nil {
				h.log.Errorf("Failed to update user: %v", err)
				return
			}

			msgText := "Вы уже подписаны на канал!"
			msg := tgbotapi.NewMessage(callback.Message.Chat.ID, msgText)
			if _, err := h.bot.Send(msg); err != nil {
				h.log.Errorf("Failed to send message: %v", err)
			}
		} else {
			// Отправляем сообщение с просьбой подписаться на канал
			msgText := "Пожалуйста, подпишитесь на наш канал и нажмите кнопку снова."
			msg := tgbotapi.NewMessage(callback.Message.Chat.ID, msgText)
			if _, err := h.bot.Send(msg); err != nil {
				h.log.Errorf("Failed to send message: %v", err)
			}

			// Отправляем ссылку на канал
			channelLink := "https://web.telegram.org/k/#-2214927764" // Замените на ссылку на ваш канал
			msgText = "Перейдите по ссылке для подписки: " + channelLink
			msg = tgbotapi.NewMessage(callback.Message.Chat.ID, msgText)
			if _, err := h.bot.Send(msg); err != nil {
				h.log.Errorf("Failed to send message: %v", err)
			}
		}
		return
	}

	// Обработка других кнопок
	h.processWalletConnection(callback.From.ID)

	// Обновление флага кнопки в базе данных
	button.Flag = true
	if err := h.storageClient.UpdateButton(button); err != nil {
		h.log.Printf("Failed to update button: %v", err)
		return
	}

	// Уведомление пользователя и возврат к списку кнопок
	msgText := "Подключение завершено! Выберите другую кнопку."
	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, msgText)

	buttons := h.getInlineButtons()
	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msg.ReplyMarkup = inlineKeyboard

	if _, err := h.bot.Send(msg); err != nil {
		h.log.Printf("Failed to send message: %v", err)
	}
}

// getInlineButtons
//
// returns inline buttons
func (h *Handler) getInlineButtons() [][]tgbotapi.InlineKeyboardButton {
	var buttons []tgbotapi.InlineKeyboardButton

	// Пример статичных кнопок
	// Замените на свой список кнопок, если он у вас другой
	staticButtons := []string{"Twitter", "Telegramm"}

	for _, name := range staticButtons {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(name, name))
	}

	var inlineButtons [][]tgbotapi.InlineKeyboardButton
	if len(buttons) > 0 {
		inlineButtons = append(inlineButtons, buttons)
	}

	return inlineButtons
}

func (h *Handler) isUserSubscribedToChannel(userID int, channelID int64) (bool, error) {
	chatMember, err := h.bot.GetChatMember(tgbotapi.ChatConfigWithUser{
		ChatID: channelID,
		UserID: userID,
	})
	if err != nil {
		return false, err
	}

	return chatMember.Status == "member" || chatMember.Status == "administrator" || chatMember.Status == "creator", nil
}

func (h *Handler) processWalletConnection(userID int) {
	// Implement your SDK connection logic here
}

func (h *Handler) HandleUnknownCommand(msg *tgbotapi.Message) {
	msgText := "Команда не распознана. Пожалуйста, используйте команду /start."
	msgConfig := tgbotapi.NewMessage(msg.Chat.ID, msgText)
	h.bot.Send(msgConfig)
}
