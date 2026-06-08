package middleware

import (
	"bytes"
	"net/http"
	"strings"
)

type bufferedResponseWriter struct {
	header http.Header
	body   bytes.Buffer
	status int
}

// BasePath mounts an application below an optional url path prefix
func BasePath(basePath string, secureCookies bool, next http.Handler) http.Handler {
	if basePath == "" && !secureCookies {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if basePath != "" {
			if r.URL.Path != basePath && !strings.HasPrefix(r.URL.Path, basePath+"/") {
				http.NotFound(w, r)
				return
			}
			r = r.Clone(r.Context())
			r.URL.Path = strings.TrimPrefix(r.URL.Path, basePath)
			if r.URL.Path == "" {
				r.URL.Path = "/"
			}
		}

		buffered := &bufferedResponseWriter{header: make(http.Header)}
		next.ServeHTTP(buffered, r)
		buffered.flush(w, basePath, secureCookies)
	})
}

// Header returns the response headers
func (w *bufferedResponseWriter) Header() http.Header {
	return w.header
}

// WriteHeader records the response status
func (w *bufferedResponseWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}

// Write records the response body
func (w *bufferedResponseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(data)
}

// flush writes the adapted response to the original response writer
func (w *bufferedResponseWriter) flush(destination http.ResponseWriter, basePath string, secureCookies bool) {
	for key, values := range w.header {
		for _, value := range values {
			destination.Header().Add(key, adaptHeader(key, value, basePath, secureCookies))
		}
	}

	body := w.body.Bytes()
	contentType := destination.Header().Get("Content-Type")
	if contentType == "" && len(body) > 0 {
		contentType = http.DetectContentType(body)
		destination.Header().Set("Content-Type", contentType)
	}
	if basePath != "" && strings.HasPrefix(contentType, "text/html") {
		body = rewriteHTMLPaths(body, basePath)
		destination.Header().Del("Content-Length")
	}

	status := w.status
	if status == 0 {
		status = http.StatusOK
	}
	destination.WriteHeader(status)
	_, _ = destination.Write(body)
}

// adaptHeader adapts redirects and cookies to the public application path
func adaptHeader(key, value, basePath string, secureCookies bool) string {
	switch http.CanonicalHeaderKey(key) {
	case "Location":
		if basePath != "" && strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") && value != basePath && !strings.HasPrefix(value, basePath+"/") {
			return basePath + value
		}
	case "Set-Cookie":
		if basePath != "" {
			value = strings.Replace(value, "Path=/;", "Path="+basePath+";", 1)
		}
		if secureCookies && !strings.Contains(strings.ToLower(value), "; secure") {
			value += "; Secure"
		}
	}
	return value
}

// rewriteHTMLPaths prefixes root-relative html urls
func rewriteHTMLPaths(body []byte, basePath string) []byte {
	for _, attribute := range []string{"href", "action", "src", "formaction"} {
		prefix := []byte(attribute + `="/`)
		for offset := 0; ; {
			index := bytes.Index(body[offset:], prefix)
			if index == -1 {
				break
			}
			index += offset
			pathStart := index + len(prefix) - 1
			pathValue := body[pathStart:]
			if bytes.HasPrefix(pathValue, []byte("//")) || bytes.HasPrefix(pathValue, []byte(basePath+"/")) {
				offset = pathStart + 1
				continue
			}
			body = append(body[:pathStart], append([]byte(basePath), body[pathStart:]...)...)
			offset = pathStart + len(basePath) + 1
		}
	}
	return body
}
