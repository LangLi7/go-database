package suggest

import (
	"math"
	"strings"
)

type FuzzyMatcher struct {
	dict    []string
	maxDist int
}

func NewFuzzyMatcher(dict []string, maxDistance int) *FuzzyMatcher {
	return &FuzzyMatcher{dict: dict, maxDist: maxDistance}
}

func Levenshtein(s, t string) int {
	s = strings.ToLower(s)
	t = strings.ToLower(t)
	if len(s) == 0 {
		return len(t)
	}
	if len(t) == 0 {
		return len(s)
	}

	d := make([][]int, len(s)+1)
	for i := range d {
		d[i] = make([]int, len(t)+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}

	for i := 1; i <= len(s); i++ {
		for j := 1; j <= len(t); j++ {
			cost := 1
			if s[i-1] == t[j-1] {
				cost = 0
			}
			d[i][j] = min3(
				d[i-1][j]+1,
				d[i][j-1]+1,
				d[i-1][j-1]+cost,
			)
		}
	}
	return d[len(s)][len(t)]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func (f *FuzzyMatcher) Match(input string, limit int) []Suggestion {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	var matches []match
	for _, word := range f.dict {
		dist := Levenshtein(input, word)
		maxLen := math.Max(float64(len(input)), float64(len(word)))
		if float64(dist) <= float64(maxLen)*0.4 {
			matches = append(matches, match{word, dist})
		}
	}

	sortMatches(matches)

	var results []Suggestion
	for i := 0; i < len(matches) && i < limit; i++ {
		m := matches[i]
		confidence := 1.0 - float64(m.dist)/math.Max(float64(len(input)), 1)
		if confidence < 0 {
			confidence = 0
		}
		results = append(results, Suggestion{
			Text:        m.word,
			Description: "fuzzy match",
			Type:        SuggKeyword,
			Confidence:  confidence,
			RiskLevel:   RiskLow,
		})
	}
	return results
}

type match struct {
	word string
	dist int
}

func sortMatches(matches []match) {
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].dist < matches[i].dist {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}
}
