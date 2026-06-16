package utils

import (
	"bytes"
	"html"
	"html/template"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
)

var markdownParser = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithRendererOptions(
		goldmarkhtml.WithHardWraps(),
		goldmarkhtml.WithXHTML(),
	),
)

var htmlSanitizer = bluemonday.UGCPolicy()
var strictSanitizer = bluemonday.StrictPolicy()

func RenderMarkdown(content string) template.HTML {
	var buf bytes.Buffer
	if err := markdownParser.Convert([]byte(content), &buf); err != nil {
		return template.HTML(template.HTMLEscapeString(content))
	}
	return template.HTML(htmlSanitizer.SanitizeBytes(buf.Bytes()))
}

func StripMarkdown(content string) string {
	var buf bytes.Buffer
	if err := markdownParser.Convert([]byte(content), &buf); err != nil {
		return content
	}
	text := strings.TrimSpace(html.UnescapeString(strictSanitizer.Sanitize(buf.String())))
	if len([]rune(text)) > 200 {
		runes := []rune(text)
		return string(runes[:200]) + "…"
	}
	return text
}
