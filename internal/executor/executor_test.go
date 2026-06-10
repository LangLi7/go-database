package executor

import (
	"testing"
)

func TestTrimSQL_Whitespace(t *testing.T) {
	got := trimSQL("  select 1")
	if got != "SELECT 1" {
		t.Fatalf("expected 'SELECT 1', got %q", got)
	}
}

func TestTrimSQL_SingleLineComment(t *testing.T) {
	got := trimSQL("-- comment\nselect 1")
	if got != "SELECT 1" {
		t.Fatalf("expected 'SELECT 1', got %q", got)
	}
}

func TestTrimSQL_BlockComment(t *testing.T) {
	got := trimSQL("/* block comment */select 1")
	if got != "SELECT 1" {
		t.Fatalf("expected 'SELECT 1', got %q", got)
	}
}

func TestTrimSQL_MixedComments(t *testing.T) {
	got := trimSQL("-- line\n/* block */  select 1")
	if got != "SELECT 1" {
		t.Fatalf("expected 'SELECT 1', got %q", got)
	}
}

func TestTrimSQL_OnlyComment(t *testing.T) {
	got := trimSQL("-- just a comment")
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestTrimSQL_OnlyBlockComment(t *testing.T) {
	got := trimSQL("/* just a block */")
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestTrimSQL_AlreadyTrimmed(t *testing.T) {
	got := trimSQL("SELECT 1")
	if got != "SELECT 1" {
		t.Fatalf("expected 'SELECT 1', got %q", got)
	}
}
