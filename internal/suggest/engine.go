package suggest

import (
	"sort"
	"strings"

	"go-database/internal/plugin"
)

type Engine struct {
	keywords *Trie
	risk     *RiskEvaluator
}

func NewEngine() *Engine {
	return &Engine{
		keywords: NewKeywordTrie(),
		risk:     NewRiskEvaluator(),
	}
}

func (e *Engine) GetSuggestions(ctx Context, limit int) []Suggestion {
	var results []Suggestion
	seen := make(map[string]bool)

	input := strings.TrimSpace(ctx.Input)
	if input == "" {
		return nil
	}

	lastWord := extractLastWord(input)

	// 1. Prefix match on SQL keywords
	if kw := e.keywords.Search(lastWord, limit); kw != nil {
		for _, word := range kw {
			if !seen[word] {
				risk, desc := e.risk.Classify(word)
				results = append(results, Suggestion{
					Text:        word,
					Description: desc,
					Type:        SuggKeyword,
					Confidence:  0.9,
					RiskLevel:   risk,
				})
				seen[word] = true
			}
		}
	}

	// 2. Schema-aware suggestions (table names, column names)
	if ctx.Schema != nil {
		// Suggest table names
		for _, tbl := range ctx.Schema.Tables {
			if strings.HasPrefix(strings.ToLower(tbl.Name), strings.ToLower(lastWord)) && !seen[tbl.Name] {
				results = append(results, Suggestion{
					Text:        tbl.Name,
					Description: "table (" + tblName(tbl.Columns) + ")",
					Type:        SuggTable,
					Confidence:  0.85,
					RiskLevel:   RiskLow,
				})
				seen[tbl.Name] = true
			}
			// Suggest column names
			for _, col := range tbl.Columns {
				if strings.HasPrefix(strings.ToLower(col.Name), strings.ToLower(lastWord)) && !seen[col.Name] {
					results = append(results, Suggestion{
						Text:        col.Name,
						Description: "column in " + tbl.Name + " (" + col.Type + ")",
						Type:        SuggColumn,
						Confidence:  0.8,
						RiskLevel:   RiskLow,
					})
					seen[col.Name] = true
				}
			}
		}
	}

	// 3. Fuzzy match if prefix match yielded too few results
	if len(results) < 3 {
		dict := buildDictionary(ctx)
		fm := NewFuzzyMatcher(dict, 3)
		fuzzy := fm.Match(lastWord, limit)
		for _, f := range fuzzy {
			if !seen[f.Text] {
				results = append(results, f)
				seen[f.Text] = true
			}
		}
	}

	// 4. Statement templates (context-aware)
	if templates := getStatementTemplates(ctx); templates != nil {
		for _, tpl := range templates {
			if !seen[tpl.Text] {
				results = append(results, tpl)
				seen[tpl.Text] = true
			}
		}
	}

	// 5. Sort by confidence descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Confidence > results[j].Confidence
	})

	if len(results) > limit {
		results = results[:limit]
	}

	return results
}

func (e *Engine) ClassifyStatement(sql string) (RiskLevel, string) {
	return e.risk.Classify(sql)
}

func extractLastWord(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}
	parts := strings.Fields(input)
	return parts[len(parts)-1]
}

func tblName(cols []plugin.ColumnInfo) string {
	if len(cols) == 0 {
		return ""
	}
	n := cols[0].Name
	if len(cols) > 3 {
		return n + ", ..."
	}
	var names []string
	for _, c := range cols {
		names = append(names, c.Name)
	}
	return strings.Join(names, ", ")
}

func buildDictionary(ctx Context) []string {
	dict := make([]string, 0)
	dict = append(dict, sqlKeywords...)
	if ctx.Schema != nil {
		for _, tbl := range ctx.Schema.Tables {
			dict = append(dict, tbl.Name)
			for _, col := range tbl.Columns {
				dict = append(dict, col.Name)
			}
		}
	}
	return dict
}

func getStatementTemplates(ctx Context) []Suggestion {
	input := strings.TrimSpace(strings.ToUpper(ctx.Input))

	if looksLikeNaturalLanguage(input) {
		role := strings.ToLower(ctx.Role)
		var stmts []Suggestion

		// If the user can't write, don't suggest write operations
		canWrite := role == "admin" || role == "developer"

		if containsAny(input, []string{"show", "all", "list", "alle", "zeige"}) {
			tbl := ctx.CurrentTable
			if tbl == "" {
				tbl = "users"
			}
			stmts = append(stmts, Suggestion{
				Text:        "SELECT * FROM " + tbl + " LIMIT 100;",
				Description: "Show all data from " + tbl,
				Type:        SuggIntent,
				Confidence:  0.7,
				RiskLevel:   RiskLow,
			})
		}

		if containsAny(input, []string{"count", "anzahl", "wie viele"}) {
			tbl := ctx.CurrentTable
			if tbl == "" {
				tbl = "users"
			}
			stmts = append(stmts, Suggestion{
				Text:        "SELECT COUNT(*) FROM " + tbl + ";",
				Description: "Count rows in " + tbl,
				Type:        SuggIntent,
				Confidence:  0.7,
				RiskLevel:   RiskLow,
			})
		}

		if containsAny(input, []string{"schema", "structure", "struktur", "spalten"}) {
			tbl := sanitizeIdent(ctx.CurrentTable)
			if tbl == "" {
				stmts = append(stmts, Suggestion{
					Text:        "SELECT table_name, column_name, data_type FROM information_schema.columns WHERE table_schema = 'public' ORDER BY table_name, ordinal_position;",
					Description: "Show database schema",
					Type:        SuggIntent,
					Confidence:  0.65,
					RiskLevel:   RiskLow,
				})
			} else {
				stmts = append(stmts, Suggestion{
					Text:        "SELECT column_name, data_type, is_nullable, column_default FROM information_schema.columns WHERE table_name = '" + tbl + "' ORDER BY ordinal_position;",
					Description: "Show columns of " + tbl,
					Type:        SuggIntent,
					Confidence:  0.7,
					RiskLevel:   RiskLow,
				})
			}
		}

		if canWrite && containsAny(input, []string{"delete", "remove", "löschen", "entfernen"}) {
			tbl := sanitizeIdent(ctx.CurrentTable)
			if tbl == "" {
				tbl = "users"
			}
			stmts = append(stmts, Suggestion{
				Text:        "DELETE FROM " + tbl + " WHERE id = ?;",
				Description: "Delete a specific row from " + tbl + " (HIGH RISK)",
				Type:        SuggIntent,
				Confidence:  0.6,
				RiskLevel:   RiskHigh,
			})
		}

		return stmts
	}

	return nil
}

func looksLikeNaturalLanguage(input string) bool {
	if len(input) < 3 {
		return false
	}
	// Check for common non-SQL patterns
	hasSQLPrefix := false
	for _, kw := range []string{"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER", "SHOW", "WITH", "EXPLAIN"} {
		if strings.HasPrefix(input, kw) {
			hasSQLPrefix = true
			break
		}
	}
	if hasSQLPrefix {
		return false
	}
	// Check if input contains spaces and looks like natural language
	words := strings.Fields(input)
	if len(words) < 2 {
		return false
	}
	// If it contains German or English natural language patterns
	naturalWords := []string{"alle", "zeige", "wie", "ist", "der", "die", "das", "show", "all", "list", "find", "where", "what", "which", "give", "delete", "remove", "create", "add", "update", "change"}
	for _, word := range words {
		lower := strings.ToLower(word)
		for _, nw := range naturalWords {
			if lower == nw {
				return true
			}
		}
	}
	return false
}

func containsAny(s string, substrs []string) bool {
	s = strings.ToLower(s)
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// sanitizeIdent allows only alphanumeric and underscore characters
func sanitizeIdent(name string) string {
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_') {
			return ""
		}
	}
	return name
}
