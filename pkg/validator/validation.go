package validator

import (
	"net/mail"
	"strings"
	"unicode/utf8"
)

const (
	MaxEmailLength    = 254
	MinUsernameLength = 3
	MaxUsernameLength = 30
	MinPasswordLength = 8
)

type AuthInput struct {
	Email           string
	Username        string
	Password        string
	ConfirmPassword string
}

type ValidationErrors map[string]string

func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

func ValidateSignup(input AuthInput) ValidationErrors {
	errors := ValidationErrors{}

	validateEmail(input.Email, errors)

	username := strings.TrimSpace(input.Username)
	usernameLength := utf8.RuneCountInString(username)
	switch {
	case username == "":
		errors["username"] = "Le nom d'utilisateur est obligatoire."
	case usernameLength < MinUsernameLength:
		errors["username"] = "Le nom d'utilisateur doit contenir au moins 3 caractères."
	case usernameLength > MaxUsernameLength:
		errors["username"] = "Le nom d'utilisateur doit contenir au maximum 30 caractères."
	}

	validatePassword(input.Password, errors)
	if input.ConfirmPassword != input.Password {
		errors["confirm_password"] = "Les mots de passe ne correspondent pas."
	}

	return errors
}

func ValidateLogin(input AuthInput) ValidationErrors {
	errors := ValidationErrors{}

	validateEmail(input.Email, errors)

	if strings.TrimSpace(input.Password) == "" {
		errors["password"] = "Le mot de passe est obligatoire."
	}

	return errors
}

func validateEmail(email string, errors ValidationErrors) {
	email = strings.TrimSpace(email)
	switch {
	case email == "":
		errors["email"] = "L'adresse e-mail est obligatoire."
	case len(email) > MaxEmailLength:
		errors["email"] = "L'adresse e-mail est trop longue."
	default:
		if _, err := mail.ParseAddress(email); err != nil {
			errors["email"] = "L'adresse e-mail est invalide."
		}
	}
}

func validatePassword(password string, errors ValidationErrors) {
	passwordLength := utf8.RuneCountInString(password)
	switch {
	case strings.TrimSpace(password) == "":
		errors["password"] = "Le mot de passe est obligatoire."
	case passwordLength < MinPasswordLength:
		errors["password"] = "Le mot de passe doit contenir au moins 8 caractères."
	}
}
