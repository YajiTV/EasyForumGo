package utils

import "testing"

func TestStripMarkdownUnescapesApostrophes(t *testing.T) {
	got := StripMarkdown("C'est l'exemple d'un post")
	want := "C'est l'exemple d'un post"
	if got != want {
		t.Fatalf("StripMarkdown() = %q, want %q", got, want)
	}
}

func TestStripMarkdownRemovesTagsButKeepsText(t *testing.T) {
	got := StripMarkdown("Un **post** avec [un lien](https://example.com)")
	want := "Un post avec un lien"
	if got != want {
		t.Fatalf("StripMarkdown() = %q, want %q", got, want)
	}
}
