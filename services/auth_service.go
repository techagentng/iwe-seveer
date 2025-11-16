package services

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/techagentng/iweapp/config"
	"github.com/techagentng/iweapp/db"
	apiError "github.com/techagentng/iweapp/errors"
	"github.com/techagentng/iweapp/models"
	"github.com/techagentng/iweapp/services/jwt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthService defines the authentication service interface
type AuthService interface {
	SignupUser(request *models.User) (*models.User, error)
	LoginUser(loginRequest *models.LoginRequest) (*models.LoginResponse, *apiError.Error)
	GoogleLoginUser(loginRequest *models.GoogleLoginRequest) (*models.LoginResponse, *apiError.Error)
	GetRoleByName(name string) (*models.Role, error)
}

// authService implements AuthService
type authService struct {
	Config   *config.Config
	authRepo db.AuthRepository
}

// NewAuthService creates a new AuthService instance
func NewAuthService(authRepo db.AuthRepository, conf *config.Config) AuthService {
	return &authService{
		Config:   conf,
		authRepo: authRepo,
	}
}

// SignupUser creates a new user account
func (s *authService) SignupUser(user *models.User) (*models.User, error) {
	if user == nil {
		return nil, errors.New("user is nil")
	}

	if user.Email == "" {
		return nil, errors.New("email is required")
	}

	// Check if email already exists
	if err := s.authRepo.IsEmailExist(user.Email); err != nil {
		return nil, fmt.Errorf("email already exists")
	}

	// Check if phone number already exists (if provided)
	if user.Telephone != "" {
		if err := s.authRepo.IsPhoneExist(user.Telephone); err != nil {
			return nil, fmt.Errorf("phone number already exists")
		}
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("SignupUser: error hashing password: %v", err)
		return nil, fmt.Errorf("failed to process password")
	}
	user.HashedPassword = string(hashedPassword)
	user.Password = "" // Clear plain password for security

	// Create user in database
	createdUser, err := s.authRepo.CreateUser(user)
	if err != nil {
		log.Printf("SignupUser: error creating user: %v", err)
		return nil, fmt.Errorf("failed to create user")
	}

	return createdUser, nil
}

// LoginUser authenticates a user with email and password
func (a *authService) LoginUser(loginRequest *models.LoginRequest) (*models.LoginResponse, *apiError.Error) {
	// Find user by email
	foundUser, err := a.authRepo.FindUserByEmail(loginRequest.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apiError.New("invalid email or password", http.StatusUnauthorized)
		}
		log.Printf("LoginUser: error finding user: %v", err)
		return nil, apiError.New("unable to find user", http.StatusInternalServerError)
	}

	// Verify password
	if err := foundUser.VerifyPassword(loginRequest.Password); err != nil {
		return nil, apiError.New("invalid email or password", http.StatusUnauthorized)
	}

	// Ensure user has a role assigned
	if foundUser.RoleID == uuid.Nil {
		log.Printf("LoginUser: user %s has no role assigned", foundUser.Email)
		return nil, apiError.New("user role not assigned", http.StatusInternalServerError)
	}

	// Fetch user's role
	role, err := a.authRepo.FindRoleByID(foundUser.RoleID)
	if err != nil {
		log.Printf("LoginUser: error fetching role for user %s: %v", foundUser.Email, err)
		return nil, apiError.New("unable to fetch role", http.StatusInternalServerError)
	}

	// Generate JWT tokens
	accessToken, refreshToken, err := jwt.GenerateTokenPair(
		foundUser.Email,
		a.Config.JWTSecret,
		foundUser.AdminStatus,
		foundUser.ID,
		role.Name,
	)
	if err != nil {
		log.Printf("LoginUser: error generating tokens for user %s: %v", foundUser.Email, err)
		return nil, apiError.ErrInternalServerError
	}

	return &models.LoginResponse{
		UserResponse: models.UserResponse{
			ID:        foundUser.ID,
			Fullname:  foundUser.Fullname,
			Username:  foundUser.Username,
			Telephone: foundUser.Telephone,
			Email:     foundUser.Email,
			RoleName:  role.Name,
		},
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// GoogleLoginUser handles Google OAuth login/signup
func (a *authService) GoogleLoginUser(loginRequest *models.GoogleLoginRequest) (*models.LoginResponse, *apiError.Error) {
	// Check if user exists
	foundUser, err := a.authRepo.FindUserByEmail(loginRequest.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new user for Google OAuth
			return a.createGoogleUser(loginRequest)
		}
		log.Printf("GoogleLoginUser: error finding user: %v", err)
		return nil, apiError.New("unable to find user", http.StatusInternalServerError)
	}

	// Fetch user's role
	role, err := a.authRepo.FindRoleByID(foundUser.RoleID)
	if err != nil {
		log.Printf("GoogleLoginUser: error fetching role: %v", err)
		return nil, apiError.New("unable to fetch role", http.StatusInternalServerError)
	}

	// Generate JWT tokens
	accessToken, refreshToken, err := jwt.GenerateTokenPair(
		foundUser.Email,
		a.Config.JWTSecret,
		foundUser.AdminStatus,
		foundUser.ID,
		role.Name,
	)
	if err != nil {
		log.Printf("GoogleLoginUser: error generating tokens: %v", err)
		return nil, apiError.ErrInternalServerError
	}

	return &models.LoginResponse{
		UserResponse: models.UserResponse{
			ID:        foundUser.ID,
			Fullname:  foundUser.Fullname,
			Username:  foundUser.Username,
			Telephone: foundUser.Telephone,
			Email:     foundUser.Email,
			RoleName:  role.Name,
		},
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// createGoogleUser creates a new user from Google OAuth
func (a *authService) createGoogleUser(loginRequest *models.GoogleLoginRequest) (*models.LoginResponse, *apiError.Error) {
	// Generate username from email
	username := strings.Split(loginRequest.Email, "@")[0]
	if len(username) < 2 {
		username = username + "user"
	}

	// Ensure username is unique
	baseUsername := username
	for i := 1; ; i++ {
		existingUser, err := a.authRepo.FindUserByUsername(username)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("createGoogleUser: error checking username: %v", err)
			return nil, apiError.New("unable to verify username", http.StatusInternalServerError)
		}
		if existingUser == nil {
			break
		}
		username = fmt.Sprintf("%s%d", baseUsername, i)
	}

	// Get the "User" role
	role, err := a.authRepo.FindRoleByName(models.RoleUser)
	if err != nil {
		log.Printf("createGoogleUser: error fetching role: %v", err)
		return nil, apiError.New("unable to assign role", http.StatusInternalServerError)
	}

	// Create new user
	newUser := &models.User{
		Email:     loginRequest.Email,
		Fullname:  loginRequest.Fullname,
		Username:  username,
		Telephone: loginRequest.Telephone,
		IsSocial:  true,
		RoleID:    role.ID,
	}

	createdUser, err := a.authRepo.CreateUser(newUser)
	if err != nil {
		log.Printf("createGoogleUser: error creating user: %v", err)
		return nil, apiError.New("unable to create user", http.StatusInternalServerError)
	}

	// Generate JWT tokens
	accessToken, refreshToken, err := jwt.GenerateTokenPair(
		createdUser.Email,
		a.Config.JWTSecret,
		createdUser.AdminStatus,
		createdUser.ID,
		role.Name,
	)
	if err != nil {
		log.Printf("createGoogleUser: error generating tokens: %v", err)
		return nil, apiError.ErrInternalServerError
	}

	return &models.LoginResponse{
		UserResponse: models.UserResponse{
			ID:        createdUser.ID,
			Fullname:  createdUser.Fullname,
			Username:  createdUser.Username,
			Telephone: createdUser.Telephone,
			Email:     createdUser.Email,
			RoleName:  role.Name,
		},
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// GetRoleByName retrieves a role by its name
func (a *authService) GetRoleByName(name string) (*models.Role, error) {
	role, err := a.authRepo.FindRoleByName(name)
	if err != nil {
		return nil, err
	}
	return role, nil
}