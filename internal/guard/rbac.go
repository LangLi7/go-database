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
