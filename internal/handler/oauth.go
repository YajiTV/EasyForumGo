package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"EasyForumGo/pkg/utils"
	"EasyForumGo/pkg/validator"
)

const (
	googleAuthURL      = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL     = "https://oauth2.googleapis.com/token"
	googleUserURL      = "https://www.googleapis.com/oauth2/v2/userinfo"
	gitHubAuthURL      = "https://github.com/login/oauth/authorize"
	gitHubTokenURL     = "https://github.com/login/oauth/access_token"
	gitHubUserURL      = "https://api.github.com/user"
	gitHubEmailsURL    = "https://api.github.com/user/emails"
	oauthPendingCookie = "oauth_pending"
)

type OAuthConfig struct {
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	GitHubClientID     string
	GitHubClientSecret string
	GitHubRedirectURL  string
}

type OAuthHandler struct {
	db              *sql.DB
	config          OAuthConfig
	sessionDuration time.Duration
	errors          *ErrorRenderer
	renderer        *PageRenderer
	client          *http.Client
}

type oauthIdentity struct {
	Provider       string
	ProviderUserID string
	Email          string
	Picture        string
	SuggestedName  string
}

type pendingOAuthUser struct {
	Token          string `json:"-"`
	Provider       string `json:"provider"`
	ProviderUserID string `json:"provider_user_id"`
	Email          string `json:"email"`
	Picture        string `json:"picture"`
	SuggestedName  string `json:"suggested_name"`
}

type googleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Picture       string `json:"picture"`
	VerifiedEmail bool   `json:"verified_email"`
}

type gitHubUserInfo struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type gitHubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func NewOAuthHandler(db *sql.DB, config OAuthConfig, sessionDuration time.Duration, errorRenderer *ErrorRenderer, renderer *PageRenderer) *OAuthHandler {
	return &OAuthHandler{
		db:              db,
		config:          config,
		sessionDuration: sessionDuration,
		errors:          errorRenderer,
		renderer:        renderer,
		client:          &http.Client{Timeout: 15 * time.Second},
	}
}

func (h *OAuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	h.beginOAuth(w, r, "google", h.config.GoogleClientID, h.config.GoogleClientSecret, h.config.GoogleRedirectURL, googleAuthURL, "openid email profile")
}

func (h *OAuthHandler) GitHubLogin(w http.ResponseWriter, r *http.Request) {
	h.beginOAuth(w, r, "github", h.config.GitHubClientID, h.config.GitHubClientSecret, h.config.GitHubRedirectURL, gitHubAuthURL, "read:user user:email")
}

func (h *OAuthHandler) beginOAuth(w http.ResponseWriter, r *http.Request, provider, clientID, clientSecret, redirectURL, authURL, scope string) {
	if clientID == "" || clientSecret == "" {
		h.errors.RenderWithRequest(w, r, http.StatusServiceUnavailable, "Ce fournisseur de connexion n'est pas encore configuré.")
		return
	}
	state, err := generateState()
	if err != nil {
		log.Printf("generate %s oauth state: %v", provider, err)
		h.errors.InternalServerError(w)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie(provider),
		Value:    state,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	params := url.Values{
		"client_id":    {clientID},
		"redirect_uri": {redirectURL},
		"scope":        {scope},
		"state":        {state},
	}
	if provider == "google" {
		params.Set("response_type", "code")
	}
	http.Redirect(w, r, authURL+"?"+params.Encode(), http.StatusTemporaryRedirect)
}

func (h *OAuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	h.finishOAuth(w, r, "google", h.config.GoogleClientID, h.config.GoogleClientSecret, h.config.GoogleRedirectURL, h.fetchGoogleIdentity)
}

func (h *OAuthHandler) GitHubCallback(w http.ResponseWriter, r *http.Request) {
	h.finishOAuth(w, r, "github", h.config.GitHubClientID, h.config.GitHubClientSecret, h.config.GitHubRedirectURL, h.fetchGitHubIdentity)
}

func (h *OAuthHandler) finishOAuth(w http.ResponseWriter, r *http.Request, provider, clientID, clientSecret, redirectURL string, fetchIdentity func(string) (oauthIdentity, error)) {
	stateCookie, err := r.Cookie(oauthStateCookie(provider))
	if err != nil || stateCookie.Value == "" || stateCookie.Value != r.URL.Query().Get("state") {
		h.errors.Forbidden(w, "État OAuth invalide.")
		return
	}
	clearCookie(w, oauthStateCookie(provider))

	if providerError := strings.TrimSpace(r.URL.Query().Get("error")); providerError != "" {
		h.errors.BadRequest(w, "La connexion avec "+providerLabel(provider)+" a été annulée ou refusée.")
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		h.errors.BadRequest(w, "Code OAuth manquant.")
		return
	}
	tokenURL := googleTokenURL
	if provider == "github" {
		tokenURL = gitHubTokenURL
	}
	token, err := h.exchangeCode(tokenURL, code, clientID, clientSecret, redirectURL)
	if err != nil {
		log.Printf("%s oauth token exchange: %v", provider, err)
		h.errors.InternalServerError(w)
		return
	}
	identity, err := fetchIdentity(token)
	if err != nil {
		log.Printf("%s oauth user info: %v", provider, err)
		h.errors.RenderWithRequest(w, r, http.StatusBadGateway, "Impossible de récupérer un e-mail vérifié auprès de "+providerLabel(provider)+".")
		return
	}
	userID, found, err := h.findOrLinkExistingUser(identity)
	if err != nil {
		log.Printf("find or link %s oauth identity: %v", provider, err)
		h.errors.InternalServerError(w)
		return
	}
	if found {
		if err := h.startSession(w, userID); err != nil {
			log.Printf("start oauth session for user %q: %v", userID, err)
			h.errors.InternalServerError(w)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if err := h.setPendingCookie(w, identity); err != nil {
		log.Printf("store pending oauth identity: %v", err)
		h.errors.InternalServerError(w)
		return
	}
	http.Redirect(w, r, "/auth/complete-profile", http.StatusSeeOther)
}

func (h *OAuthHandler) ShowCompleteProfile(w http.ResponseWriter, r *http.Request) {
	pending, ok := h.readPendingCookie(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	h.renderCompleteProfile(w, pending, "")
}

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
	if validationErrors := validator.ValidateProfile(validator.ProfileInput{Username: username}); validationErrors.HasErrors() {
		h.renderCompleteProfile(w, pending, firstValidationMessage(validationErrors))
		return
	}
	var existingID string
	err := h.db.QueryRow("SELECT id FROM users WHERE username = ? LIMIT 1", username).Scan(&existingID)
	if err == nil {
		h.renderCompleteProfile(w, pending, "Ce nom d'utilisateur est déjà pris.")
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		log.Printf("check oauth username availability: %v", err)
		h.errors.InternalServerError(w)
		return
	}
	userID, err := h.createOAuthUser(pending, username)
	if err != nil {
		log.Printf("create %s oauth user: %v", pending.Provider, err)
		h.errors.InternalServerError(w)
		return
	}
	clearCookie(w, oauthPendingCookie)
	if err := h.startSession(w, userID); err != nil {
		log.Printf("start new oauth user session: %v", err)
		h.errors.InternalServerError(w)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *OAuthHandler) renderCompleteProfile(w http.ResponseWriter, pending pendingOAuthUser, message string) {
	renderAuthTemplate(h.renderer, w, "complete_profile.html", map[string]any{
		"Email":         pending.Email,
		"Provider":      providerLabel(pending.Provider),
		"SuggestedName": pending.SuggestedName,
		"Error":         message,
	})
}

func (h *OAuthHandler) findOrLinkExistingUser(identity oauthIdentity) (string, bool, error) {
	var userID string
	err := h.db.QueryRow(
		"SELECT user_id FROM oauth_identities WHERE provider = ? AND provider_user_id = ? LIMIT 1",
		identity.Provider, identity.ProviderUserID,
	).Scan(&userID)
	if err == nil {
		return userID, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", false, err
	}
	err = h.db.QueryRow("SELECT id FROM users WHERE lower(email) = lower(?) LIMIT 1", identity.Email).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	tx, err := h.db.Begin()
	if err != nil {
		return "", false, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		"INSERT INTO oauth_identities (id, user_id, provider, provider_user_id) VALUES (?, ?, ?, ?)",
		utils.NewUUID(), userID, identity.Provider, identity.ProviderUserID,
	); err != nil {
		return "", false, err
	}
	if _, err := tx.Exec(
		"UPDATE users SET profile_picture = CASE WHEN profile_picture = '' THEN ? ELSE profile_picture END WHERE id = ?",
		identity.Picture, userID,
	); err != nil {
		return "", false, err
	}
	return userID, true, tx.Commit()
}

func (h *OAuthHandler) createOAuthUser(pending pendingOAuthUser, username string) (string, error) {
	userID := utils.NewUUID()
	tx, err := h.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		"INSERT INTO users (id, email, username, password, profile_picture, oauth_provider, oauth_id) VALUES (?, ?, ?, ?, ?, ?, ?)",
		userID, pending.Email, username, "$oauth$"+utils.NewUUID(), pending.Picture, pending.Provider, pending.ProviderUserID,
	); err != nil {
		return "", err
	}
	if _, err := tx.Exec(
		"INSERT INTO oauth_identities (id, user_id, provider, provider_user_id) VALUES (?, ?, ?, ?)",
		utils.NewUUID(), userID, pending.Provider, pending.ProviderUserID,
	); err != nil {
		return "", err
	}
	if _, err := tx.Exec("DELETE FROM oauth_pending_flows WHERE token = ?", pending.Token); err != nil {
		return "", err
	}
	return userID, tx.Commit()
}

func (h *OAuthHandler) fetchGoogleIdentity(accessToken string) (oauthIdentity, error) {
	var info googleUserInfo
	if err := h.getJSON(googleUserURL, accessToken, &info); err != nil {
		return oauthIdentity{}, err
	}
	if info.ID == "" || info.Email == "" || !info.VerifiedEmail {
		return oauthIdentity{}, errors.New("google account has no verified email")
	}
	return oauthIdentity{Provider: "google", ProviderUserID: info.ID, Email: strings.ToLower(info.Email), Picture: info.Picture}, nil
}

func (h *OAuthHandler) fetchGitHubIdentity(accessToken string) (oauthIdentity, error) {
	var info gitHubUserInfo
	if err := h.getJSON(gitHubUserURL, accessToken, &info); err != nil {
		return oauthIdentity{}, err
	}
	var emails []gitHubEmail
	if err := h.getJSON(gitHubEmailsURL, accessToken, &emails); err != nil {
		return oauthIdentity{}, err
	}
	email := ""
	for _, candidate := range emails {
		if candidate.Primary && candidate.Verified {
			email = candidate.Email
			break
		}
		if email == "" && candidate.Verified {
			email = candidate.Email
		}
	}
	if info.ID == 0 || email == "" {
		return oauthIdentity{}, errors.New("github account has no verified email")
	}
	return oauthIdentity{
		Provider:       "github",
		ProviderUserID: fmt.Sprintf("%d", info.ID),
		Email:          strings.ToLower(email),
		Picture:        info.AvatarURL,
		SuggestedName:  info.Login,
	}, nil
}

func (h *OAuthHandler) exchangeCode(tokenURL, code, clientID, clientSecret, redirectURL string) (string, error) {
	form := url.Values{
		"code":          {code},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"redirect_uri":  {redirectURL},
		"grant_type":    {"authorization_code"},
	}
	req, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := h.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("token endpoint returned %s", resp.Status)
	}
	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("access token missing: %s", result.Error)
	}
	return result.AccessToken, nil
}

func (h *OAuthHandler) getJSON(endpoint, accessToken string, destination any) error {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json, application/json")
	req.Header.Set("User-Agent", "EasyForumGo")
	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("user endpoint returned %s", resp.Status)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(destination)
}

func (h *OAuthHandler) setPendingCookie(w http.ResponseWriter, identity oauthIdentity) error {
	token, err := generateState()
	if err != nil {
		return err
	}
	if _, err := h.db.Exec("DELETE FROM oauth_pending_flows WHERE expires_at <= ?", time.Now().UTC()); err != nil {
		return err
	}
	if _, err := h.db.Exec(
		`INSERT INTO oauth_pending_flows
		 (token, provider, provider_user_id, email, picture, suggested_name, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		token, identity.Provider, identity.ProviderUserID, identity.Email, identity.Picture, identity.SuggestedName, time.Now().Add(15*time.Minute).UTC(),
	); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oauthPendingCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   900,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func (h *OAuthHandler) readPendingCookie(r *http.Request) (pendingOAuthUser, bool) {
	cookie, err := r.Cookie(oauthPendingCookie)
	if err != nil || cookie.Value == "" {
		return pendingOAuthUser{}, false
	}
	var pending pendingOAuthUser
	err = h.db.QueryRow(
		`SELECT provider, provider_user_id, email, picture, suggested_name
		 FROM oauth_pending_flows WHERE token = ? AND expires_at > ?`,
		cookie.Value, time.Now().UTC(),
	).Scan(&pending.Provider, &pending.ProviderUserID, &pending.Email, &pending.Picture, &pending.SuggestedName)
	if err != nil {
		return pendingOAuthUser{}, false
	}
	pending.Token = cookie.Value
	return pending, true
}

func (h *OAuthHandler) startSession(w http.ResponseWriter, userID string) error {
	sessionToken, expiresAt, err := h.createSession(userID)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionToken,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

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
	if _, err := tx.Exec(
		"INSERT INTO sessions (id, user_id, session_token, expires_at) VALUES (?, ?, ?, ?)",
		utils.NewUUID(), userID, sessionToken, expiresAt.UTC(),
	); err != nil {
		return "", time.Time{}, err
	}
	return sessionToken, expiresAt, tx.Commit()
}

func oauthStateCookie(provider string) string {
	return "oauth_state_" + provider
}

func providerLabel(provider string) string {
	switch provider {
	case "github":
		return "GitHub"
	case "google":
		return "Google"
	default:
		return "OAuth"
	}
}

func clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
}

func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
