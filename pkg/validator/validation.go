package validator

import (
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	MaxEmailLength       = 254
	MinUsernameLength    = 3
	MaxUsernameLength    = 30
	MinPasswordLength    = 8
	MinPostTitleLength   = 3
	MaxPostTitleLength   = 120
	MaxPostContentLength = 5000
	MaxCommentLength     = 1000
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type AuthInput struct {
	Email           string
	Username        string
	Password        string
	ConfirmPassword string
}

type PostInput struct {
	Title       string
	Content     string
	CategoryIDs []string
}

type CommentInput struct {
	Content string
}

type ProfileInput struct {
	Username string
}

type ValidationErrors map[string]string

func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

func ValidateSignup(input AuthInput) ValidationErrors {
	errors := ValidationErrors{}

	validateEmail(input.Email, errors)

	validateUsername(input.Username, errors)
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

func ValidatePost(input PostInput) ValidationErrors {
	errors := ValidationErrors{}

	title := strings.TrimSpace(input.Title)
	titleLength := utf8.RuneCountInString(title)
	switch {
	case title == "":
		errors["title"] = "Le titre est obligatoire."
	case titleLength < MinPostTitleLength:
		errors["title"] = "Le titre doit contenir au moins 3 caractères."
	case titleLength > MaxPostTitleLength:
		errors["title"] = "Le titre ne peut pas dépasser 120 caractères."
	}

	content := strings.TrimSpace(input.Content)
	contentLength := utf8.RuneCountInString(content)
	switch {
	case content == "":
		errors["content"] = "Le contenu est obligatoire."
	case contentLength > MaxPostContentLength:
		errors["content"] = "Le contenu ne peut pas dépasser 5000 caractères."
	}

	if len(input.CategoryIDs) == 0 {
		errors["categories"] = "Sélectionnez au moins une catégorie."
	}

	return errors
}

func ValidateComment(input CommentInput) ValidationErrors {
	errors := ValidationErrors{}

	content := strings.TrimSpace(input.Content)
	contentLength := utf8.RuneCountInString(content)
	switch {
	case content == "":
		errors["content"] = "Le commentaire est obligatoire."
	case contentLength > MaxCommentLength:
		errors["content"] = "Le commentaire ne peut pas dépasser 1000 caractères."
	}

	return errors
}

func ValidateProfile(input ProfileInput) ValidationErrors {
	errors := ValidationErrors{}
	validateUsername(input.Username, errors)
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

func validateUsername(username string, errors ValidationErrors) {
	username = strings.TrimSpace(username)
	usernameLength := utf8.RuneCountInString(username)
	switch {
	case username == "":
		errors["username"] = "Le nom d'utilisateur est obligatoire."
	case usernameLength < MinUsernameLength:
		errors["username"] = "Le nom d'utilisateur doit contenir au moins 3 caractères."
	case usernameLength > MaxUsernameLength:
		errors["username"] = "Le nom d'utilisateur doit contenir au maximum 30 caractères."
	case !usernamePattern.MatchString(username):
		errors["username"] = "Le nom d'utilisateur peut contenir uniquement lettres, chiffres, tirets et underscores."
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
