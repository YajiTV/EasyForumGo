package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"ForumJS/pkg/utils"
	"ForumJS/pkg/validator"
)

type AuthHandler struct {
	db *sql.DB
}

func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{db: db}
}

func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée.", http.StatusMethodNotAllowed)
		return
	}

	input := validator.AuthInput{
		Email:    strings.TrimSpace(strings.ToLower(r.FormValue("email"))),
		Username: strings.TrimSpace(r.FormValue("username")),
		Password: r.FormValue("password"),
	}

	if validationErrors := validator.ValidateSignup(input); validationErrors.HasErrors() {
		writeValidationError(w, validationErrors)
		return
	}

	exists, err := h.userExists(input.Email, input.Username)
	if err != nil {
		http.Error(w, "Une erreur est survenue.", http.StatusInternalServerError)
		return
	}
	if exists {
		http.Error(w, "Cette adresse e-mail ou ce nom d'utilisateur est déjà utilisé.", http.StatusConflict)
		return
	}

	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		http.Error(w, "Une erreur est survenue.", http.StatusInternalServerError)
		return
	}

	if err := h.createUser(input.Email, input.Username, hashedPassword); err != nil {
		http.Error(w, "Une erreur est survenue.", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *AuthHandler) userExists(email, username string) (bool, error) {
	var id string
	err := h.db.QueryRow(
		"SELECT id FROM users WHERE email = ? OR username = ? LIMIT 1",
		email,
		username,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (h *AuthHandler) createUser(email, username, hashedPassword string) error {
	_, err := h.db.Exec(
		"INSERT INTO users (id, email, username, password) VALUES (?, ?, ?, ?)",
		utils.NewUUID(),
		email,
		username,
		hashedPassword,
	)
	return err
}

func writeValidationError(w http.ResponseWriter, validationErrors validator.ValidationErrors) {
	for _, message := range validationErrors {
		http.Error(w, message, http.StatusBadRequest)
		return
	}

	http.Error(w, "Formulaire invalide.", http.StatusBadRequest)
}
