package entity

import (
	"net/mail"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/apperror"
	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	DisplayName  string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func ValidateRegistrationData(
	email string,
	password string,
	displayName string,
) (string, string, error) {
	email, err := NormalizeEmail(email)
	if err != nil {
		return "", "", err
	}

	displayName = strings.TrimSpace(displayName)
	if utf8.RuneCountInString(displayName) < 1 {
		return "", "", apperror.ErrInvalidDisplayName
	}

	if err = ValidatePassword(password); err != nil {
		return "", "", err
	}

	return email, displayName, nil
}

func NormalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return "", apperror.ErrInvalidEmail
	}

	return email, nil
}

func ValidatePassword(password string) error {
	if len(password) < 8 || len(password) > 72 {
		return apperror.ErrWeakPassword
	}

	var hasUpper, hasSpecial bool
	for _, symbol := range password {
		hasUpper = hasUpper || unicode.IsUpper(symbol)
		hasSpecial = hasSpecial ||
			unicode.IsPunct(symbol) || unicode.IsSymbol(symbol)
	}
	if !hasUpper || !hasSpecial {
		return apperror.ErrWeakPassword
	}

	return nil
}
