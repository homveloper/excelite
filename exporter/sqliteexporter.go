package exporter

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteExporter implements database and schema generation for SQLite
type SQLiteExporter struct {
	BaseExporter
}

func NewSQLiteExporter() Exporter {
	return &SQLiteExporter{
		BaseExporter: NewBaseExporter("sqlite"),
	}
}

func (e *SQLiteExporter) Export(tables []Table, opts Options) error {
	// 1. Create database file
	dbPath := filepath.Join(opts.OutputDir, opts.PackageName+".db")
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// 2. Connect to SQLite database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	// 3. Enable foreign key support
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %v", err)
	}

	// 4. Create tables
	if err := e.createTables(db, tables); err != nil {
		return fmt.Errorf("failed to create tables: %v", err)
	}

	// 5. Insert data
	if err := e.insertData(db, tables); err != nil {
		return fmt.Errorf("failed to insert data: %v", err)
	}

	// 5. Generate schema file (optional)
	if err := e.generateSchemaFile(tables, opts); err != nil {
		return fmt.Errorf("failed to generate schema file: %v", err)
	}

	return nil
}

func (e *SQLiteExporter) insertData(db *sql.DB, tables []Table) error {
	// Begin transaction for all data insertion
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert data for each table
	for _, table := range tables {
		if err := e.insertTableData(tx, table); err != nil {
			return fmt.Errorf("failed to insert data for table %s: %v", table.Name, err)
		}
	}

	return tx.Commit()
}

func (e *SQLiteExporter) insertTableData(tx *sql.Tx, table Table) error {
	// Build insert statement
	var quotedColumns []string
	var placeholders []string
	var columnTypes []SQLiteType

	for _, col := range table.Columns {
		quotedColumns = append(quotedColumns, QuoteIdentifier(col.Name))
		placeholders = append(placeholders, "?")
		columnTypes = append(columnTypes, GetSQLiteType(col.Type))
	}

	quotedTableName := QuoteIdentifier(table.Name)
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		quotedTableName,
		strings.Join(quotedColumns, ", "),
		strings.Join(placeholders, ", "))

	// Prepare statement for bulk insert
	stmt, err := tx.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Insert each row
	for rowIdx, row := range table.Rows {
		values := make([]interface{}, len(quotedColumns))

		// Convert values according to SQLite types
		for i, col := range table.Columns {
			value := row[i]
			sqliteType := columnTypes[i]

			// Convert value based on SQLite type
			convertedValue, err := convertToSQLiteValue(value, sqliteType, col)
			if err != nil {
				return fmt.Errorf("error converting value at row %d, column %s: %v", rowIdx+1, col.Name, err)
			}

			values[i] = convertedValue
		}

		// Execute insert
		if _, err := stmt.Exec(values...); err != nil {
			return fmt.Errorf("error inserting row %d: %v", rowIdx+1, err)
		}
	}

	return nil
}

func convertToSQLiteValue(value interface{}, sqliteType SQLiteType, col Column) (interface{}, error) {
	// Handle nil values
	if value == nil {
		return nil, nil
	}

	// Convert based on SQLite type
	switch sqliteType {
	case SQLiteInteger:
		switch v := value.(type) {
		case string:
			return strconv.ParseInt(v, 10, 64)
		case float64:
			return int64(v), nil
		default:
			return value, nil
		}

	case SQLiteReal:
		switch v := value.(type) {
		case string:
			return strconv.ParseFloat(v, 64)
		default:
			return value, nil
		}

	case SQLiteBoolean:
		switch v := value.(type) {
		case string:
			return strconv.ParseBool(v)
		case int:
			return v != 0, nil
		default:
			return value, nil
		}

	case SQLiteDateTime:
		switch v := value.(type) {
		case string:
			return time.Parse("2006-01-02 15:04:05", v)
		default:
			return value, nil
		}

	case SQLiteText:
		if col.Type.IsArray {
			// Handle array types by converting to JSON
			jsonBytes, err := json.Marshal(value)
			if err != nil {
				return nil, err
			}
			return string(jsonBytes), nil
		}
		return fmt.Sprintf("%v", value), nil

	case SQLiteBlob:
		switch v := value.(type) {
		case []byte:
			return v, nil
		default:
			return nil, fmt.Errorf("unsupported type for BLOB: %T", value)
		}

	default:
		return value, nil
	}
}

func (e *SQLiteExporter) createTables(db *sql.DB, tables []Table) error {
	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Create each table
	for _, table := range tables {
		query := e.buildCreateTableQuery(table)

		log.Println("query:", query)

		if _, err := tx.Exec(query); err != nil {
			return fmt.Errorf("failed to create table %s: %v", table.Name, err)
		}

		// Create indices
		if err := e.createIndices(tx, table); err != nil {
			return fmt.Errorf("failed to create indices for table %s: %v", table.Name, err)
		}
	}

	// Commit transaction
	return tx.Commit()
}

func (e *SQLiteExporter) buildCreateTableQuery(table Table) string {
	var b strings.Builder

	quotedTableName := QuoteIdentifier(table.Name)
	b.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", quotedTableName))

	// Add id column as primary key
	b.WriteString("  id INTEGER PRIMARY KEY AUTOINCREMENT,\n")

	// check array column

	arrayFieldsUsed := make(map[string]bool)
	for _, col := range table.Columns {
		if col.Type.IsArray {
			arrayFieldsUsed[col.Name] = false
		}
	}

	// 배열인 필드를 따로 처리하지 말고 table 생성하는 단계에서 배열 필드를 만들어서 제공
	// 여기서는 주어진 테이블 필드 정보를 받아서 생성하는 것이 목적

	// Add columns
	for i, col := range table.Columns {
		quotedColName := QuoteIdentifier(col.Name)
		constraints := e.buildColumnConstraints(col)
		sqlType := GetSQLiteType(col.Type).String()

		b.WriteString(fmt.Sprintf("  %s %s%s", quotedColName, sqlType, constraints))

		if i < len(table.Columns)-1 {
			b.WriteString(",\n")
		}
	}

	// 	// Add timestamp columns
	// 	b.WriteString(",\n  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,\n")
	// 	b.WriteString("  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,\n")
	// 	b.WriteString("  deleted_at DATETIME DEFAULT NULL\n")

	// Add foreign key constraints
	for _, rel := range table.Relations {
		if rel.RelationType == "belongsTo" {
			quotedFK := QuoteIdentifier(rel.ForeignKey)
			quotedTargetTable := QuoteIdentifier(rel.TargetTable)

			b.WriteString(fmt.Sprintf(",\n  FOREIGN KEY(%s) REFERENCES %s(id)",
				quotedFK, quotedTargetTable))
		}
	}

	b.WriteString(");\n")

	// 	// Add trigger for updated_at
	// 	b.WriteString(fmt.Sprintf(`
	// CREATE TRIGGER IF NOT EXISTS tg_%s_updated_at
	//   AFTER UPDATE ON %s
	//   BEGIN
	//     UPDATE %s SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
	//   END;`, table.Name, table.Name, table.Name))

	return b.String()
}

func (e *SQLiteExporter) buildColumnConstraints(col Column) string {
	var constraints []string

	// Handle NOT NULL
	if HasTag(col.Tags, TagNotNull) {
		constraints = append(constraints, "NOT NULL")
	}

	// Handle UNIQUE
	if col.IsUnique || HasTag(col.Tags, TagUnique) {
		constraints = append(constraints, "UNIQUE")
	}

	// Handle DEFAULT
	if defaultVal, ok := GetTagValue(col.Tags, TagDefault); ok {
		constraints = append(constraints, fmt.Sprintf("DEFAULT %s", defaultVal))
	}

	if len(constraints) > 0 {
		return " " + strings.Join(constraints, " ")
	}
	return ""
}

func (e *SQLiteExporter) createIndices(tx *sql.Tx, table Table) error {
	// Create index for indexed columns

	for _, col := range table.Columns {
		if HasTag(col.Tags, TagIndex) {
			quotedTableName := QuoteIdentifier(table.Name)
			quotedColumnName := QuoteIdentifier(col.Name)

			indexName := fmt.Sprintf("idx_%s_%s", quotedTableName, quotedColumnName)
			query := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s(%s);",
				indexName, quotedTableName, quotedColumnName)

			if _, err := tx.Exec(query); err != nil {
				return err
			}
		}
	}

	// Create indices for foreign keys
	for _, rel := range table.Relations {
		if rel.RelationType == "belongsTo" {

			quotedTableName := QuoteIdentifier(table.Name)
			quotedFK := QuoteIdentifier(rel.ForeignKey)

			indexName := fmt.Sprintf("idx_%s_%s", quotedTableName, quotedFK)
			query := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s(%s);",
				indexName, quotedTableName, quotedFK)

			if _, err := tx.Exec(query); err != nil {
				return err
			}
		}
	}

	return nil
}

func buildColumnDefinition(col Column) string {
	typeDef := GetSQLiteTypeFromColumnType(col.Type)

	// Apply column constraints from tags
	typeDef.AllowNull = !HasTag(col.Tags, TagNotNull)
	typeDef.IsUnique = col.IsUnique || HasTag(col.Tags, TagUnique)
	typeDef.IsPrimary = HasTag(col.Tags, TagPrimaryKey)

	// Handle default value
	if defaultVal, ok := GetTagValue(col.Tags, TagDefault); ok {
		typeDef.DefaultVal = defaultVal
	}

	// Handle size constraint
	if sizeVal, ok := GetTagValue(col.Tags, TagSize); ok {
		if size, err := strconv.ParseInt(sizeVal, 10, 64); err == nil {
			typeDef.Size = size
		}
	}

	return typeDef.BuildColumnDefinition(col.Name)
}

// generateSchemaFile creates a SQL file with the schema definition
func (e *SQLiteExporter) generateSchemaFile(tables []Table, opts Options) error {
	var schema strings.Builder

	schema.WriteString("-- Schema generated by excelite\n\n")
	schema.WriteString("PRAGMA foreign_keys=ON;\n\n")

	for _, table := range tables {
		schema.WriteString(e.buildCreateTableQuery(table))
		schema.WriteString("\n\n")
	}

	schemaPath := filepath.Join(opts.OutputDir, "schema.sql")
	return os.WriteFile(schemaPath, []byte(schema.String()), 0644)
}
