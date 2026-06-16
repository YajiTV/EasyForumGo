package repository

import "testing"

func TestComposeUnbracedDollarExpansion(t *testing.T) {
	got := composeUnbracedDollarExpansion("Kx7!qP2$mR9#vN4@eL6w")
	want := "Kx7!qP2#vN4@eL6w"
	if got != want {
		t.Fatalf("composeUnbracedDollarExpansion() = %q, want %q", got, want)
	}
}

func TestComposeUnbracedDollarExpansionKeepsLiteralDollar(t *testing.T) {
	got := composeUnbracedDollarExpansion("abc$-def")
	want := "abc$-def"
	if got != want {
		t.Fatalf("composeUnbracedDollarExpansion() = %q, want %q", got, want)
	}
}
