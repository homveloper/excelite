// columnbuilder.go
package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// ColumnBuilder는 컬럼 생성에 필요한 데이터를 보관합니다
type ColumnBuilder struct {
	fieldNames []string
	types      []string
	tags       []string
}

// NewColumnBuilder는 새로운 ColumnBuilder를 생성합니다
func NewColumnBuilder(fieldNames, types, tags []string) *ColumnBuilder {
	return &ColumnBuilder{
		fieldNames: fieldNames,
		types:      types,
		tags:       tags,
	}
}

// BuildColumns는 엑셀 데이터로부터 컬럼 리스트를 생성합니다
func (cb *ColumnBuilder) BuildColumns() ([]Column, error) {
	// 컬럼별 배열 크기 계산
	arrayColumnCounts := make(map[string]int)
	processedNames := make(map[string]string) // 원본 이름 -> 포맷된 이름 매핑

	// 먼저 모든 필드 이름을 포맷팅하고 매핑 생성
	for i, name := range cb.fieldNames {
		name = strings.TrimSpace(name)
		if name == "" || i >= len(cb.types) {
			continue
		}

		typeStr := strings.TrimSpace(cb.types[i])
		formattedName := FormatColumnName(name)
		processedNames[name] = formattedName

		if strings.HasPrefix(typeStr, "array<") {
			arrayColumnCounts[formattedName]++
		}
	}

	// 컬럼 맵 생성
	columnMap := make(map[string]*Column)

	// 배열 컬럼 처리
	for origName, count := range arrayColumnCounts {
		formattedName := processedNames[origName]
		if IsReservedColumnName(formattedName) {
			return nil, fmt.Errorf("column name '%s' is reserved by the system", formattedName)
		}

		baseTypeStr := ""
		// 해당 이름을 가진 첫 번째 컬럼의 타입 찾기
		for i, fieldName := range cb.fieldNames {
			if strings.TrimSpace(fieldName) == origName && i < len(cb.types) {
				typeStr := strings.TrimSpace(cb.types[i])
				baseTypeStr = strings.TrimSuffix(strings.TrimPrefix(typeStr, "array<"), ">")
				break
			}
		}
		if baseTypeStr == "" {
			continue
		}

		baseType := ParseColumnType(baseTypeStr)

		// 배열 컬럼 추가
		columnMap[formattedName] = &Column{
			Name: formattedName,
			Type: ColumnType{
				Type:     reflect.SliceOf(baseType.Type),
				SQLType:  "TEXT",
				IsArray:  true,
				BaseType: &baseType,
			},
			Tags: `gorm:"type:text"`,
		}

		// 개별 컬럼 추가
		for i := 0; i < count; i++ {
			columnName := fmt.Sprintf("%s_%d", formattedName, i)
			columnMap[columnName] = &Column{
				Name: columnName,
				Type: baseType,
				Tags: fmt.Sprintf(`gorm:"column:%s_%d"`, strings.ToLower(formattedName), i),
			}
		}
	}

	// 일반 컬럼 처리
	for i, origName := range cb.fieldNames {
		origName = strings.TrimSpace(origName)
		if origName == "" || i >= len(cb.types) {
			continue
		}

		formattedName := processedNames[origName]
		if IsReservedColumnName(formattedName) {
			return nil, fmt.Errorf("column name '%s' is reserved by the system", formattedName)
		}

		// 이미 처리된 배열 컬럼은 건너뛰기
		if _, exists := columnMap[formattedName]; exists {
			continue
		}

		typeStr := strings.TrimSpace(cb.types[i])
		tagValue := ""
		if i < len(cb.tags) {
			tagValue = strings.TrimSpace(cb.tags[i])
		}

		// design 태그가 있는 컬럼 무시
		if strings.Contains(tagValue, "design") {
			continue
		}

		gormTag, _ := parseTagToGORMTag(tagValue)
		columnMap[formattedName] = &Column{
			Name:     formattedName,
			Type:     ParseColumnType(typeStr),
			Tags:     gormTag,
			IsUnique: strings.Contains(tagValue, "unique"),
		}
	}

	// 최종 컬럼 리스트 생성
	var columns []Column
	for _, col := range columnMap {
		columns = append(columns, *col)
	}

	// 컬럼 정렬
	sort.Slice(columns, func(i, j int) bool {
		return columns[i].Name < columns[j].Name
	})

	return columns, nil
}

func (cb *ColumnBuilder) calculateArrayCounts() map[string]int {
	counts := make(map[string]int)
	for i, name := range cb.fieldNames {
		name = strings.TrimSpace(name)
		if name == "" || i >= len(cb.types) {
			continue
		}

		typeStr := strings.TrimSpace(cb.types[i])
		if strings.HasPrefix(typeStr, "array<") {
			counts[name]++
		}
	}
	return counts
}
