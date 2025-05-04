package auth

import (
	"golang.org/x/crypto/bcrypt"
)

// PasswordHasher defines the interface for hashing and verifying passwords.
type PasswordHasher interface {
	HashPassword(password string) (string, error)
	CheckPassword(hashedPassword, password string) error
}

// BcryptPasswordHasher implements PasswordHasher using bcrypt.
type BcryptPasswordHasher struct{}

// NewBcryptPasswordHasher creates a new instance of BcryptPasswordHasher.
func NewBcryptPasswordHasher() PasswordHasher {
	return &BcryptPasswordHasher{}
}

// HashPassword hashes the given password using bcrypt.
func (h *BcryptPasswordHasher) HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// CheckPassword compares a hashed password with a plaintext password.
func (h *BcryptPasswordHasher) CheckPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
