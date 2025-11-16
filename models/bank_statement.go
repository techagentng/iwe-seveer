package models

import (
	"time"

	"github.com/google/uuid"
)

// BankStatement represents a parsed row from a bank statement CSV
type BankStatement struct {
	ID              uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	FileID          uuid.UUID  `gorm:"type:uuid;not null;index" json:"file_id"`
	TransactionDate time.Time  `gorm:"type:date;not null" json:"transaction_date"`
	Description     string     `gorm:"type:text" json:"description"`
	DebitAmount     *float64   `gorm:"type:numeric(14,2)" json:"debit_amount,omitempty"`
	CreditAmount    *float64   `gorm:"type:numeric(14,2)" json:"credit_amount,omitempty"`
	Balance         float64    `gorm:"type:numeric(14,2);not null" json:"balance"`
	Currency        string     `gorm:"type:varchar(10);default:'NGN'" json:"currency"`
	CreatedAt       time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	
	// Relationship
	UploadedFile UploadedFile `gorm:"foreignKey:FileID" json:"-"`
}

// TableName specifies the table name for BankStatement
func (BankStatement) TableName() string {
	return "bank_statements"
}

// BankStatementCSVRow represents a row from CSV before parsing
type BankStatementCSVRow struct {
	TransactionDate string
	Description     string
	DebitAmount     string
	CreditAmount    string
	Balance         string
	Currency        string
}
