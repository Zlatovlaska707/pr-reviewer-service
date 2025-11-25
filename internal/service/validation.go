package service

import (
	"errors"
	"strings"
)

var (
	// ErrInvalidInput ошибка валидации входных данных
	ErrInvalidInput = errors.New("invalid input")
)

// ValidateTeamName проверяет корректность имени команды.
func ValidateTeamName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("team name cannot be empty")
	}
	if len(name) > 100 {
		return errors.New("team name too long (max 100 characters)")
	}
	return nil
}

// ValidateUserID проверяет корректность ID пользователя.
func ValidateUserID(userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("user ID cannot be empty")
	}
	if len(userID) > 100 {
		return errors.New("user ID too long (max 100 characters)")
	}
	return nil
}

// ValidatePRID проверяет корректность ID PR.
func ValidatePRID(prID string) error {
	prID = strings.TrimSpace(prID)
	if prID == "" {
		return errors.New("PR ID cannot be empty")
	}
	if len(prID) > 200 {
		return errors.New("PR ID too long (max 200 characters)")
	}
	return nil
}

// ValidatePRName проверяет корректность имени PR.
func ValidatePRName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("PR name cannot be empty")
	}
	if len(name) > 500 {
		return errors.New("PR name too long (max 500 characters)")
	}
	return nil
}
