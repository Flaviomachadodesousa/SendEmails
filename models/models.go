package models

import (
	"gorm.io/gorm"
	"time"
)

// Usuario representa destinatários
type Usuario struct {
	ID        uint      `gorm:"primaryKey"`
	Nome      string    `gorm:"type:varchar(100);not null"`
	Email     string    `gorm:"type:varchar(255);unique;not null"`
	CreatedAt time.Time
}

// EmailStatus registra envio de e-mails
type EmailStatus struct {
	ID        uint      `gorm:"primaryKey"`
	Email     string    `gorm:"type:varchar(255);not null"`
	Status    string    `gorm:"type:varchar(50);not null"`
	CreatedAt time.Time
}

// AutoMigrate faz a migração automática
func AutoMigrate(db *gorm.DB) {
	db.AutoMigrate(&Usuario{}, &EmailStatus{})
}
