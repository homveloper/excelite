// parser.go
package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// ValueParser 인터페이스
type ValueParser interface {
	Parse(value string) (Value, error)
	Type() ColumnType
}

// 기본 파서 구조체
type baseParser struct {
	columnName string
	columnType ColumnType
}

func (p *baseParser) Type() ColumnType {
	return p.columnType
}

// ReflectParser 리플렉션 기반 파서
type ReflectParser struct {
	baseParser
	parse func(string) (interface{}, error)
}

func NewReflectParser(columnName string, columnType ColumnType, parse func(string) (interface{}, error)) *ReflectParser {
	return &ReflectParser{
		baseParser: baseParser{
			columnName: columnName,
			columnType: columnType,
		},
		parse: parse,
	}
}

func (p *ReflectParser) Parse(value string) (Value, error) {
	if value = strings.TrimSpace(value); value == "" {
		return ZeroValue(p.columnType), nil
	}

	parsed, err := p.parse(value)
	if err != nil {
		return ZeroValue(p.columnType), fmt.Errorf("column %s: %v", p.columnName, err)
	}

	return NewValue(p.columnType, parsed), nil
}

// CreateParser creates a parser for the given column
func CreateParser(column Column) ValueParser {
	if column.Type.IsArray {
		return createArrayParser(column)
	}
	return createValueParser(column)
}

func createValueParser(column Column) ValueParser {
	switch column.Type.Type.Kind() {
	case reflect.Int32:
		return NewReflectParser(column.Name, column.Type, func(s string) (interface{}, error) {
			val, err := strconv.ParseInt(s, 10, 32)
			return int32(val), err
		})

	case reflect.Int64:
		return NewReflectParser(column.Name, column.Type, func(s string) (interface{}, error) {
			return strconv.ParseInt(s, 10, 64)
		})

	case reflect.Float64:
		return NewReflectParser(column.Name, column.Type, func(s string) (interface{}, error) {
			return strconv.ParseFloat(s, 64)
		})

	case reflect.Bool:
		return NewReflectParser(column.Name, column.Type, func(s string) (interface{}, error) {
			return strconv.ParseBool(s)
		})

	case reflect.String:
		return NewReflectParser(column.Name, column.Type, func(s string) (interface{}, error) {
			return s, nil
		})
	}

	// time.Time 특별 처리
	if column.Type.Type == reflect.TypeOf(time.Time{}) {
		return NewTimeParser(column.Name, column.Type)
	}

	// 기본값은 문자열 파서
	return NewReflectParser(column.Name, StringType, func(s string) (interface{}, error) {
		return s, nil
	})
}

// TimeParser for time.Time
type TimeParser struct {
	baseParser
	formats []timeFormat
}

type timeFormat struct {
	format     string
	hasChanged bool
}

func NewTimeParser(columnName string, columnType ColumnType) *TimeParser {
	return &TimeParser{
		baseParser: baseParser{
			columnName: columnName,
			columnType: columnType,
		},
		formats: []timeFormat{
			{"2006-01-02 15:04:05.999", false},
			{"2006-01-02 15:04:05.999Z", true},
			{"2006-01-02T15:04:05.999", false},
			{"2006-01-02T15:04:05.999Z", true},
			{"2006-01-02 15:04:05", false},
			{"2006-01-02T15:04:05", false},
			{"2006-01-02T15:04:05Z", true},
			{"2006-01-02", false},
		},
	}
}

func (p *TimeParser) Parse(value string) (Value, error) {
	if value = strings.TrimSpace(value); value == "" {
		return ZeroValue(p.columnType), nil
	}

	var lastErr error
	for _, tf := range p.formats {
		t, err := time.Parse(tf.format, value)
		if err == nil {
			if !tf.hasChanged && !strings.ContainsAny(value, "Zz+-") {
				if t.Hour() != 0 || t.Minute() != 0 || t.Second() != 0 {
					t = t.UTC()
				}
			}
			return NewValue(p.columnType, t), nil
		}
		lastErr = err
	}
	return ZeroValue(p.columnType), fmt.Errorf("column %s: failed to parse date '%s': %v", p.columnName, value, lastErr)
}

func createArrayParser(column Column) ValueParser {
	baseParser := createValueParser(Column{
		Name: column.Name,
		Type: ColumnType{
			Type:    column.Type.Type.Elem(),
			SQLType: column.Type.SQLType,
		},
	})

	return NewReflectParser(column.Name, column.Type, func(s string) (interface{}, error) {
		items := strings.Split(s, ",")
		values := make([]interface{}, 0, len(items))

		for _, item := range items {
			parsed, err := baseParser.Parse(item)
			if err != nil {
				return nil, err
			}
			if !parsed.IsZero() {
				values = append(values, parsed.Interface())
			}
		}

		jsonData, err := json.Marshal(values)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal array: %v", err)
		}
		return string(jsonData), nil
	})
}
