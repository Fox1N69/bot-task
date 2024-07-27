package storage

import (
	"github.com/Fox1N69/bot-task/storage/models"
	"gorm.io/gorm"
)

type Storage interface {
	GetUser(telegramID int) (*models.User, error)
	CreateUser(user *models.User) error
	UpdateUser(user *models.User) error
	GetButton(name string) (*models.Button, error)
	CreateButton(button *models.Button) error
	UpdateButton(button *models.Button) error
	GetAllButtons() ([]models.Button, error)
	RecordButtonPress(userID int, buttonName string) error
	RecordWalletConnection(userID int, wallet string) error
}
type storage struct {
	db *gorm.DB
}

func NewSotrage(db *gorm.DB) Storage {
	return &storage{
		db: db,
	}
}

// GetUser получает пользователя по ID
func (s *storage) GetUser(telegramID int) (*models.User, error) {
	var user models.User
	result := s.db.First(&user, "telegram_id = ?", telegramID)
	return &user, result.Error
}

// CreateUser создает нового пользователя
func (s *storage) CreateUser(user *models.User) error {
	return s.db.Create(user).Error
}

// UpdateUser обновляет данные пользователя
func (s *storage) UpdateUser(user *models.User) error {
	return s.db.Save(user).Error
}

// GetButton получает кнопку по имени
func (s *storage) GetButton(name string) (*models.Button, error) {
	var button models.Button
	result := s.db.First(&button, "name = ?", name)
	return &button, result.Error
}

// CreateButton создает новую кнопку
func (s *storage) CreateButton(button *models.Button) error {
	return s.db.Create(button).Error
}

// UpdateButton обновляет данные кнопки
func (s *storage) UpdateButton(button *models.Button) error {
	return s.db.Save(button).Error
}

// GetAllButtons получает все кнопки
func (s *storage) GetAllButtons() ([]models.Button, error) {
	var buttons []models.Button
	result := s.db.Find(&buttons)
	return buttons, result.Error
}

func (s *storage) RecordButtonPress(userID int, buttonName string) error {
	// Проверьте, что кнопка уже не была нажата
	var existingPress models.ButtonPress
	if err := s.db.Where("user_id = ? AND button = ?", userID, buttonName).First(&existingPress).Error; err == nil {
		return nil // кнопка уже была нажата
	}

	// Запись нового нажатия
	return s.db.Create(&models.ButtonPress{
		UserID:  userID,
		Button:  buttonName,
		Pressed: true,
	}).Error
}

func (s *storage) RecordWalletConnection(userID int, wallet string) error {
	return s.db.Create(&models.WalletConnection{
		UserID: userID,
		Wallet: wallet,
	}).Error
}
