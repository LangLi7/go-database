package suggest

import (
	"strings"
)

type RiskEvaluator struct{}

func NewRiskEvaluator() *RiskEvaluator {
	return &RiskEvaluator{}
}

func (r *RiskEvaluator) Classify(sql string) (RiskLevel, string) {
	sql = strings.TrimSpace(strings.ToUpper(sql))

	switch {
	case strings.HasPrefix(sql, "SELECT"):
		return r.evaluateSelect(sql)
	case strings.HasPrefix(sql, "SHOW") || strings.HasPrefix(sql, "DESCRIBE") || strings.HasPrefix(sql, "EXPLAIN"):
		return RiskLow, "Read-only query"
	case strings.HasPrefix(sql, "INSERT"):
		return RiskMedium, "Inserts data into a table"
	case strings.HasPrefix(sql, "UPDATE"):
		return r.evaluateUpdate(sql)
	case strings.HasPrefix(sql, "DELETE"):
		return r.evaluateDelete(sql)
	case strings.HasPrefix(sql, "TRUNCATE"):
		return RiskHigh, "Destructive: removes all rows from a table"
	case strings.HasPrefix(sql, "DROP"):
		return RiskHigh, "Destructive: drops database object permanently"
	case strings.HasPrefix(sql, "CREATE"):
		if strings.Contains(sql, "TABLE") || strings.Contains(sql, "DATABASE") || strings.Contains(sql, "INDEX") {
			return RiskMedium, "Creates a new database object"
		}
		return RiskMedium, "CREATE operation"
	case strings.HasPrefix(sql, "ALTER"):
		return RiskHigh, "Modifies database schema structure"
	case strings.HasPrefix(sql, "GRANT") || strings.HasPrefix(sql, "REVOKE"):
		return RiskHigh, "Modifies permissions"
	default:
		return RiskMedium, "Unknown operation"
	}
}

func (r *RiskEvaluator) evaluateSelect(sql string) (RiskLevel, string) {
	if strings.Contains(sql, "INTO") {
		return RiskMedium, "SELECT INTO creates a new table"
	}
	return RiskLow, "Read-only query"
}

func (r *RiskEvaluator) evaluateUpdate(sql string) (RiskLevel, string) {
	if strings.Contains(sql, "WHERE") {
		return RiskMedium, "Updates data with WHERE clause"
	}
	return RiskHigh, "UPDATE without WHERE clause - affects ALL rows"
}

func (r *RiskEvaluator) evaluateDelete(sql string) (RiskLevel, string) {
	if strings.Contains(sql, "WHERE") {
		// Check if WHERE is specific enough (has = or IN with a literal)
		if hasSpecificCondition(sql) {
			return RiskMedium, "Deletes specific rows"
		}
		return RiskHigh, "DELETE with potentially broad WHERE clause"
	}
	return RiskHigh, "DELETE without WHERE clause - removes ALL rows!"
}

func hasSpecificCondition(sql string) bool {
	// Simple heuristic: check for = 'value' or IN ('val1', ...) patterns
	afterWhere := extractWhereClause(sql)
	return strings.Contains(afterWhere, "=") || strings.Contains(afterWhere, "IN (")
}

func extractWhereClause(sql string) string {
	upper := strings.ToUpper(sql)
	idx := strings.Index(upper, "WHERE")
	if idx == -1 {
		return ""
	}
	return sql[idx+5:]
}

func (r *RiskEvaluator) Sanitize(sql string) (string, error) {
	upper := strings.TrimSpace(strings.ToUpper(sql))

	if strings.HasPrefix(upper, "DROP") || strings.HasPrefix(upper, "TRUNCATE") {
		return sql, nil
	}

	if strings.HasPrefix(upper, "DELETE") {
		if !strings.Contains(upper, "WHERE") {
			return sql, nil
		}
	}

	return sql, nil
}
