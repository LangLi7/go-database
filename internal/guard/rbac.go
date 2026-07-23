package guard

import (
	"strings"

	"go-database/internal/auth"
	"go-database/internal/suggest"
)

type Guard struct{}

func New() *Guard {
	return &Guard{}
}

var commandPermissions = map[suggest.CommandType]string{
	suggest.CmdSelect:   auth.PermConnectionsQuery,
	suggest.CmdInsert:   auth.PermConnectionsExec,
	suggest.CmdUpdate:   auth.PermConnectionsExec,
	suggest.CmdDelete:   auth.PermConnectionsExec,
	suggest.CmdCreate:   auth.PermConnectionsExec,
	suggest.CmdDrop:     auth.PermConnectionsExec,
	suggest.CmdAlter:    auth.PermConnectionsExec,
	suggest.CmdTruncate: auth.PermConnectionsExec,
}

func (g *Guard) CheckCommand(sql string, permissions []string) (suggest.CommandType, bool) {
	cmd := detectCommand(sql)
	perm, ok := commandPermissions[cmd]
	if !ok {
		return cmd, false
	}
	for _, p := range permissions {
		if p == perm {
			return cmd, true
		}
	}
	return cmd, false
}

func (g *Guard) FilterSuggestions(sugg []suggest.Suggestion, permissions []string) []suggest.Suggestion {
	var filtered []suggest.Suggestion
	for _, s := range sugg {
		cmd := detectCommand(s.Text)
		perm, ok := commandPermissions[cmd]
		if !ok {
			filtered = append(filtered, s)
			continue
		}
		for _, p := range permissions {
			if p == perm {
				filtered = append(filtered, s)
				break
			}
		}
	}
	return filtered
}

func detectCommand(sql string) suggest.CommandType {
	sql = trimSQL(sql)
	// ponytail: split on ';' and classify the HIGHEST-risk statement. A
	// "SELECT 1; DROP TABLE users;" must be caught as DROP, not SELECT.
	// Real SQL grammars (vitess/sqlparser) can be added later if nested
	// literals/comments need full parsing.
	highest := suggest.CmdUnknown
	var highestRisk int
	for _, stmt := range splitStatements(sql) {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		c := prefixCommand(stmt)
		if c == suggest.CmdUnknown {
			continue
		}
		if r := riskLevel(c); r > highestRisk {
			highestRisk = r
			highest = c
		}
	}
	return highest
}

// riskLevel ranks command destructiveness (higher = more dangerous). Used to
// pick the worst statement in a multi-statement query.
func riskLevel(c suggest.CommandType) int {
	switch c {
	case suggest.CmdDrop, suggest.CmdTruncate:
		return 5
	case suggest.CmdDelete, suggest.CmdAlter:
		return 4
	case suggest.CmdCreate, suggest.CmdUpdate:
		return 3
	case suggest.CmdInsert:
		return 2
	case suggest.CmdSelect:
		return 1
	default:
		return 0
	}
}

// splitStatements splits SQL on ';', ignoring ';' inside string literals.
func splitStatements(sql string) []string {
	var out []string
	var cur strings.Builder
	inStr := false
	quote := byte(0)
	for i := 0; i < len(sql); i++ {
		ch := sql[i]
		if inStr {
			cur.WriteByte(ch)
			if ch == quote {
				inStr = false
			}
			continue
		}
		switch ch {
		case '\'', '"', '`':
			inStr = true
			quote = ch
			cur.WriteByte(ch)
		case ';':
			out = append(out, cur.String())
			cur.Reset()
		default:
			cur.WriteByte(ch)
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

// prefixCommand maps the first keyword of a single statement to a command type.
func prefixCommand(sql string) suggest.CommandType {
	switch {
	case strings.HasPrefix(sql, "SELECT"):
		return suggest.CmdSelect
	case strings.HasPrefix(sql, "INSERT"):
		return suggest.CmdInsert
	case strings.HasPrefix(sql, "UPDATE"):
		return suggest.CmdUpdate
	case strings.HasPrefix(sql, "DELETE"):
		return suggest.CmdDelete
	case strings.HasPrefix(sql, "CREATE"):
		return suggest.CmdCreate
	case strings.HasPrefix(sql, "DROP"):
		return suggest.CmdDrop
	case strings.HasPrefix(sql, "ALTER"):
		return suggest.CmdAlter
	case strings.HasPrefix(sql, "TRUNCATE"):
		return suggest.CmdTruncate
	default:
		return suggest.CmdUnknown
	}
}

// trimSQL strips leading whitespace and SQL comments, returning uppercase
func trimSQL(sql string) string {
	s := strings.TrimSpace(sql)
	for {
		switch {
		case strings.HasPrefix(s, "--"):
			if idx := strings.Index(s, "\n"); idx >= 0 {
				s = strings.TrimSpace(s[idx+1:])
			} else {
				return ""
			}
		case strings.HasPrefix(s, "/*"):
			if idx := strings.Index(s, "*/"); idx >= 0 {
				s = strings.TrimSpace(s[idx+2:])
			} else {
				return ""
			}
		default:
			return strings.ToUpper(s)
		}
	}
}
