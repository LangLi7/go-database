package suggest

import (
	"testing"
)

func TestLevenshtein_Empty(t *testing.T) {
	if d := Levenshtein("", ""); d != 0 {
		t.Fatalf("expected 0, got %d", d)
	}
	if d := Levenshtein("abc", ""); d != 3 {
		t.Fatalf("expected 3, got %d", d)
	}
	if d := Levenshtein("", "abc"); d != 3 {
		t.Fatalf("expected 3, got %d", d)
	}
}

func TestLevenshtein_Exact(t *testing.T) {
	if d := Levenshtein("hello", "hello"); d != 0 {
		t.Fatalf("expected 0, got %d", d)
	}
}

func TestLevenshtein_OneSubstitution(t *testing.T) {
	if d := Levenshtein("cat", "car"); d != 1 {
		t.Fatalf("expected 1, got %d", d)
	}
}

func TestLevenshtein_CaseInsensitive(t *testing.T) {
	if d := Levenshtein("HELLO", "hello"); d != 0 {
		t.Fatalf("expected 0, got %d", d)
	}
}

func TestFuzzyMatcher_EmptyInput(t *testing.T) {
	fm := NewFuzzyMatcher([]string{"SELECT", "FROM"}, 2)
	if res := fm.Match("", 5); res != nil {
		t.Fatal("expected nil for empty input")
	}
}

func TestFuzzyMatcher_ExactMatch(t *testing.T) {
	fm := NewFuzzyMatcher([]string{"SELECT", "FROM", "WHERE"}, 2)
	res := fm.Match("SELECT", 5)
	if len(res) == 0 {
		t.Fatal("expected at least one match")
	}
	if res[0].Text != "SELECT" {
		t.Fatalf("expected SELECT, got %s", res[0].Text)
	}
}

func TestFuzzyMatcher_FuzzyMatch(t *testing.T) {
	fm := NewFuzzyMatcher([]string{"SELECT", "SET", "FROM"}, 2)
	res := fm.Match("SELEC", 5) // one char off
	if len(res) == 0 {
		t.Fatal("expected at least one fuzzy match")
	}
}

func TestFuzzyMatcher_NoMatch(t *testing.T) {
	fm := NewFuzzyMatcher([]string{"SELECT", "FROM"}, 2)
	res := fm.Match("ZZZZZ", 5)
	if len(res) != 0 {
		t.Fatalf("expected no matches, got %d", len(res))
	}
}

func TestFuzzyMatcher_Limit(t *testing.T) {
	fm := NewFuzzyMatcher([]string{"SELECT", "SET", "FROM", "WHERE"}, 2)
	res := fm.Match("SE", 2)
	if len(res) > 2 {
		t.Fatalf("expected at most 2 results, got %d", len(res))
	}
}

func TestMin3(t *testing.T) {
	if v := min3(1, 2, 3); v != 1 {
		t.Fatalf("expected 1, got %d", v)
	}
	if v := min3(3, 1, 2); v != 1 {
		t.Fatalf("expected 1, got %d", v)
	}
	if v := min3(3, 2, 1); v != 1 {
		t.Fatalf("expected 1, got %d", v)
	}
}

func TestSortMatches(t *testing.T) {
	ms := []match{{"b", 3}, {"a", 1}, {"c", 2}}
	sortMatches(ms)
	if ms[0].word != "a" || ms[1].word != "c" || ms[2].word != "b" {
		t.Fatal("matches not sorted by distance")
	}
}
