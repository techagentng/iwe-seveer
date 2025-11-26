package db

import (
	"github.com/techagentng/iweapp/models"
	"gorm.io/gorm"
)

// ContactRepository handles database operations for contacts
type ContactRepository struct {
	db *gorm.DB
}

// NewContactRepository creates a new ContactRepository
func NewContactRepository(db *gorm.DB) *ContactRepository {
	return &ContactRepository{db: db}
}

// CreateContact creates a new contact in the database
func (r *ContactRepository) CreateContact(contact *models.Contact) error {
	return r.db.Create(contact).Error
}

// GetContactByEmail retrieves a contact by email
func (r *ContactRepository) GetContactByEmail(email string) (*models.Contact, error) {
	var contact models.Contact
	err := r.db.Where("email = ?", email).First(&contact).Error
	if err != nil {
		return nil, err
	}
	return &contact, nil
}
