package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"go-database/internal/api/response"
	"go-database/internal/connection"
)

// BrowseTable returns paginated data from a table
func BrowseTable(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		connID := c.Param("id")
		table := c.Param("table")

		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "50"))
		sortBy := c.DefaultQuery("sort", "")
		sortDir := c.DefaultQuery("dir", "asc")
		filter := sanitizeFilter(c.Query("filter"))

		if page < 1 {
			page = 1
		}
		if perPage < 1 || perPage > 200 {
			perPage = 50
		}

		// Build query with pagination
		offset := (page - 1) * perPage
		var query string

		// Get schema first to know column names
		schema, err := mgr.Schema(c.Request.Context(), connID)
		if err != nil || len(schema.Tables) == 0 {
			// Fallback: simple SELECT
			query = fmt.Sprintf("SELECT * FROM %s", quoteTable(table))
		} else {
			// Find the matching table
			for _, t := range schema.Tables {
				if strings.EqualFold(t.Name, table) {
					colNames := make([]string, len(t.Columns))
					for i, col := range t.Columns {
						colNames[i] = col.Name
					}
					query = fmt.Sprintf("SELECT %s FROM %s",
						strings.Join(colNames, ", "), quoteTable(table))
					break
				}
			}
			if query == "" {
				query = fmt.Sprintf("SELECT * FROM %s", quoteTable(table))
			}
		}

		// Add filter
		if filter != "" {
			query += fmt.Sprintf(" WHERE %s", filter)
		}

		// Add sorting
		if sortBy != "" {
			dir := "ASC"
			if strings.ToUpper(sortDir) == "DESC" {
				dir = "DESC"
			}
			query += fmt.Sprintf(" ORDER BY %s %s", quoteTable(sortBy), dir)
		}

		// Add LIMIT/OFFSET
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", perPage, offset)

		result, err := mgr.Query(c.Request.Context(), connID, query)
		if err != nil {
			response.Error(c, http.StatusBadGateway, "QUERY_FAILED", err.Error())
			return
		}

		// Get total count for pagination info
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", quoteTable(table))
		if filter != "" {
			countQuery += fmt.Sprintf(" WHERE %s", filter)
		}

		countResult, err := mgr.Query(c.Request.Context(), connID, countQuery)
		if err != nil {
			slog.Warn("explorer: count query failed", "error", err)
		}
		total := 0
		if countResult != nil && len(countResult.Rows) > 0 {
			switch v := countResult.Rows[0][0].(type) {
			case int64:
				total = int(v)
			case float64:
				total = int(v)
			case int:
				total = v
			}
		}

		totalPages := 0
		if perPage > 0 {
			totalPages = (total + perPage - 1) / perPage
		}

		response.Success(c, gin.H{
			"data":        result.Rows,
			"columns":     result.Columns,
			"page":        page,
			"per_page":    perPage,
			"total":       total,
			"total_pages": totalPages,
			"duration_ms": result.Duration,
		})
	}
}

// InsertRow creates a new row in a table
func InsertRow(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		connID := c.Param("id")
		table := c.Param("table")

		var data map[string]any
		if err := c.ShouldBindJSON(&data); err != nil {
			response.BadRequest(c, "invalid row data")
			return
		}

		cols := make([]string, 0, len(data))
		vals := make([]string, 0, len(data))

		for col, val := range data {
			cols = append(cols, quoteTable(col))
			vals = append(vals, quoteVal(val))
		}

		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			quoteTable(table), strings.Join(cols, ", "), strings.Join(vals, ", "))

		result, err := mgr.Execute(c.Request.Context(), connID, query)
		if err != nil {
			response.Error(c, http.StatusBadGateway, "INSERT_FAILED", err.Error())
			return
		}

		response.Created(c, result)
	}
}

// UpdateRow modifies an existing row
func UpdateRow(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		connID := c.Param("id")
		table := c.Param("table")
		pk := c.Param("pk")
		pkVal := c.Param("val")

		var data map[string]any
		if err := c.ShouldBindJSON(&data); err != nil {
			response.BadRequest(c, "invalid row data")
			return
		}

		sets := make([]string, 0, len(data))
		for col, val := range data {
			sets = append(sets, fmt.Sprintf("%s = %s", quoteTable(col), quoteVal(val)))
		}

		query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = %s",
			quoteTable(table), strings.Join(sets, ", "), quoteTable(pk), quoteVal(pkVal))

		result, err := mgr.Execute(c.Request.Context(), connID, query)
		if err != nil {
			response.Error(c, http.StatusBadGateway, "UPDATE_FAILED", err.Error())
			return
		}

		response.Success(c, result)
	}
}

// DeleteRow removes a row
func DeleteRow(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		connID := c.Param("id")
		table := c.Param("table")
		pk := c.Param("pk")
		pkVal := c.Param("val")

		query := fmt.Sprintf("DELETE FROM %s WHERE %s = %s",
			quoteTable(table), quoteTable(pk), quoteVal(pkVal))

		result, err := mgr.Execute(c.Request.Context(), connID, query)
		if err != nil {
			response.Error(c, http.StatusBadGateway, "DELETE_FAILED", err.Error())
			return
		}

		response.Success(c, result)
	}
}

// sanitizeFilter validates that a WHERE clause contains only safe SQL
func sanitizeFilter(filter string) string {
	s := strings.TrimSpace(filter)
	if s == "" {
		return ""
	}

	upper := strings.ToUpper(s)

	// Block dangerous SQL keywords
	dangerous := []string{
		"DROP ", "DELETE ", "INSERT ", "UPDATE ", "ALTER ", "CREATE ",
		"TRUNCATE ", "EXEC ", "EXECUTE ", "GRANT ", "REVOKE ",
		"INTO ", "INFILE ", "LOAD ",
		"UNION", "INFORMATION_SCHEMA",
		"SLEEP(", "BENCHMARK(", "SYS.",
	}
	for _, kw := range dangerous {
		if strings.Contains(upper, kw) {
			return ""
		}
	}

	// Block statement separators and comments
	if strings.ContainsAny(s, ";") ||
		strings.Contains(s, "--") ||
		strings.Contains(s, "/*") ||
		strings.Contains(s, "*/") ||
		strings.Contains(s, "#") {
		return ""
	}

	// Only allow printable ASCII, spaces, and common SQL punctuation
	// letters, digits, spaces, = < > ! , . ( ) ' _ % * + - / & | ~ : @ [ ]
	for _, r := range s {
		if r > 126 || r < 32 {
			return ""
		}
	}

	return s
}

func quoteTable(name string) string {
	if name == "" {
		return name
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			// contains special chars — escape and quote with backticks
			escaped := strings.ReplaceAll(name, "`", "``")
			return "`" + escaped + "`"
		}
	}
	return name // simple identifier, no quoting needed
}

func quoteVal(v any) string {
	if v == nil {
		return "NULL"
	}
	switch val := v.(type) {
	case string:
		escaped := strings.ReplaceAll(val, "'", "''")
		return fmt.Sprintf("'%s'", escaped)
	case float64:
		return fmt.Sprintf("%v", val)
	case bool:
		if val {
			return "1"
		}
		return "0"
	default:
		escaped := strings.ReplaceAll(fmt.Sprintf("%v", v), "'", "''")
		return fmt.Sprintf("'%s'", escaped)
	}
}
