package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const themeCookieName = "easy_theme"

// ThemeHandler persists and serves the selected color theme
type ThemeHandler struct{}

// NewThemeHandler creates a new instance
func NewThemeHandler() *ThemeHandler {
	return &ThemeHandler{}
}

// Set stores the selected theme and returns to the previous page
func (h *ThemeHandler) Set(w http.ResponseWriter, r *http.Request) {
	theme := r.PathValue("theme")
	if !validTheme(theme) {
		http.NotFound(w, r)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     themeCookieName,
		Value:    theme,
		Path:     "/",
		MaxAge:   int((365 * 24 * time.Hour).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, themeRedirect(r), http.StatusSeeOther)
}

// CSS serves cookie-specific theme overrides
func (h *ThemeHandler) CSS(w http.ResponseWriter, r *http.Request) {
	theme := selectedTheme(r)
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store")
	fmt.Fprintf(w, `.theme-switcher [data-theme="%s"] { background: var(--surface-raised); color: var(--text); box-shadow: 0 1px 4px rgba(0, 0, 0, 0.14); }`, theme)

	switch theme {
	case "light":
		fmt.Fprint(w, lightThemeCSS)
	case "dark":
		fmt.Fprint(w, darkThemeCSS)
	}
}

// selectedTheme returns the stored theme or the system default
func selectedTheme(r *http.Request) string {
	cookie, err := r.Cookie(themeCookieName)
	if err != nil || !validTheme(cookie.Value) {
		return "system"
	}
	return cookie.Value
}

// validTheme checks whether a theme can be selected
func validTheme(theme string) bool {
	return theme == "light" || theme == "system" || theme == "dark"
}

// themeRedirect returns a safe same-site destination
func themeRedirect(r *http.Request) string {
	referer := strings.TrimSpace(r.Referer())
	parsed, err := url.Parse(referer)
	if err == nil && parsed.Host == r.Host && strings.HasPrefix(parsed.Path, "/") && !strings.HasPrefix(parsed.Path, "//") {
		return parsed.RequestURI()
	}
	return "/"
}

const lightThemeCSS = `
.theme-shell {
	--text: #242424;
	--muted: #6b6b6b;
	--soft: #f7f7f7;
	--line: #e9e9e9;
	--surface: #fff;
	--surface-raised: #fff;
	--surface-input: #fff;
	--surface-hover: #efefef;
	--accent: #1a8917;
	--accent-dark: #156912;
	--danger: #c0392b;
	color-scheme: light;
}
body { background: #fff; }
`

const darkThemeCSS = `
.theme-shell {
	--text: #ededed;
	--muted: #aaa;
	--soft: #202020;
	--line: #343434;
	--surface: #151515;
	--surface-raised: #202020;
	--surface-input: #272727;
	--surface-hover: #303030;
	--accent: #58b957;
	--accent-dark: #79cf77;
	--danger: #ff8174;
	color-scheme: dark;
}
body { background: #151515; }
`
