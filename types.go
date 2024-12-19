// types.go
package main

import (
	"reflect"
	"strconv"
	"strings"
	"time"
)

type ErrorDomain = string

var errdoamin = struct {
	Generator ErrorDomain
}{
	Generator: ErrorDomain("generator"),
}

// Column은 테이블 컬럼 정의를 나타냅니다
type Column struct {
	Name     string     // 컬럼 이름
	Type     ColumnType // 컬럼 타입
	Tags     string     // GORM 태그
	IsUnique bool       // 유니크 컬럼 여부
}

// ColumnType은 컬럼의 타입 정보를 나타냅니다
type ColumnType struct {
	Type     reflect.Type // Go 타입
	SQLType  string       // SQL 타입
	IsArray  bool         // 배열 여부
	BaseType *ColumnType  // 배열인 경우 기본 타입
}

// 기본 타입 정의
var (
	Int32Type = ColumnType{
		Type:    reflect.TypeOf(int32(0)),
		SQLType: "INTEGER",
	}

	Int64Type = ColumnType{
		Type:    reflect.TypeOf(int64(0)),
		SQLType: "BIGINT",
	}

	Float64Type = ColumnType{
		Type:    reflect.TypeOf(float64(0)),
		SQLType: "REAL",
	}

	BoolType = ColumnType{
		Type:    reflect.TypeOf(bool(false)),
		SQLType: "BOOLEAN",
	}

	StringType = ColumnType{
		Type:    reflect.TypeOf(""),
		SQLType: "TEXT",
	}

	DateTimeType = ColumnType{
		Type:    reflect.TypeOf(time.Time{}),
		SQLType: "DATETIME",
	}

	BytesType = ColumnType{
		Type:    reflect.TypeOf([]byte{}),
		SQLType: "BLOB",
	}
)

// ParseColumnType은 문자열 타입 정의를 파싱하여 ColumnType을 반환합니다
func ParseColumnType(typeStr string) ColumnType {
	typeStr = strings.TrimSpace(strings.ToLower(typeStr))

	// 배열 타입 처리
	if strings.HasPrefix(typeStr, "array<") && strings.HasSuffix(typeStr, ">") {
		baseTypeStr := strings.TrimSuffix(strings.TrimPrefix(typeStr, "array<"), ">")
		baseType := ParseColumnType(baseTypeStr)
		return ColumnType{
			Type:     reflect.SliceOf(baseType.Type),
			SQLType:  "TEXT", // 배열은 JSON으로 저장되므로 TEXT
			IsArray:  true,
			BaseType: &baseType,
		}
	}

	// 기본 타입 처리
	switch typeStr {
	case "int", "int32", "integer":
		return Int32Type
	case "int64", "bigint":
		return Int64Type
	case "float", "float64", "double":
		return Float64Type
	case "bool", "boolean":
		return BoolType
	case "time", "datetime", "timestamp", "date":
		return DateTimeType
	case "[]byte", "blob":
		return BytesType
	case "string", "text", "varchar":
		return StringType
	default:
		return StringType
	}
}

// GoTypeString은 Go 코드 생성에 사용할 타입 문자열을 반환합니다
func (ct ColumnType) GoTypeString() string {
	if ct.IsArray {
		return "[]" + ct.BaseType.Type.String()
	}
	return ct.Type.String()
}

// SQLTypeString은 SQL 스키마 생성에 사용할 타입 문자열을 반환합니다
func (ct ColumnType) SQLTypeString() string {
	if ct.IsArray {
		return "TEXT"
	}
	return ct.SQLType
}

// Value represents a typed value using reflection
type Value struct {
	Type  ColumnType
	Value reflect.Value
}

func NewValue(columnType ColumnType, value interface{}) Value {
	if value == nil {
		return Value{
			Type:  columnType,
			Value: reflect.Zero(columnType.Type),
		}
	}
	return Value{
		Type:  columnType,
		Value: reflect.ValueOf(value),
	}
}

func (v Value) Interface() interface{} {
	if !v.Value.IsValid() {
		return nil
	}
	return v.Value.Interface()
}

func (v Value) IsZero() bool {
	return !v.Value.IsValid() || v.Value.IsZero()
}

// ZeroValue creates a zero value for the given column type
func ZeroValue(columnType ColumnType) Value {
	return Value{
		Type:  columnType,
		Value: reflect.Zero(columnType.Type),
	}
}

// 지원하는 태그 타입들
const (
	TagPrimaryKey = "pk"      // 기본키
	TagUnique     = "unique"  // 유니크 키
	TagIndex      = "index"   // 인덱스
	TagNotNull    = "notnull" // NOT NULL
	TagAutoInc    = "autoinc" // 자동 증가
	TagDefault    = "default" // 기본값
	TagSize       = "size"    // 크기 제한
	TagForeignKey = "fk"      // 외래키
)

/*
| SourceTable | TargetTable | RelationType | ForeignKey | ReferenceKey |
|------------|-------------|--------------|------------|--------------|
| User       | Post        | hasMany      | UserID     | ID          |
| User       | Profile     | hasOne       | UserID     | ID          |
| Post       | User        | belongsTo    | UserID     | ID          |
*/

// Relation represents a table relationship
type Relation struct {
	SourceTable  string // 관계의 시작 테이블
	TargetTable  string // 관계의 대상 테이블
	RelationType string // 관계 유형 (hasOne, hasMany, belongsTo)
	ForeignKey   string // 외래 키 컬럼 이름
	ReferenceKey string // 참조 키 컬럼 이름
}

// Table represents a database table definition
type Table struct {
	Name       string     // 테이블 이름
	SourceFile string     // 원본 Excel 파일 경로
	TempFile   string     // 임시 파일 경로
	SheetName  string     // Excel 시트 이름
	Columns    []Column   // 테이블 컬럼들
	Relations  []Relation // 이 테이블과 관련된 관계들
}

// parseArrayType은 array<type> 형식의 타입 문자열을 파싱합니다
func parseArrayType(typeStr string) (baseType string, size int) {
	if !strings.HasPrefix(typeStr, "array<") || !strings.HasSuffix(typeStr, ">") {
		return typeStr, 0
	}

	// array<type> 형식에서 type 추출
	baseType = strings.TrimSuffix(strings.TrimPrefix(typeStr, "array<"), ">")

	// 기본 배열 크기 설정
	size = 6 // 기본값으로 6 설정

	return baseType, size
}

// IsArrayColumn은 주어진 이름이 배열의 개별 컬럼인지 확인합니다
func IsArrayColumn(name string) (baseName string, index int, isArray bool) {
	parts := strings.Split(name, "_")
	if len(parts) < 2 {
		return name, -1, false
	}

	lastPart := parts[len(parts)-1]
	if _, err := strconv.Atoi(lastPart); err == nil {
		baseName = strings.Join(parts[:len(parts)-1], "_")
		index, _ = strconv.Atoi(lastPart)
		return baseName, index, true
	}

	return name, -1, false
}

// gorm keyword

var reservedColumnNames = map[string]bool{
	"id":         true,
	"created_at": true,
	"updated_at": true,
	"deleted_at": true,
}

// IsReservedColumnName checks if the column name is reserved
func IsReservedColumnName(name string) bool {
	return reservedColumnNames[strings.ToLower(name)]
}
