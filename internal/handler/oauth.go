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
)

type OAuthHandler struct {
	db              *sql.DB
	clientID        string
	clientSecret    string
	redirectURL     string
	sessionDuration time.Duration
	errors          *ErrorRenderer
}

type googleUserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func NewOAuthHandler(db *sql.DB, clientID, clientSecret, redirectURL string, sessionDuration time.Duration, errors *ErrorRenderer) *OAuthHandler {
	return &OAuthHandler{
		db:              db,
		clientID:        clientID,
		clientSecret:    clientSecret,
		redirectURL:     redirectURL,
		sessionDuration: sessionDuration,
		errors:          errors,
	}
}

// GoogleLogin redirige vers Google
func (h *OAuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	// Générer un état aléatoire pour prévenir les attaques CSRF
	state, err := generateState()
	if err != nil {
		h.errors.InternalServerError(w)
		return
	}

	// Stocker l'état dans un cookie temporaire
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    state,
		Path:     "/",
		MaxAge:   300, // 5 minutes
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// Construire l'URL Google
	params := url.Values{}
	params.Set("client_id", h.clientID)
	params.Set("redirect_uri", h.redirectURL)
	params.Set("response_type", "code")
	params.Set("scope", "openid email profile")
	params.Set("state", state)

	http.Redirect(w, r, googleAuthURL+"?"+params.Encode(), http.StatusTemporaryRedirect)
}

// GoogleCallback traite le retour de Google
func (h *OAuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	// Vérifier le state anti-CSRF
	stateCookie, err := r.Cookie(oauthStateCookie)
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		h.errors.Forbidden(w, "État OAuth invalide.")
		return
	}
	// Supprimer le cookie d'état
	http.SetCookie(w, &http.Cookie{
		Name:   oauthStateCookie,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		h.errors.BadRequest(w, "Code OAuth manquant.")
		return
	}

	// Échanger le code contre un token
	token, err := h.exchangeCode(code)
	if err != nil {
		h.errors.InternalServerError(w)
		return
	}

	// Récupérer les infos utilisateur
	userInfo, err := h.fetchUserInfo(token)
	if err != nil {
		h.errors.InternalServerError(w)
		return
	}

	// Trouver ou créer l'utilisateur
	userID, err := h.findOrCreateUser(userInfo)
	if err != nil {
		h.errors.InternalServerError(w)
		return
	}

	// Créer la session
	sessionToken, expiresAt, err := h.createSession(userID)
	if err != nil {
		h.errors.InternalServerError(w)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName, // même constante que dans auth.go
		Value:    sessionToken,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// exchangeCode échange le code contre un access token Google
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

// fetchUserInfo récupère les infos du compte Google
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

// findOrCreateUser trouve ou crée l'utilisateur en base
func (h *OAuthHandler) findOrCreateUser(info googleUserInfo) (string, error) {
	// Chercher par oauth_id
	var userID string
	err := h.db.QueryRow(
		"SELECT id FROM users WHERE oauth_provider = 'google' AND oauth_id = ? LIMIT 1",
		info.ID,
	).Scan(&userID)
	if err == nil {
		return userID, nil // utilisateur existant
	}
	if err != sql.ErrNoRows {
		return "", err
	}

	// Chercher par email (compte existant sans OAuth)
	err = h.db.QueryRow(
		"SELECT id FROM users WHERE email = ? LIMIT 1",
		info.Email,
	).Scan(&userID)
	if err == nil {
		// Lier le compte existant à Google
		_, err = h.db.Exec(
			"UPDATE users SET oauth_provider = 'google', oauth_id = ? WHERE id = ?",
			info.ID, userID,
		)
		return userID, err
	}
	if err != sql.ErrNoRows {
		return "", err
	}

	// Créer un nouvel utilisateur
	username := h.uniqueUsername(info.Name)
	userID = utils.NewUUID()
	// Mot de passe inutilisable (l'utilisateur se connecte via Google)
	fakePassword := "$oauth$" + utils.NewUUID()

	_, err = h.db.Exec(
		"INSERT INTO users (id, email, username, password, oauth_provider, oauth_id) VALUES (?, ?, ?, ?, 'google', ?)",
		userID, info.Email, username, fakePassword, info.ID,
	)
	return userID, err
}

// uniqueUsername génère un username unique
func (h *OAuthHandler) uniqueUsername(name string) string {
	base := strings.ReplaceAll(strings.ToLower(name), " ", "_")
	if base == "" {
		base = "user"
	}
	username := base
	for i := 2; ; i++ {
		var id string
		err := h.db.QueryRow("SELECT id FROM users WHERE username = ? LIMIT 1", username).Scan(&id)
		if err == sql.ErrNoRows {
			break
		}
		username = fmt.Sprintf("%s_%d", base, i)
	}
	return username
}

// createSession crée une session (même logique que dans auth.go)
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

// generateState génère un token aléatoire anti-CSRF
func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
