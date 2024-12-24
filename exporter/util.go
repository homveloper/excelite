// exporter/excel.go
package exporter

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ParseExcelFile은 Excel 파일을 파싱하여 테이블 정의를 반환합니다.
func ParseExcelFile(filePath string) ([]Table, error) {
	// ~$로 시작하는 임시 파일 무시
	if strings.HasPrefix(filePath, "~$") {
		return nil, nil
	}

	// Excel 파일 열기
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %v", err)
	}
	defer f.Close()

	var tables []Table

	// 각 시트 처리
	for _, sheetName := range f.GetSheetList() {
		// #로 시작하는 시트는 건너뛰기 (메타데이터/설정 시트)
		if strings.HasPrefix(sheetName, "#") {
			continue
		}

		// 시트의 데이터 읽기
		rows, err := f.GetRows(sheetName)
		if err != nil {
			return nil, fmt.Errorf("failed to read sheet %s: %v", sheetName, err)
		}

		if len(rows) < 4 { // 최소 4줄(컬럼명, 태그, 타입, 데이터) 필요
			continue
		}

		// 시트에서 테이블 정의 파싱
		table, err := parseSheet(sheetName, rows)
		if err != nil {
			return nil, fmt.Errorf("failed to parse sheet %s: %v", sheetName, err)
		}

		tables = append(tables, table)
	}

	relations, err := parseRelations(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse relations: %v", err)
	}

	tables = assignRelationsToTables(tables, relations)

	return tables, nil
}

// parseSheet는 시트 데이터로부터 테이블 정의를 파싱합니다.
func parseSheet(sheetName string, rows [][]string) (Table, error) {

	// 첫 번째 행: 컬럼명
	// 두 번째 행: 태그
	// 세 번째 행: 타입
	columnNames := rows[0]
	columnTags := rows[1]
	columnTypes := rows[2]

	table := Table{
		Name:      formatTableName(sheetName),
		SheetName: sheetName,
	}

	// TODO: 컬럼 타입이 배열이면,  TEXT 타입인 필드하나가 있고,  배열 원소의 수 만큼  원소의 해당 타입으로 FIELDNAME_0, FIELDNAME_1, ... 으로 추가 필드가 생성되어야 함

	for i := 0; i < len(columnNames); i++ {
		name := ParseColumnName(columnNames[i])
		if len(name) <= 0 {
			continue
		}

		tagValeus := ParseColumnTags(parseTags(columnTags[i]))

		typeStr := strings.TrimSpace(columnTypes[i])

		// 디자인용 컬럼은 건너뛰기
		if HasTag(tagValeus, TagDesign) {
			continue
		}

		columnType := ParseColumnType(typeStr)

		column := Column{
			Name:     name,
			Type:     columnType,
			Tags:     tagValeus,
			IsUnique: HasTag(tagValeus, TagUnique),
		}

		table.Columns = append(table.Columns, column)
	}

	return table, nil
}

func formatTableName(name string) string {
	name = strings.TrimSpace(name)
	parts := strings.Fields(name)
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(string(part[0])) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, "")
}

func ParseColumnName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	// 첫 글자를 대문자로 변환
	parts := strings.Fields(name)
	for i, part := range parts {
		if len(part) > 0 {
			// 첫 문자를 대문자로, 나머지는 그대로 유지
			parts[i] = strings.ToUpper(string(part[0])) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

// parseTags는 태그 문자열을 태그 슬라이스로 파싱합니다.
func parseTags(tagStr string) []string {
	tags := strings.Split(strings.TrimSpace(tagStr), ",")
	var result []string
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			result = append(result, tag)
		}
	}
	return result
}

// extractDefaultValue extracts default value from tags
func extractDefaultValue(tags string) string {
	if idx := strings.Index(tags, "default:"); idx != -1 {
		start := idx + len("default:")
		end := strings.Index(tags[start:], ";")
		if end == -1 {
			end = len(tags[start:])
		}
		return tags[start : start+end]
	}
	return ""
}

func FormatColumnName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	// 첫 글자를 대문자로 변환
	parts := strings.Fields(name)
	for i, part := range parts {
		if len(part) > 0 {
			// 첫 문자를 대문자로, 나머지는 그대로 유지
			parts[i] = strings.ToUpper(string(part[0])) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

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
