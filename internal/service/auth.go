package service

import (
	models "ax-management/internal/model"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	db        *gorm.DB
	jwtSecret []byte
}

// Pass the secret in via constructor to avoid global variables
func NewAuthService(db *gorm.DB, secret string) *AuthService {
	return &AuthService{
		db:        db,
		jwtSecret: []byte(secret),
	}
}

func (s *AuthService) RegisterUser(req models.RegisterRequest) (*models.User, error) {
	// 1. Hash the password securely
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to process password")
	}

	// 2. Create the User record
	user := &models.User{
		FullName:     req.FullName,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
	}

	// 3. Save to database (will fail if email is not unique)
	if err := s.db.Create(user).Error; err != nil {
		return nil, errors.New("user with this email already exists")
	}

	return user, nil
}

func (s *AuthService) GenerateJWT(userID uuid.UUID, email string) (string, error) {
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"exp":   time.Now().Add(time.Hour * 24).Unix(), // 24-hour expiration
		"iat":   time.Now().Unix(),                     // Issued at
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// Add this below your existing RegisterUser function

func (s *AuthService) AuthenticateUser(email, password string) (*models.User, error) {
	var user models.User

	// 1. Find user by email
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid email or password")
		}
		return nil, errors.New("database error")
	}

	// 2. Compare the provided password with the stored hash
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, errors.New("invalid email or password") // Keep error ambiguous for security
	}

	return &user, nil
}
