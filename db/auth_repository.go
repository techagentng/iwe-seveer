package db

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/techagentng/iweapp/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AuthRepository defines the authentication repository interface
type AuthRepository interface {
	// User operations
	CreateUser(user *models.User) (*models.User, error)
	FindUserByEmail(email string) (*models.User, error)
	FindUserByUsername(username string) (*models.User, error)
	FindUserByID(id uint) (*models.User, error)
	IsEmailExist(email string) error
	IsPhoneExist(phone string) error

	// Role operations
	FindRoleByID(id uuid.UUID) (*models.Role, error)
	FindRoleByName(name string) (*models.Role, error)

	// Token blacklist operations
	AddToBlackList(blacklist *models.Blacklist) error
	IsTokenInBlacklist(token string) bool
}

type authRepo struct {
	DB *gorm.DB
}

// NewAuthRepo creates a new AuthRepository instance
func NewAuthRepo(db *GormDB) AuthRepository {
	return &authRepo{db.DB}
}

// CreateUser creates a new user in the database
func (a *authRepo) CreateUser(user *models.User) (*models.User, error) {
	if user == nil {
		return nil, fmt.Errorf("user is nil")
	}

	// Assign default "User" role if not set
	if user.RoleID == uuid.Nil {
		var defaultRole models.Role
		
		// Try to create the default role (ignore if exists)
		createDefault := models.Role{
			ID:   uuid.New(),
			Name: models.RoleUser,
		}
		_ = a.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&createDefault).Error
		
		// Fetch the default role
		if err := a.DB.Where("name = ?", models.RoleUser).First(&defaultRole).Error; err != nil {
			log.Printf("CreateUser: error fetching default role: %v", err)
			return nil, fmt.Errorf("failed to assign user role")
		}
		user.RoleID = defaultRole.ID
	}

	// Create the user
	if err := a.DB.Create(user).Error; err != nil {
		log.Printf("CreateUser: error: %v", err)
		return nil, err
	}

	return user, nil
}

// FindUserByEmail finds a user by email address
func (a *authRepo) FindUserByEmail(email string) (*models.User, error) {
	var user models.User
	if err := a.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindUserByUsername finds a user by username
func (a *authRepo) FindUserByUsername(username string) (*models.User, error) {
	var user models.User
	if err := a.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindUserByID finds a user by ID
func (a *authRepo) FindUserByID(id uint) (*models.User, error) {
	var user models.User
	if err := a.DB.Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// IsEmailExist checks if email already exists
func (a *authRepo) IsEmailExist(email string) error {
	var user models.User
	if err := a.DB.Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil // Email doesn't exist (good)
		}
		return err
	}
	return fmt.Errorf("email already exists")
}

// IsPhoneExist checks if phone number already exists
func (a *authRepo) IsPhoneExist(phone string) error {
	if phone == "" {
		return nil // Empty phone is allowed
	}
	
	var user models.User
	if err := a.DB.Where("telephone = ?", phone).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil // Phone doesn't exist (good)
		}
		return err
	}
	return fmt.Errorf("phone number already exists")
}

// FindRoleByID finds a role by UUID
func (a *authRepo) FindRoleByID(id uuid.UUID) (*models.Role, error) {
	var role models.Role
	if err := a.DB.Where("id = ?", id).First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// FindRoleByName finds a role by name
func (a *authRepo) FindRoleByName(name string) (*models.Role, error) {
	var role models.Role
	if err := a.DB.Where("name = ?", name).First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// AddToBlackList adds a token to the blacklist
func (a *authRepo) AddToBlackList(blacklist *models.Blacklist) error {
	return a.DB.Create(blacklist).Error
}

// IsTokenInBlacklist checks if a token is blacklisted
func (a *authRepo) IsTokenInBlacklist(token string) bool {
	var blacklist models.Blacklist
	err := a.DB.Where("token = ?", token).First(&blacklist).Error
	return err == nil // Returns true if token found in blacklist
}