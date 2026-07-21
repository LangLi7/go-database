package transfer

import (
	"fmt"
	"strings"

	"go-database/internal/plugin"
)

// StandardType represents a normalized database type
type StandardType int

const (
	TypeSerial StandardType = iota
	TypeBigSerial
	TypeInt
	TypeBigInt
	TypeSmallInt
	TypeTinyInt
	TypeVarChar
	TypeChar
	TypeText
	TypeBool
	TypeTimestamp
	TypeTimestamptz
	TypeDate
	TypeTime
	TypeFloat
	TypeDouble
	TypeDecimal
	TypeJSON
	TypeBinary
	TypeUUID
	TypeInterval
	TypeBlob
	TypeOther
)

var standardNames = map[StandardType]string{
	TypeSerial:      "SERIAL",
	TypeBigSerial:   "BIGSERIAL",
	TypeInt:         "INTEGER",
	TypeBigInt:      "BIGINT",
	TypeSmallInt:    "SMALLINT",
	TypeTinyInt:     "TINYINT",
	TypeVarChar:     "VARCHAR",
	TypeChar:        "CHAR",
	TypeText:        "TEXT",
	TypeBool:        "BOOLEAN",
	TypeTimestamp:   "TIMESTAMP",
	TypeTimestamptz: "TIMESTAMPTZ",
	TypeDate:        "DATE",
	TypeTime:        "TIME",
	TypeFloat:       "REAL",
	TypeDouble:      "DOUBLE PRECISION",
	TypeDecimal:     "NUMERIC",
	TypeJSON:        "JSON",
	TypeBinary:      "BYTEA",
	TypeUUID:        "UUID",
	TypeInterval:    "INTERVAL",
	TypeBlob:        "BLOB",
	TypeOther:       "TEXT",
}

// classifyType normalizes a raw type string to a StandardType
func classifyType(raw string) StandardType {
	base := strings.ToUpper(strings.TrimSpace(raw))
	if idx := strings.Index(base, "("); idx > 0 {
		base = base[:idx]
	}
	switch base {
	case "SERIAL", "SERIAL4":
		return TypeSerial
	case "BIGSERIAL", "SERIAL8":
		return TypeBigSerial
	case "SMALLSERIAL", "SERIAL2":
		return TypeSmallInt
	case "INT", "INT4", "INTEGER", "INT32":
		return TypeInt
	case "BIGINT", "INT8", "INT64", "LONG":
		return TypeBigInt
	case "SMALLINT", "INT2", "SHORT", "TINY", "YEAR":
		return TypeSmallInt
	case "TINYINT":
		return TypeTinyInt
	case "VARCHAR", "VARCHAR2", "NVARCHAR", "CHARACTER VARYING":
		return TypeVarChar
	case "CHAR", "BPCHAR", "NCHAR":
		return TypeChar
	case "TEXT", "LONGTEXT", "MEDIUMTEXT", "CLOB", "STRING":
		return TypeText
	case "BOOLEAN", "BOOL", "BIT":
		return TypeBool
	case "TIMESTAMP", "DATETIME", "TIMESTAMP WITHOUT TIME ZONE":
		return TypeTimestamp
	case "TIMESTAMPTZ", "TIMESTAMP WITH TIME ZONE":
		return TypeTimestamptz
	case "DATE":
		return TypeDate
	case "TIME", "TIME WITHOUT TIME ZONE":
		return TypeTime
	case "FLOAT", "FLOAT4", "FLOAT8", "REAL", "SINGLE":
		return TypeFloat
	case "DOUBLE", "DOUBLE PRECISION":
		return TypeDouble
	case "DECIMAL", "NUMERIC", "NUMBER", "MONEY":
		return TypeDecimal
	case "JSON", "JSONB":
		return TypeJSON
	case "BYTEA", "BINARY", "VARBINARY", "IMAGE":
		return TypeBinary
	case "UUID":
		return TypeUUID
	case "INTERVAL":
		return TypeInterval
	case "BLOB", "LONGBLOB", "MEDIUMBLOB", "TINYBLOB", "RAW":
		return TypeBlob
	default:
		return TypeOther
	}
}

// mapTypeToTarget converts a source type string to the target DB type
func mapTypeToTarget(raw string, target plugin.DBType) string {
	base := strings.ToUpper(strings.TrimSpace(raw))
	hasSize := strings.Contains(base, "(")

	// Preserve size information for VARCHAR/CHAR
	var sizePart string
	if hasSize {
		start := strings.Index(raw, "(")
		end := strings.Index(raw, ")")
		if start > 0 && end > start {
			sizePart = raw[start : end+1]
		}
	}

	// Remove size for classification
	classRaw := raw
	if hasSize {
		if idx := strings.Index(classRaw, "("); idx > 0 {
			classRaw = classRaw[:idx]
		}
	}
	class := classifyType(classRaw)

	switch target {
	case plugin.TypePostgres:
		return mapToPostgres(class, sizePart)
	case plugin.TypeMySQL, plugin.TypeMariaDB:
		return mapToMySQL(class, sizePart, target)
	case plugin.TypeSQLite:
		return mapToSQLite(class)
	case plugin.TypeMongoDB:
		return mapToMongo(class)
	case plugin.TypeRedis:
		return "STRING"
	default:
		return "TEXT"
	}
}

func mapToPostgres(std StandardType, size string) string {
	switch std {
	case TypeSerial:
		return "SERIAL"
	case TypeBigSerial:
		return "BIGSERIAL"
	case TypeInt:
		return "INTEGER"
	case TypeBigInt:
		return "BIGINT"
	case TypeSmallInt:
		return "SMALLINT"
	case TypeTinyInt:
		return "SMALLINT"
	case TypeVarChar:
		if size != "" {
			return "VARCHAR" + size
		}
		return "VARCHAR(255)"
	case TypeChar:
		if size != "" {
			return "CHAR" + size
		}
		return "CHAR(1)"
	case TypeText:
		return "TEXT"
	case TypeBool:
		return "BOOLEAN"
	case TypeTimestamp:
		return "TIMESTAMP"
	case TypeTimestamptz:
		return "TIMESTAMPTZ"
	case TypeDate:
		return "DATE"
	case TypeTime:
		return "TIME"
	case TypeFloat:
		return "REAL"
	case TypeDouble:
		return "DOUBLE PRECISION"
	case TypeDecimal:
		return "NUMERIC"
	case TypeJSON:
		return "JSONB"
	case TypeBinary:
		return "BYTEA"
	case TypeUUID:
		return "UUID"
	case TypeInterval:
		return "INTERVAL"
	case TypeBlob:
		return "BYTEA"
	default:
		return "TEXT"
	}
}

func mapToMySQL(std StandardType, size string, target plugin.DBType) string {
	ai := "AUTO_INCREMENT"
	// MySQL SERIAL is an alias for BIGINT UNSIGNED NOT NULL AUTO_INCREMENT
	switch std {
	case TypeSerial, TypeBigSerial:
		return "BIGINT " + ai
	case TypeInt:
		return "INT"
	case TypeBigInt:
		return "BIGINT"
	case TypeSmallInt:
		return "SMALLINT"
	case TypeTinyInt:
		return "TINYINT"
	case TypeVarChar:
		if size != "" {
			return "VARCHAR" + size
		}
		return "VARCHAR(255)"
	case TypeChar:
		if size != "" {
			return "CHAR" + size
		}
		return "CHAR(1)"
	case TypeText:
		return "TEXT"
	case TypeBool:
		return "TINYINT(1)"
	case TypeTimestamp, TypeTimestamptz:
		return "DATETIME(6)"
	case TypeDate:
		return "DATE"
	case TypeTime:
		return "TIME"
	case TypeFloat:
		return "FLOAT"
	case TypeDouble:
		return "DOUBLE"
	case TypeDecimal:
		return "DECIMAL(38,6)"
	case TypeJSON:
		return "JSON"
	case TypeBinary:
		return "BLOB"
	case TypeUUID:
		return "VARCHAR(36)"
	case TypeBlob:
		return "LONGBLOB"
	default:
		return "TEXT"
	}
}

func mapToSQLite(std StandardType) string {
	switch std {
	case TypeSerial, TypeBigSerial, TypeInt, TypeBigInt, TypeSmallInt, TypeTinyInt:
		return "INTEGER"
	case TypeVarChar, TypeChar, TypeText:
		return "TEXT"
	case TypeBool:
		return "INTEGER"
	case TypeTimestamp, TypeTimestamptz, TypeDate, TypeTime:
		return "TEXT"
	case TypeFloat, TypeDouble, TypeDecimal:
		return "REAL"
	case TypeJSON, TypeBinary, TypeUUID, TypeBlob:
		return "TEXT"
	default:
		return "TEXT"
	}
}

func mapToMongo(std StandardType) string {
	switch std {
	case TypeSerial, TypeBigSerial, TypeInt, TypeBigInt, TypeSmallInt, TypeTinyInt:
		return "NumberLong"
	case TypeFloat, TypeDouble, TypeDecimal:
		return "NumberDouble"
	case TypeBool:
		return "Boolean"
	case TypeTimestamp, TypeTimestamptz, TypeDate, TypeTime:
		return "Date"
	case TypeJSON:
		return "Object"
	default:
		return "String"
	}
}

// ValueMapper converts a value from source to target representation
func ValueMapper(value any, sourceType, targetType plugin.DBType) any {
	if value == nil {
		return nil
	}

	switch targetType {
	case plugin.TypePostgres, plugin.TypeMySQL, plugin.TypeMariaDB, plugin.TypeSQLite:
		// SQL DBs handle most values natively
		return value
	case plugin.TypeMongoDB:
		// MongoDB needs special conversions
		return value
	case plugin.TypeRedis:
		// Redis stores everything as strings
		return fmt.Sprintf("%v", value)
	}
	return value
}
