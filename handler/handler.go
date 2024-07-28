package handler

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Fox1N69/bot-task/storage"
	"github.com/Fox1N69/bot-task/storage/models"
	"github.com/Fox1N69/bot-task/utils/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/skip2/go-qrcode"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
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

	h.checkSubscriptions(msg, user)

	var buttons [][]tgbotapi.InlineKeyboardButton

	if !user.TGSubscribed || !user.TwitterSubscribed {
		buttons = h.getInlineButtonsForSubscriptions()
		msgConfig := tgbotapi.NewMessage(msg.Chat.ID, "Выберите действие:")
		msgConfig.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)

		if _, err := h.bot.Send(msgConfig); err != nil {
			h.log.Printf("Failed to send message: %v", err)
		}
	} else {
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
		if button.Name != "Telegram" && button.Name != "Twitter" && button.Name != "Wallet" {
			buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(button.Name, button.Name))
		}
	}

	var inlineButtons [][]tgbotapi.InlineKeyboardButton
	if len(buttons) > 0 {
		inlineButtons = append(inlineButtons, buttons)
	}

	return inlineButtons
}

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

func (h *Handler) handleWalletConnection(callback *tgbotapi.CallbackQuery) {
	userID := callback.From.ID

	if err := h.processWalletConnection(); err != nil {
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

func (h *Handler) processWalletConnection() error {
	client := liteclient.NewConnectionPool()

	// get config
	cfg, err := liteclient.GetConfigFromUrl(context.Background(), "https://ton.org/testnet-global.config.json")
	if err != nil {
		h.log.Fatalln("get config err: ", err.Error())
		return err
	}

	// connect to mainnet lite servers
	err = client.AddConnectionsFromConfig(context.Background(), cfg)
	if err != nil {
		h.log.Fatalln("connection err: ", err.Error())
		return err
	}

	api := ton.NewAPIClient(client, ton.ProofCheckPolicyFast).WithRetry()
	api.SetTrustedBlockFromConfig(cfg)

	ctx := client.StickyContext(context.Background())

	seed := wallet.NewSeed()

	w, err := wallet.FromSeed(api, seed, wallet.V4R2)
	if err != nil {
		return err
	}

	h.log.Println("wallet address:", w.WalletAddress())

	h.log.Println("fetching and checking proofs since config init block, it may take near a minute...")

	block, err := api.CurrentMasterchainInfo(context.Background())
	if err != nil {
		h.log.Fatalln("get masterchain info err: ", err.Error())
		return err
	}
	h.log.Println("master proof checks are completed successfully, now communication is 100% safe!")

	balance, err := w.GetBalance(ctx, block)
	if err != nil {
		h.log.Fatalln("GetBalance err:", err.Error())
		return err
	}

	if balance.Nano().Uint64() >= 3000000 {
		addr := address.MustParseAddr("EQCD39VS5jcptHL8vMjEXrzGaRcCVYto7HUn4bpAOg8xqB2N")

		h.log.Println("sending transaction and waiting for confirmation...")

		// if destination wallet is not initialized (or you don't care)
		// you should set bounce to false to not get money back.
		// If bounce is true, money will be returned in case of not initialized destination wallet or smart-contract error
		bounce := false

		transfer, err := w.BuildTransfer(addr, tlb.MustFromTON("0.003"), bounce, "Hello from tonutils-go!")
		if err != nil {
			h.log.Fatalln("Transfer err:", err.Error())
			return err
		}

		tx, block, err := w.SendWaitTransaction(ctx, transfer)
		if err != nil {
			h.log.Fatalln("SendWaitTransaction err:", err.Error())
			return err
		}

		balance, err = w.GetBalance(ctx, block)
		if err != nil {
			h.log.Fatalln("GetBalance err:", err.Error())
			return err
		}

		h.log.Printf("transaction confirmed at block %d, hash: %s balance left: %s", block.SeqNo,
			base64.StdEncoding.EncodeToString(tx.Hash), balance.String())

		return err
	}

	h.log.Println("not enough balance:", balance.String())

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

// Генерация и отправка QR-кода
func (h *Handler) GenerateAndSendQRCode(userID int) error {
	fileName := fmt.Sprintf("qrcodes/user_%d.png", userID)
	err := generateQRCode(fileName, "http://example.com/your-link")
	if err != nil {
		return fmt.Errorf("failed to generate QR code: %w", err)
	}

	msg := tgbotapi.NewPhotoUpload(int64(userID), fileName)
	if _, err := h.bot.Send(msg); err != nil {
		return fmt.Errorf("failed to send QR code: %w", err)
	}

	return nil
}

func generateQRCode(fileName, content string) error {
	// Создание директории, если не существует
	dir := filepath.Dir(fileName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Генерация QR-кода
	qr, err := qrcode.Encode(content, qrcode.Medium, 256)
	if err != nil {
		return err
	}

	// Сохранение QR-кода в файл
	return os.WriteFile(fileName, qr, 0644)
}
