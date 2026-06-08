package validator

import (
	"context"
	"net"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	MaxEmailLength       = 254
	MinUsernameLength    = 3
	MaxUsernameLength    = 30
	MinPasswordLength    = 12
	MinPostTitleLength   = 3
	MaxPostTitleLength   = 120
	MaxPostContentLength = 5000
	MaxCommentLength     = 1000
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
var domainLabelPattern = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)

var blockedEmailDomains = map[string]struct{}{
	"example.com": {},
	"example.net": {},
	"example.org": {},
	"invalid":     {},
	"localhost":   {},
	"local":       {},
	"test":        {},
}

var commonPasswords = map[string]struct{}{
	"12345678":      {},
	"123456789":     {},
	"1234567890":    {},
	"azerty123":     {},
	"azerty1234":    {},
	"password":      {},
	"password1":     {},
	"password123":   {},
	"motdepasse":    {},
	"motdepasse123": {},
	"qwerty123":     {},
	"qwerty1234":    {},
	"admin1234":     {},
	"forumjs123":    {},
	"letmein123":    {},
	"iloveyou123":   {},
	"bonjour123":    {},
	"bonjour1234":   {},
}

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

type PasswordStrength struct {
	Score          int
	MaxScore       int
	Percent        int
	Level          string
	HasMinLength   bool
	HasLowercase   bool
	HasUppercase   bool
	HasDigit       bool
	HasSpecial     bool
	NoPersonalInfo bool
	NotCommon      bool
	IsSecure       bool
}

type ValidationErrors map[string]string

// HasErrors checks whether validation failed
func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

// ValidateSignup validates the provided input
func ValidateSignup(input AuthInput) ValidationErrors {
	errors := ValidationErrors{}

	validateSignupEmail(input.Email, errors)

	validateUsername(input.Username, errors)
	validatePassword(input.Password, input.Username, input.Email, errors)
	if input.ConfirmPassword != input.Password {
		errors["confirm_password"] = "Les mots de passe ne correspondent pas."
	}

	return errors
}

// EvaluatePasswordStrength evaluates the provided input
func EvaluatePasswordStrength(password, username, email string) PasswordStrength {
	strength := PasswordStrength{
		MaxScore: 7,
		Level:    "Très faible",
	}
	if password == "" {
		return strength
	}

	passwordLength := utf8.RuneCountInString(password)
	strength.HasMinLength = passwordLength >= MinPasswordLength
	strength.NoPersonalInfo = !containsPersonalInfo(password, username, email)
	strength.NotCommon = !isCommonPassword(password)

	for _, r := range password {
		switch {
		case unicode.IsLower(r):
			strength.HasLowercase = true
		case unicode.IsUpper(r):
			strength.HasUppercase = true
		case unicode.IsDigit(r):
			strength.HasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			strength.HasSpecial = true
		}
	}

	conditions := []bool{
		strength.HasMinLength,
		strength.HasLowercase,
		strength.HasUppercase,
		strength.HasDigit,
		strength.HasSpecial,
		strength.NoPersonalInfo,
		strength.NotCommon,
	}
	for _, condition := range conditions {
		if condition {
			strength.Score++
		}
	}

	strength.Percent = strength.Score * 100 / strength.MaxScore
	switch {
	case strength.Score >= 7:
		strength.Level = "Sécurisé"
	case strength.Score >= 5:
		strength.Level = "Correct"
	case strength.Score >= 3:
		strength.Level = "Faible"
	}

	strength.IsSecure = strength.Score == strength.MaxScore
	return strength
}

// ValidateLogin validates the provided input
func ValidateLogin(input AuthInput) ValidationErrors {
	errors := ValidationErrors{}

	validateEmail(input.Email, errors)

	if strings.TrimSpace(input.Password) == "" {
		errors["password"] = "Le mot de passe est obligatoire."
	}

	return errors
}

// ValidatePost validates the provided input
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

// ValidateComment validates the provided input
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

// ValidateProfile validates the provided input
func ValidateProfile(input ProfileInput) ValidationErrors {
	errors := ValidationErrors{}
	validateUsername(input.Username, errors)
	return errors
}

// validateEmail validates the provided input
func validateEmail(email string, errors ValidationErrors) {
	validateEmailSyntax(email, errors)
}

// validateSignupEmail validates the provided input
func validateSignupEmail(email string, errors ValidationErrors) {
	if validateEmailSyntax(email, errors) {
		validateEmailDomain(email, errors)
	}
}

// validateEmailSyntax validates the provided input
func validateEmailSyntax(email string, errors ValidationErrors) bool {
	email = strings.TrimSpace(email)
	switch {
	case email == "":
		errors["email"] = "L'adresse e-mail est obligatoire."
	case len(email) > MaxEmailLength:
		errors["email"] = "L'adresse e-mail est trop longue."
	default:
		address, err := mail.ParseAddress(email)
		if err != nil || address.Address != email {
			errors["email"] = "L'adresse e-mail est invalide."
		}
	}

	return errors["email"] == ""
}

// validateEmailDomain validates the provided input
func validateEmailDomain(email string, validationErrors ValidationErrors) {
	domain := emailDomain(email)
	if domain == "" || !isPlausibleEmailDomainName(domain) {
		validationErrors["email"] = "L'adresse e-mail doit utiliser un domaine valide."
		return
	}

	if _, blocked := blockedEmailDomains[domain]; blocked {
		validationErrors["email"] = "L'adresse e-mail doit utiliser un domaine réel."
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	mxRecords, err := net.DefaultResolver.LookupMX(ctx, domain)
	if err == nil && len(mxRecords) > 0 {
		return
	}

	if _, hostErr := net.DefaultResolver.LookupHost(ctx, domain); hostErr == nil {
		return
	}

	if ctx.Err() == context.DeadlineExceeded {
		validationErrors["email"] = "La vérification du domaine e-mail a expiré. Réessayez."
		return
	}

	validationErrors["email"] = "Le domaine de cette adresse e-mail ne semble pas recevoir d'e-mails."
}

// validateUsername validates the provided input
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

// validatePassword validates the provided input
func validatePassword(password, username, email string, errors ValidationErrors) {
	passwordLength := utf8.RuneCountInString(password)
	switch {
	case strings.TrimSpace(password) == "":
		errors["password"] = "Le mot de passe est obligatoire."
	case passwordLength < MinPasswordLength:
		errors["password"] = "Le mot de passe doit contenir au moins 12 caractères."
	default:
		strength := EvaluatePasswordStrength(password, username, email)
		if !strength.IsSecure {
			errors["password"] = "Le mot de passe n'est pas assez sécurisé."
		}
	}
}

// emailDomain gets the domain from an email address
func emailDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}

	return strings.TrimSuffix(strings.ToLower(parts[1]), ".")
}

// isPlausibleEmailDomainName checks whether an email domain is plausible
func isPlausibleEmailDomainName(domain string) bool {
	if len(domain) < 4 || len(domain) > 253 || strings.Contains(domain, "..") || !strings.Contains(domain, ".") {
		return false
	}

	labels := strings.Split(domain, ".")
	for _, label := range labels {
		if !domainLabelPattern.MatchString(label) {
			return false
		}
	}

	tld := labels[len(labels)-1]
	return len(tld) >= 2 && !strings.ContainsAny(tld, "0123456789-")
}

// containsPersonalInfo checks whether a password contains personal information
func containsPersonalInfo(password, username, email string) bool {
	normalizedPassword := strings.ToLower(password)
	normalizedUsername := strings.ToLower(strings.TrimSpace(username))
	if normalizedUsername != "" && len(normalizedUsername) >= 3 && strings.Contains(normalizedPassword, normalizedUsername) {
		return true
	}

	localPart := strings.Split(strings.ToLower(strings.TrimSpace(email)), "@")[0]
	return localPart != "" && len(localPart) >= 3 && strings.Contains(normalizedPassword, localPart)
}

// isCommonPassword checks whether a password is common
func isCommonPassword(password string) bool {
	normalizedPassword := strings.ToLower(strings.TrimSpace(password))
	if _, common := commonPasswords[normalizedPassword]; common {
		return true
	}

	return false
}
