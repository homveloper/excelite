// exporter/relations.go
package exporter

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// parseRelations는 #Relation 시트에서 테이블 간의 관계를 파싱합니다.
func parseRelations(f *excelize.File) ([]Relation, error) {
	empty := make([]Relation, 0)

	relationSheet := "#Relation"
	if !contains(f.GetSheetList(), relationSheet) {
		return empty, nil // 관계 시트가 없으면 빈 슬라이스 반환
	}

	rows, err := f.GetRows(relationSheet)
	if err != nil {
		return nil, fmt.Errorf("failed to read relation sheet: %v", err)
	}

	if len(rows) < 2 { // 헤더 + 최소 1개의 데이터 필요
		return empty, nil
	}

	// 헤더 검증 및 컬럼 인덱스 찾기
	colIndexes := map[string]int{
		"SourceTable":  -1,
		"TargetTable":  -1,
		"RelationType": -1,
		"ForeignKey":   -1,
		"ReferenceKey": -1,
	}

	for i, cell := range rows[0] {
		colName := strings.TrimSpace(cell)
		if _, ok := colIndexes[colName]; ok {
			colIndexes[colName] = i
		}
	}

	// 필수 컬럼 존재 확인
	for col, idx := range colIndexes {
		if idx == -1 {
			return nil, fmt.Errorf("required column %s not found in relation sheet", col)
		}
	}

	// 관계 데이터 파싱
	var relations []Relation
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) < len(colIndexes) {
			continue // 불완전한 행 무시
		}

		relation := Relation{
			SourceTable:  strings.TrimSpace(row[colIndexes["SourceTable"]]),
			TargetTable:  strings.TrimSpace(row[colIndexes["TargetTable"]]),
			RelationType: strings.TrimSpace(row[colIndexes["RelationType"]]),
			ForeignKey:   strings.TrimSpace(row[colIndexes["ForeignKey"]]),
			ReferenceKey: strings.TrimSpace(row[colIndexes["ReferenceKey"]]),
		}

		// 기본값 처리
		if relation.ReferenceKey == "" {
			relation.ReferenceKey = "ID"
		}
		if relation.ForeignKey == "" {
			relation.ForeignKey = relation.SourceTable + "ID"
		}

		// 관계 타입 정규화
		relation.RelationType = normalizeRelationType(relation.RelationType)

		relations = append(relations, relation)
	}

	return relations, nil
}

// normalizeRelationType은 관계 타입을 표준 형식으로 변환합니다.
func normalizeRelationType(relType string) string {
	relType = strings.ToLower(strings.TrimSpace(relType))
	switch relType {
	case "hasone", "has_one", "has-one":
		return "hasOne"
	case "hasmany", "has_many", "has-many":
		return "hasMany"
	case "belongsto", "belongs_to", "belongs-to":
		return "belongsTo"
	default:
		return relType
	}
}

// contains는 슬라이스에 문자열이 포함되어 있는지 확인합니다.
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func assignRelationsToTables(tables []Table, relations []Relation) []Table {
	result := make([]Table, len(tables))

	// Create a map for quick table lookup
	tableMap := make(map[string]int)
	for i, table := range tables {
		// Copy the original table
		result[i] = Table{
			Name:      table.Name,
			Columns:   append([]Column(nil), table.Columns...),
			Relations: make([]Relation, 0),
			SheetName: table.SheetName,
		}
		tableMap[table.Name] = i
	}

	// Assign relations to appropriate tables
	for _, rel := range relations {
		if idx, ok := tableMap[rel.SourceTable]; ok {
			result[idx].Relations = append(result[idx].Relations, rel)
		}
	}

	return result
}
