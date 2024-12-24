// exporter/types.go
package exporter

import (
	"reflect"
	"strings"
	"time"
)

// Exporter는 코드 생성을 실행하는 인터페이스입니다.
type Exporter interface {
	// ExportFiles는 여러 Excel 파일로부터 코드를 생성합니다.
	Export(tables []Table, opts Options) error

	// Language는 이 exporter가 지원하는 언어를 반환합니다.
	Language() string
}

// FactoryFunc는 새로운 exporter 인스턴스를 생성하는 함수 타입입니다.
type FactoryFunc func() Exporter

// Options는 코드 생성에 필요한 설정들을 포함합니다.
type Options struct {
	// 출력 디렉토리 경로
	OutputDir string

	// 생성될 코드의 패키지/네임스페이스 이름
	PackageName string

	// 타겟 언어별 추가 옵션들
	ExtraOptions map[string]interface{}

	// 템플릿 디렉토리 경로
	TemplateDir string

	// 데이터베이스 설정
	DBDriver string
	DBName   string
}

// Table represents a parsed Excel table structure
type Table struct {
	Name      string
	SheetName string
	Columns   []Column
	Relations []Relation
	Rows      [][]interface{} // 실제 데이터를 저장할 필드 추가
}

// Relation represents a table relationship
type Relation struct {
	SourceTable  string // 관계의 시작 테이블
	TargetTable  string // 관계의 대상 테이블
	RelationType string // 관계 유형 (hasOne, hasMany, belongsTo)
	ForeignKey   string // 외래 키 컬럼 이름
	ReferenceKey string // 참조 키 컬럼 이름
}

// Column은 테이블 컬럼 정의를 나타냅니다
type Column struct {
	Name     string     // 컬럼 이름
	Type     ColumnType // 컬럼 타입
	Tags     []TagValue //  태그
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
