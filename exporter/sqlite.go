package exporter

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

// SQLiteType represents SQLite data types
type SQLiteType int

const (
	SQLiteNone SQLiteType = iota
	SQLiteInteger
	SQLiteReal
	SQLiteText
	SQLiteBlob
	SQLiteDateTime
	SQLiteBoolean // Will be stored as INTEGER internally
)

// SQLiteTypeInfo contains metadata about SQLite types
type SQLiteTypeInfo struct {
	Name         string
	IsNumeric    bool
	AllowDefault bool
	DefaultValue string
	MaxSize      int64 // -1 for unlimited
}

// Type metadata
var sqliteTypeInfoMap = map[SQLiteType]SQLiteTypeInfo{
	SQLiteInteger: {
		Name:         "INTEGER",
		IsNumeric:    true,
		AllowDefault: true,
		DefaultValue: "0",
		MaxSize:      8, // 8 bytes for int64
	},
	SQLiteReal: {
		Name:         "REAL",
		IsNumeric:    true,
		AllowDefault: true,
		DefaultValue: "0.0",
		MaxSize:      8, // 8 bytes for double
	},
	SQLiteText: {
		Name:         "TEXT",
		IsNumeric:    false,
		AllowDefault: true,
		DefaultValue: "''",
		MaxSize:      -1, // Unlimited
	},
	SQLiteBlob: {
		Name:         "BLOB",
		IsNumeric:    false,
		AllowDefault: false,
		MaxSize:      -1, // Unlimited
	},
	SQLiteDateTime: {
		Name:         "DATETIME",
		IsNumeric:    false,
		AllowDefault: true,
		DefaultValue: "CURRENT_TIMESTAMP",
		MaxSize:      -1,
	},
	SQLiteBoolean: {
		Name:         "INTEGER",
		IsNumeric:    true,
		AllowDefault: true,
		DefaultValue: "0",
		MaxSize:      1, // 1 byte
	},
}

// SQLiteTypeDefinition represents a complete type definition including constraints
type SQLiteTypeDefinition struct {
	Type        SQLiteType
	Size        int64  // Optional size constraint
	DefaultVal  string // Optional default value
	AllowNull   bool
	IsPrimary   bool
	IsUnique    bool
	Description string
}

func (st SQLiteType) String() string {
	if info, ok := sqliteTypeInfoMap[st]; ok {
		return info.Name
	}
	return "TEXT" // Default to TEXT for unknown types
}

// GetSQLiteType returns the corresponding SQLiteType for a ColumnType
func GetSQLiteType(colType ColumnType) SQLiteType {
	// Handle special types first
	if colType.Type == reflect.TypeOf(time.Time{}) {
		return SQLiteDateTime
	}

	// Handle array types
	if colType.IsArray {
		return SQLiteText // Arrays are stored as JSON
	}

	// Handle basic types
	switch colType.Type.Kind() {
	case reflect.Bool:
		return SQLiteBoolean
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		return SQLiteInteger
	case reflect.Float32, reflect.Float64:
		return SQLiteReal
	case reflect.String:
		return SQLiteText
	case reflect.Slice:
		if colType.Type.Elem().Kind() == reflect.Uint8 {
			return SQLiteBlob
		}
		return SQLiteText // Other slices as JSON
	default:
		return SQLiteText
	}
}

// GetSQLiteTypeFromColumnType converts ColumnType to SQLiteType
func GetSQLiteTypeFromColumnType(colType ColumnType) SQLiteTypeDefinition {
	if colType.IsArray {
		// Arrays are stored as JSON text
		return SQLiteTypeDefinition{
			Type:        SQLiteText,
			Description: "JSON array",
			AllowNull:   true,
		}
	}

	typeDef := SQLiteTypeDefinition{
		AllowNull: true, // Default to nullable
	}

	// Handle special types first
	if colType.Type == reflect.TypeOf(time.Time{}) {
		typeDef.Type = SQLiteDateTime
		return typeDef
	}

	// Handle basic types
	switch colType.Type.Kind() {
	case reflect.Bool:
		typeDef.Type = SQLiteBoolean

	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		typeDef.Type = SQLiteInteger

	case reflect.Float32, reflect.Float64:
		typeDef.Type = SQLiteReal

	case reflect.String:
		typeDef.Type = SQLiteText

	case reflect.Slice:
		if colType.Type.Elem().Kind() == reflect.Uint8 {
			typeDef.Type = SQLiteBlob
		} else {
			typeDef.Type = SQLiteText
			typeDef.Description = "JSON array"
		}

	default:
		typeDef.Type = SQLiteText
	}

	return typeDef
}

// BuildColumnDefinition generates SQLite column definition string
func (std SQLiteTypeDefinition) BuildColumnDefinition(colName string) string {
	parts := []string{colName, std.Type.String()}

	if std.Size > 0 && sqliteTypeInfoMap[std.Type].MaxSize > 0 {
		parts = append(parts, fmt.Sprintf("(%d)", std.Size))
	}

	if !std.AllowNull {
		parts = append(parts, "NOT NULL")
	}

	if std.IsPrimary {
		parts = append(parts, "PRIMARY KEY")
	}

	if std.IsUnique {
		parts = append(parts, "UNIQUE")
	}

	if std.DefaultVal != "" && sqliteTypeInfoMap[std.Type].AllowDefault {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", std.DefaultVal))
	}

	return strings.Join(parts, " ")
}

// ValidationError represents a type validation error
type ValidationError struct {
	Type    SQLiteType
	Value   interface{}
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("SQLite type validation error for %v: %s (value: %v)",
		e.Type, e.Message, e.Value)
}

// ValidateValue checks if a value is valid for the SQLite type
func (st SQLiteType) ValidateValue(value interface{}) error {
	_, ok := sqliteTypeInfoMap[st]
	if !ok {
		return ValidationError{Type: st, Value: value, Message: "unknown type"}
	}

	if value == nil {
		return nil // NULL values are handled separately by AllowNull
	}

	switch st {
	case SQLiteInteger:
		switch v := value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32:
			return nil
		default:
			return ValidationError{Type: st, Value: v, Message: "not an integer value"}
		}

	case SQLiteReal:
		switch v := value.(type) {
		case float32, float64:
			return nil
		default:
			return ValidationError{Type: st, Value: v, Message: "not a floating-point value"}
		}

	case SQLiteBoolean:
		switch value.(type) {
		case bool:
			return nil
		default:
			return ValidationError{Type: st, Value: value, Message: "not a boolean value"}
		}
	}

	return nil // Text and BLOB types accept any value
}

// SQLite 예약어 목록
var sqliteKeywords = map[string]bool{
	"abort": true, "action": true, "add": true, "after": true, "all": true,
	"alter": true, "analyze": true, "and": true, "as": true, "asc": true,
	"attach": true, "autoincrement": true, "before": true, "begin": true,
	"between": true, "by": true, "cascade": true, "case": true, "cast": true,
	"check": true, "collate": true, "column": true, "commit": true, "conflict": true,
	"constraint": true, "create": true, "cross": true, "current": true,
	"current_date": true, "current_time": true, "current_timestamp": true,
	"database": true, "default": true, "deferrable": true, "deferred": true,
	"delete": true, "desc": true, "detach": true, "distinct": true, "drop": true,
	"each": true, "else": true, "end": true, "escape": true, "except": true,
	"exclusive": true, "exists": true, "explain": true, "fail": true, "for": true,
	"foreign": true, "from": true, "full": true, "glob": true, "group": true,
	"having": true, "if": true, "ignore": true, "immediate": true, "in": true,
	"index": true, "indexed": true, "initially": true, "inner": true, "insert": true,
	"instead": true, "intersect": true, "into": true, "is": true, "isnull": true,
	"join": true, "key": true, "left": true, "like": true, "limit": true,
	"match": true, "natural": true, "no": true, "not": true, "notnull": true,
	"null": true, "of": true, "offset": true, "on": true, "or": true, "order": true,
	"outer": true, "plan": true, "pragma": true, "primary": true, "query": true,
	"raise": true, "recursive": true, "references": true, "regexp": true,
	"reindex": true, "release": true, "rename": true, "replace": true,
	"restrict": true, "right": true, "rollback": true, "row": true, "savepoint": true,
	"select": true, "set": true, "table": true, "temp": true, "temporary": true,
	"then": true, "to": true, "transaction": true, "trigger": true, "union": true,
	"unique": true, "update": true, "using": true, "vacuum": true, "values": true,
	"view": true, "virtual": true, "when": true, "where": true, "with": true,
	"without": true,
}

// QuoteIdentifier는 SQLite 식별자를 안전하게 만듭니다.
func QuoteIdentifier(name string) string {
	// SQLite는 대소문자 구분이 없으므로 소문자로 변환하여 검사
	lowered := strings.ToLower(name)
	if sqliteKeywords[lowered] {
		return fmt.Sprintf(`"%s"`, name)
	}

	// 특수문자나 공백이 있는 경우도 따옴표로 감싸기
	if strings.ContainsAny(name, " -+()[]{}.,;") {
		return fmt.Sprintf(`"%s"`, name)
	}

	return name
}
