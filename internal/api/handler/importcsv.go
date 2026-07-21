package handler

import (
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"go-database/internal/api/response"
	"go-database/internal/connection"
)

func ImportCSV(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		connID := c.Param("id")
		tableName := c.PostForm("table")
		createTable := c.PostForm("create") == "true"

		file, _, err := c.Request.FormFile("file")
		if err != nil {
			response.BadRequest(c, "CSV file required: "+err.Error())
			return
		}
		defer file.Close()

		reader := csv.NewReader(file)
		reader.LazyQuotes = true
		reader.TrimLeadingSpace = true

		header, err := reader.Read()
		if err != nil {
			response.BadRequest(c, "failed to read CSV header: "+err.Error())
			return
		}

		if len(header) == 0 {
			response.BadRequest(c, "CSV has no columns")
			return
		}

		// Read all rows
		var rows [][]string
		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				slog.Warn("csv import: skipping row", "error", err)
				continue
			}
			rows = append(rows, record)
		}

		if len(rows) == 0 {
			response.BadRequest(c, "CSV has no data rows")
			return
		}

		// If creating a new table, infer types and create it
		if createTable || tableName == "" {
			tableName = inferTableName(tableName, c.PostForm("table"))
		}

		colTypes := inferTypes(header, rows)

		// Build CREATE TABLE if needed
		if createTable {
			createSQL := buildCreateSQL(tableName, header, colTypes)
			slog.Info("csv import: creating table", "sql", createSQL)
			if _, err := mgr.Execute(c.Request.Context(), connID, createSQL); err != nil {
				response.InternalError(c, "create table failed: "+err.Error())
				return
			}
		}

		// Insert rows in batches
		batchSize := 100
		inserted := 0
		for i := 0; i < len(rows); i += batchSize {
			end := i + batchSize
			if end > len(rows) {
				end = len(rows)
			}
			batch := rows[i:end]

			var valueRows []string
			for _, row := range batch {
				var vals []string
				for j, col := range header {
					if j < len(row) {
						vals = append(vals, quoteCSVValue(row[j], colTypes[col]))
					} else {
						vals = append(vals, "NULL")
					}
				}
				valueRows = append(valueRows, "("+strings.Join(vals, ",")+")")
			}

			quotedCols := make([]string, len(header))
			for i, col := range header {
				quotedCols[i] = `"` + col + `"`
			}

			insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES\n%s",
				`"`+tableName+`"`,
				strings.Join(quotedCols, ","),
				strings.Join(valueRows, ",\n"))

			if _, err := mgr.Execute(c.Request.Context(), connID, insertSQL); err != nil {
				slog.Error("csv import: insert batch failed", "batch", i/batchSize, "error", err)
				return
			}
			inserted += len(batch)
		}

		slog.Info("csv import complete", "table", tableName, "rows", inserted)
		response.Created(c, gin.H{
			"table":    tableName,
			"columns":  header,
			"inserted": inserted,
		})
	}
}

func inferTableName(name string, formName string) string {
	if name != "" {
		return name
	}
	return "imported_data"
}

func inferTypes(header []string, rows [][]string) map[string]string {
	types := make(map[string]string)
	for _, col := range header {
		types[col] = "TEXT"
	}

	for _, row := range rows {
		for j, col := range header {
			if j >= len(row) {
				continue
			}
			val := strings.TrimSpace(row[j])
			if val == "" || val == "NULL" || val == "null" {
				continue
			}
			current := types[col]
			if current == "TEXT" {
				if _, err := strconv.ParseInt(val, 10, 64); err == nil {
					types[col] = "INTEGER"
				} else if _, err := strconv.ParseFloat(val, 64); err == nil {
					types[col] = "REAL"
				}
			} else if current == "INTEGER" {
				if _, err := strconv.ParseInt(val, 10, 64); err != nil {
					if _, err := strconv.ParseFloat(val, 64); err == nil {
						types[col] = "REAL"
					} else {
						types[col] = "TEXT"
					}
				}
			} else if current == "REAL" {
				if _, err := strconv.ParseFloat(val, 64); err != nil {
					types[col] = "TEXT"
				}
			}
		}
	}
	return types
}

func buildCreateSQL(tableName string, columns []string, types map[string]string) string {
	var cols []string
	for _, col := range columns {
		colType := types[col]
		if colType == "" {
			colType = "TEXT"
		}
		cols = append(cols, fmt.Sprintf("  %s %s", `"`+col+`"`, colType))
	}
	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n%s\n);", `"`+tableName+`"`, strings.Join(cols, ",\n"))
}

func quoteCSVValue(val string, colType string) string {
	val = strings.TrimSpace(val)
	if val == "" || val == "NULL" || val == "null" {
		return "NULL"
	}
	if colType == "INTEGER" || colType == "REAL" {
		if _, err := strconv.ParseFloat(val, 64); err == nil {
			return val
		}
	}
	return "'" + strings.ReplaceAll(val, "'", "''") + "'"
}
