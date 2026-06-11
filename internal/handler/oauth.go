package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ForumJS/pkg/utils"
)

const (
	googleAuthURL    = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL   = "https://oauth2.googleapis.com/token"
	googleUserURL    = "https://www.googleapis.com/oauth2/v2/userinfo"
	oauthStateCookie = "oauth_state"
	oauthPendingCookie = "oauth_pending"
)

type OAuthHandler struct {
	db              *sql.DB
	clientID        string
	clientSecret    string
	redirectURL     string
	sessionDuration time.Duration
	errors          *ErrorRenderer
	renderer        *PageRenderer
}

// googleUserInfo contient les données renvoyées par l'API Google
type googleUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Picture string `json:"picture"`
}

// pendingOAuthUser stocke temporairement les données Google
// le temps que l'utilisateur choisisse son username
type pendingOAuthUser struct {
	GoogleID string `json:"google_id"`
	Email    string `json:"email"`
	Picture  string `json:"picture"`
}

func NewOAuthHandler(db *sql.DB, clientID, clientSecret, redirectURL string, sessionDuration time.Duration, errors *ErrorRenderer, renderer *PageRenderer) *OAuthHandler {
	return &OAuthHandler{
		db:              db,
		clientID:        clientID,
		clientSecret:    clientSecret,
		redirectURL:     redirectURL,
		sessionDuration: sessionDuration,
		errors:          errors,
		renderer:        renderer,
	}
}

// GoogleLogin redirige l'utilisateur vers la page de connexion Google
func (h *OAuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	state, err := generateState()
	if err != nil {
		h.errors.InternalServerError(w)
		return
	}

	// Cookie anti-CSRF valable 5 minutes
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    state,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	params := url.Values{}
	params.Set("client_id", h.clientID)
	params.Set("redirect_uri", h.redirectURL)
	params.Set("response_type", "code")
	params.Set("scope", "openid email profile")
	params.Set("state", state)

	http.Redirect(w, r, googleAuthURL+"?"+params.Encode(), http.StatusTemporaryRedirect)
}

// GoogleCallback traite le retour de Google après authentification
func (h *OAuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	// Vérifier le state anti-CSRF
	stateCookie, err := r.Cookie(oauthStateCookie)
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		h.errors.Forbidden(w, "État OAuth invalide.")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: oauthStateCookie, Value: "", Path: "/", MaxAge: -1})

	code := r.URL.Query().Get("code")
	if code == "" {
		h.errors.BadRequest(w, "Code OAuth manquant.")
		return
	}

	// Étape 1 : échanger le code contre un access token
	token, err := h.exchangeCode(code)
	if err != nil {
		h.errors.InternalServerError(w)
		return
	}

	// Étape 2 : récupérer les infos du compte Google
	userInfo, err := h.fetchUserInfo(token)
	if err != nil {
		h.errors.InternalServerError(w)
		return
	}

	// Étape 3 : chercher si l'utilisateur existe déjà
	userID, found, err := h.findExistingUser(userInfo)
	if err != nil {
		h.errors.InternalServerError(w)
		return
	}

	if found {
		// Utilisateur connu → on crée la session directement
		h.startSession(w, userID)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Nouvel utilisateur → on stocke ses données dans un cookie temporaire
	// et on lui demande de choisir un username
	pending := pendingOAuthUser{
		GoogleID: userInfo.ID,
		Email:    userInfo.Email,
		Picture:  userInfo.Picture,
	}
	data, err := json.Marshal(pending)
	if err != nil {
		h.errors.InternalServerError(w)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oauthPendingCookie,
		Value:    base64.URLEncoding.EncodeToString(data),
		Path:     "/",
		MaxAge:   900, // 15 minutes pour choisir son username
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/auth/complete-profile", http.StatusSeeOther)
}

// ShowCompleteProfile affiche le formulaire de choix du username
func (h *OAuthHandler) ShowCompleteProfile(w http.ResponseWriter, r *http.Request) {
	pending, ok := h.readPendingCookie(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	renderAuthTemplate(h.renderer, w, "complete_profile.html", map[string]any{
		"Email": pending.Email,
		"Error": "",
	})
}

// CompleteProfile traite le formulaire : crée le compte et démarre la session
func (h *OAuthHandler) CompleteProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.errors.MethodNotAllowed(w, "Méthode non autorisée.")
		return
	}

	pending, ok := h.readPendingCookie(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))

	// Validation simple du username
	if len(username) < 3 || len(username) > 30 {
		renderAuthTemplate(h.renderer, w, "complete_profile.html", map[string]any{
			"Email": pending.Email,
			"Error": "Le username doit faire entre 3 et 30 caractères.",
		})
		return
	}

	// Vérifier que le username n'est pas déjà pris
	var existingID string
	err := h.db.QueryRow("SELECT id FROM users WHERE username = ? LIMIT 1", username).Scan(&existingID)
	if err != sql.ErrNoRows {
		renderAuthTemplate(h.renderer, w, "complete_profile.html", map[string]any{
			"Email": pending.Email,
			"Error": "Ce username est déjà pris.",
		})
		return
	}

	// Créer le compte — le mot de passe est inutilisable volontairement
	userID := utils.NewUUID()
	fakePassword := "$oauth$" + utils.NewUUID()
	_, err = h.db.Exec(
		"INSERT INTO users (id, email, username, password, profile_picture, oauth_provider, oauth_id) VALUES (?, ?, ?, ?, ?, 'google', ?)",
		userID, pending.Email, username, fakePassword, pending.Picture, pending.GoogleID,
	)
	if err != nil {
		h.errors.InternalServerError(w)
		return
	}

	// Supprimer le cookie temporaire
	http.SetCookie(w, &http.Cookie{Name: oauthPendingCookie, Value: "", Path: "/", MaxAge: -1})

	// Démarrer la session
	h.startSession(w, userID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// findExistingUser cherche un utilisateur par oauth_id ou par email
func (h *OAuthHandler) findExistingUser(info googleUserInfo) (string, bool, error) {
	var userID string

	// Chercher par oauth_id (connexion Google déjà utilisée)
	err := h.db.QueryRow(
		"SELECT id FROM users WHERE oauth_provider = 'google' AND oauth_id = ? LIMIT 1",
		info.ID,
	).Scan(&userID)
	if err == nil {
		return userID, true, nil
	}
	if err != sql.ErrNoRows {
		return "", false, err
	}

	// Chercher par email (compte classique existant)
	err = h.db.QueryRow(
		"SELECT id FROM users WHERE email = ? LIMIT 1",
		info.Email,
	).Scan(&userID)
	if err == nil {
		// Lier le compte Google à ce compte existant
		_, err = h.db.Exec(
			"UPDATE users SET oauth_provider = 'google', oauth_id = ?, profile_picture = CASE WHEN profile_picture = '' THEN ? ELSE profile_picture END WHERE id = ?",
			info.ID, info.Picture, userID,
		)
		return userID, true, err
	}
	if err != sql.ErrNoRows {
		return "", false, err
	}

	return "", false, nil
}

// startSession crée un cookie de session pour l'utilisateur
func (h *OAuthHandler) startSession(w http.ResponseWriter, userID string) {
	sessionToken, expiresAt, err := h.createSession(userID)
	if err != nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionToken,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// readPendingCookie lit et décode le cookie temporaire OAuth
func (h *OAuthHandler) readPendingCookie(r *http.Request) (pendingOAuthUser, bool) {
	cookie, err := r.Cookie(oauthPendingCookie)
	if err != nil {
		return pendingOAuthUser{}, false
	}
	data, err := base64.URLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return pendingOAuthUser{}, false
	}
	var pending pendingOAuthUser
	if err := json.Unmarshal(data, &pending); err != nil {
		return pendingOAuthUser{}, false
	}
	return pending, true
}

// exchangeCode échange le code d'autorisation contre un access token Google
func (h *OAuthHandler) exchangeCode(code string) (string, error) {
	resp, err := http.PostForm(googleTokenURL, url.Values{
		"code":          {code},
		"client_id":     {h.clientID},
		"client_secret": {h.clientSecret},
		"redirect_uri":  {h.redirectURL},
		"grant_type":    {"authorization_code"},
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	accessToken, ok := result["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("access_token manquant dans la réponse Google")
	}
	return accessToken, nil
}

// fetchUserInfo appelle l'API Google pour récupérer les infos du compte
func (h *OAuthHandler) fetchUserInfo(accessToken string) (googleUserInfo, error) {
	req, err := http.NewRequest("GET", googleUserURL, nil)
	if err != nil {
		return googleUserInfo{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return googleUserInfo{}, err
	}
	defer resp.Body.Close()

	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return googleUserInfo{}, err
	}
	return info, nil
}

// createSession crée une nouvelle session en base de données
func (h *OAuthHandler) createSession(userID string) (string, time.Time, error) {
	sessionToken := utils.NewSessionToken()
	expiresAt := time.Now().Add(h.sessionDuration)

	tx, err := h.db.Begin()
	if err != nil {
		return "", time.Time{}, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM sessions WHERE user_id = ?", userID); err != nil {
		return "", time.Time{}, err
	}
	_, err = tx.Exec(
		"INSERT INTO sessions (id, user_id, session_token, expires_at) VALUES (?, ?, ?, ?)",
		utils.NewUUID(), userID, sessionToken, expiresAt.UTC(),
	)
	if err != nil {
		return "", time.Time{}, err
	}

	return sessionToken, expiresAt, tx.Commit()
}

// generateState génère un token aléatoire pour la protection CSRF
func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
