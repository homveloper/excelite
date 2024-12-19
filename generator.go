package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/samber/oops"
	"github.com/schollz/progressbar/v3"
	"github.com/xuri/excelize/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewGenerator(outputDir string) *Generator {
	return &Generator{
		OutputDir: outputDir,
	}
}

// Generator is the main generator structure
type Generator struct {
	OutputDir string     // 생성된 파일들이 저장될 디렉토리
	Tables    []Table    // 모든 테이블 정의
	Relations []Relation // 전체 관계 목록
	mu        sync.Mutex // 동시성 제어를 위한 뮤텍스
}

func (g *Generator) Destruct() {
}

// Worker 함수도 간단해짐
func (g *Generator) Worker(id int, files <-chan string, bar *progressbar.ProgressBar) error {

	for file := range files {
		if err := g.parseExcel(file); err != nil {
			return oops.Wrapf(err, "Worker %d: Error processing %s: %v\n", id, file, err)
		}
		bar.Add(1)
	}
	return nil
}

// 임시 파일 생성 함수
func (g *Generator) copyToTemp(srcPath string) (string, error) {
	// 임시 파일 이름 생성
	timestamp := time.Now().Format("20060102_150405")
	fileName := strings.TrimSuffix(filepath.Base(srcPath), filepath.Ext(srcPath))
	tempPath := filepath.Join(os.TempDir(), fmt.Sprintf("%s_%s%s",
		fileName, timestamp, filepath.Ext(srcPath)))

	src, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %v", err)
	}
	defer src.Close()

	dst, err := os.Create(tempPath)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(tempPath)
		return "", fmt.Errorf("failed to copy file: %v", err)
	}

	return tempPath, nil
}

// copyExcelFile creates a temporary copy of the Excel file
func (g *Generator) copyExcelFile(srcPath string) (string, error) {
	// 임시 파일 이름 생성 (원본파일명_timestamp.xlsx)
	timestamp := time.Now().Format("20060102_150405")
	fileName := strings.TrimSuffix(filepath.Base(srcPath), filepath.Ext(srcPath))
	tempPath := filepath.Join(os.TempDir(), fmt.Sprintf("%s_%s%s",
		fileName, timestamp, filepath.Ext(srcPath)))

	// 원본 파일 열기
	src, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %v", err)
	}
	defer src.Close()

	// 임시 파일 생성
	dst, err := os.Create(tempPath)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	defer dst.Close()

	// 파일 복사
	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(tempPath) // 복사 실패 시 임시 파일 삭제
		return "", fmt.Errorf("failed to copy file: %v", err)
	}

	return tempPath, nil
}

func (g *Generator) parseExcel(filePath string) error {
	errBuilder := oops.With("file", filePath)

	// ~$로 시작하는 임시 파일은 무시
	if strings.HasPrefix(filepath.Base(filePath), "~$") {
		return nil
	}

	// 임시 파일 생성
	timestamp := time.Now().Format("20060102_150405")
	fileName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	tempPath := filepath.Join(os.TempDir(), fmt.Sprintf("%s_%s%s",
		fileName, timestamp, filepath.Ext(filePath)))

	// 원본 파일 복사
	src, err := os.Open(filePath)
	if err != nil {
		return errBuilder.Errorf("failed to open source file: %v", err)
	}
	defer src.Close()

	dst, err := os.Create(tempPath)
	if err != nil {
		return errBuilder.Errorf("failed to create temp file: %v", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(tempPath)
		return errBuilder.Errorf("failed to copy file: %v", err)
	}

	// 임시 파일 열기
	f, err := excelize.OpenFile(tempPath)
	if err != nil {
		return errBuilder.Errorf("failed to open Excel file: %v", err)
	}
	defer f.Close()

	// #Relation 시트 확인 및 파싱
	sheets := f.GetSheetList()
	for _, sheet := range sheets {
		if sheet == "#Relation" {
			if err := g.parseRelations(f); err != nil {
				return errBuilder.Errorf("Warning: Error parsing #Relation sheet: %v\n", err)
			}
		}
	}

	// 일반 테이블 시트 처리
	for _, sheet := range sheets {
		if strings.HasPrefix(sheet, "#") {
			continue
		}

		if err := g.parseSheet(f, sheet, filePath, tempPath); err != nil {
			return oops.With("sheet", sheet).Errorf("Warning: Error parsing sheet : %v\n", err)
		}
	}

	return nil
}

func (g *Generator) GenerateFiles() error {
	log.Println("Generating files...")

	// 각 테이블별로 파일 생성
	for _, table := range g.Tables {
		// 1. GORM 모델 생성
		if err := g.generateGormModel(table); err != nil {
			return oops.Wrap(err)
		}

		// 2. SQLite 스키마 생성
		if err := g.generateSQLSchema(table); err != nil {
			return oops.Wrap(err)
		}

		// 3. SQLite DB 파일 생성 및 데이터 삽입
		if err := g.generateDatabase(table); err != nil {
			return oops.Wrap(err)
		}
	}

	return nil
}

func (g *Generator) generateGormModel(table Table) error {
	// 필요한 import 패키지 수집
	imports := g.collectImports(table.Columns)

	// 모델 정의 생성
	modelData := struct {
		Name        string
		Imports     []string
		Columns     []Column
		ArrayFields []Column
	}{
		Name:        table.Name,
		Imports:     imports,
		Columns:     table.Columns,
		ArrayFields: filterArrayColumns(table.Columns),
	}

	// 템플릿 정의
	tmpl := template.Must(template.New("model").Parse(`package models

import (
{{range .Imports}}    "{{.}}"
{{end}}
)

type {{.Name}} struct {
    gorm.Model
{{range .Columns}}    {{.Name}} {{.Type.GoTypeString}} {{if .Tags}}` + "`{{.Tags}}`" + `{{end}}
{{end}}
}

{{if .ArrayFields}}
// BeforeSave handles array field serialization
func (m *{{.Name}}) BeforeSave(tx *gorm.DB) error {
{{range .ArrayFields}}    // Handle {{.Name}} array
    if m.{{.Name}} != nil {
        for i, v := range m.{{.Name}} {
            fieldName := fmt.Sprintf("{{.Name}}_%d", i)
            tx.Statement.SetColumn(fieldName, v)
        }
    }
{{end}}
    return nil
}

// AfterFind handles array field deserialization
func (m *{{.Name}}) AfterFind(tx *gorm.DB) error {
{{range .ArrayFields}}    // Initialize {{.Name}} array
    m.{{.Name}} = make([]{{.Type.BaseType.GoTypeString}}, 0)
    for i := 0; ; i++ {
        field := reflect.ValueOf(m).Elem().FieldByName(fmt.Sprintf("{{.Name}}_%d", i))
        if !field.IsValid() {
            break
        }
        if !field.IsZero() {
            m.{{.Name}} = append(m.{{.Name}}, field.Interface().({{.Type.BaseType.GoTypeString}}))
        }
    }
{{end}}
    return nil
}
{{end}}`))

	// 파일 생성 및 템플릿 실행
	fileName := filepath.Join(g.OutputDir, strings.ToLower(table.Name)+".go")
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, modelData)
}

// 필요한 import 패키지 수집
func (g *Generator) collectImports(columns []Column) []string {
	imports := make(map[string]bool)
	imports["gorm.io/gorm"] = true

	hasArrayField := false
	for _, col := range columns {
		if col.Type.IsArray {
			hasArrayField = true
		}

		switch col.Type.Type {
		case reflect.TypeOf(time.Time{}):
			imports["time"] = true
		}
	}

	if hasArrayField {
		imports["fmt"] = true
		imports["reflect"] = true
	}

	// 정렬된 import 목록 생성
	var sortedImports []string
	for pkg := range imports {
		sortedImports = append(sortedImports, pkg)
	}
	sort.Strings(sortedImports)

	return sortedImports
}

// 배열 타입 컬럼 필터링
func filterArrayColumns(columns []Column) []Column {
	var arrayColumns []Column
	processedNames := make(map[string]bool)

	for _, col := range columns {
		if col.Type.IsArray && !processedNames[col.Name] {
			arrayColumns = append(arrayColumns, col)
			processedNames[col.Name] = true
		}
	}

	return arrayColumns
}

// generateSQLSchema 함수 수정
func (g *Generator) generateSQLSchema(table Table) error {
	var schema strings.Builder
	schema.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", QuoteIdentifier(strings.ToLower(table.Name))))

	// 기본 ID 및 타임스탬프 컬럼
	schema.WriteString("    id INTEGER PRIMARY KEY AUTOINCREMENT,\n")
	schema.WriteString("    created_at DATETIME,\n")
	schema.WriteString("    updated_at DATETIME,\n")
	schema.WriteString("    deleted_at DATETIME,\n")

	// 일반 컬럼들을 위한 정의 수집
	var columnDefs []string

	processedArrayColumns := make(map[string]bool)

	for _, col := range table.Columns {
		if col.Type.IsArray {
			if processedArrayColumns[col.Name] {
				continue
			}
			processedArrayColumns[col.Name] = true

			// 배열을 위한 JSON 저장용 TEXT 컬럼
			columnDefs = append(columnDefs, fmt.Sprintf(
				"    %s TEXT",
				QuoteIdentifier(col.Name),
			))

			// 개별 값 저장용 컬럼들
			for i := 0; ; i++ {
				arrayColName := fmt.Sprintf("%s_%d", col.Name, i)
				// 해당 이름의 컬럼이 있는지 확인
				exists := false
				for _, checkCol := range table.Columns {
					if checkCol.Name == arrayColName {
						exists = true
						break
					}
				}
				if !exists {
					break
				}

				// 개별 컬럼 정의 추가
				columnDefs = append(columnDefs, fmt.Sprintf(
					"    %s %s",
					QuoteIdentifier(arrayColName),
					col.Type.BaseType.SQLTypeString(),
				))
			}
		} else {
			// 이미 처리된 배열의 개별 컬럼인지 확인
			isArrayColumn := false
			for baseName := range processedArrayColumns {
				if strings.HasPrefix(col.Name, baseName+"_") {
					isArrayColumn = true
					break
				}
			}
			if isArrayColumn {
				continue
			}

			// 일반 컬럼 정의
			def := fmt.Sprintf("    %s %s",
				QuoteIdentifier(col.Name),
				col.Type.SQLTypeString())

			if col.IsUnique {
				def += " UNIQUE"
			}

			// NOT NULL 제약 조건
			if strings.Contains(col.Tags, "not null") {
				def += " NOT NULL"
			}

			// 기본값 처리
			if defaultVal := extractDefaultValue(col.Tags); defaultVal != "" {
				def += fmt.Sprintf(" DEFAULT %s", defaultVal)
			}

			columnDefs = append(columnDefs, def)
		}
	}

	// 모든 컬럼 정의를 합치기
	schema.WriteString(strings.Join(columnDefs, ",\n"))
	schema.WriteString("\n);")

	// 인덱스 생성 (옵션)
	var indexDefs []string
	for _, col := range table.Columns {
		if strings.Contains(col.Tags, "index") {
			indexName := fmt.Sprintf("idx_%s_%s",
				strings.ToLower(table.Name),
				strings.ToLower(col.Name))
			indexDef := fmt.Sprintf(
				"CREATE INDEX IF NOT EXISTS %s ON %s (%s);",
				QuoteIdentifier(indexName),
				QuoteIdentifier(strings.ToLower(table.Name)),
				QuoteIdentifier(col.Name),
			)
			indexDefs = append(indexDefs, indexDef)
		}
	}

	if len(indexDefs) > 0 {
		schema.WriteString("\n\n")
		schema.WriteString(strings.Join(indexDefs, "\n"))
	}

	// SQL 파일 생성
	fileName := filepath.Join(g.OutputDir, strings.ToLower(table.Name)+".sql")
	return os.WriteFile(fileName, []byte(schema.String()), 0644)
}

func (g *Generator) generateDatabase(table Table) error {
	errBuilder := oops.In(errdoamin.Generator).With("table", table)

	db, err := gorm.Open(sqlite.Open(filepath.Join(g.OutputDir, strings.ToLower(table.Name)+".db")), &gorm.Config{})
	if err != nil {
		return errBuilder.Wrap(err)
	}

	// 스키마 실행
	schemaSQL, err := os.ReadFile(filepath.Join(g.OutputDir, strings.ToLower(table.Name)+".sql"))
	if err != nil {
		return errBuilder.Wrap(err)
	}

	if err := db.Exec(string(schemaSQL)).Error; err != nil {
		return errBuilder.Wrap(err)
	}

	// 임시 파일로 작업
	f, err := excelize.OpenFile(table.TempFile)
	if err != nil {
		return errBuilder.Wrapf(err, "failed to open Excel file")
	}
	defer f.Close()

	// 데이터 읽기
	rows, err := f.GetRows(table.SheetName)
	if err != nil {
		return oops.Wrap(err)
	}

	// 컬럼 매핑 및 파서 생성
	columnParsers := make(map[string]ValueParser)
	columnIndices := make(map[string]int)

	for i, name := range rows[0] {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		columnIndices[name] = i

		for _, col := range table.Columns {
			if col.Name == name {
				columnParsers[name] = CreateParser(col)
				break
			}
		}
	}

	// INSERT 문 준비
	insertSQL := prepareInsertSQL(table)

	// 트랜잭션 시작
	tx := db.Begin()
	if tx.Error != nil {
		return oops.Wrap(tx.Error)
	}

	// 데이터 행 처리 (4번째 행부터)
	for rowIdx := 3; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]

		if isEmptyRow(row) {
			continue
		}

		values := make([]interface{}, len(table.Columns))
		for i, col := range table.Columns {
			colIdx, exists := columnIndices[col.Name]
			if !exists || colIdx >= len(row) {
				values[i] = nil
				continue
			}

			parser := columnParsers[col.Name]
			if parser == nil {
				continue
			}

			value, err := parser.Parse(row[colIdx])
			if err != nil {
				log.Printf("Warning: Row %d: %v", rowIdx+1, err)
				values[i] = nil
			} else {
				values[i] = value.Interface()
			}
		}

		if err := tx.Exec(insertSQL, values...).Error; err != nil {
			tx.Rollback()
			return oops.Errorf("failed to insert row %d: %v", rowIdx+1, err)
		}
	}

	return tx.Commit().Error
}

func prepareInsertSQL(table Table) string {
	var cols []string
	for _, col := range table.Columns {
		cols = append(cols, QuoteIdentifier(col.Name))
	}

	placeholders := make([]string, len(table.Columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	return fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		QuoteIdentifier(strings.ToLower(table.Name)),
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
	)
}

func (g *Generator) parseSheet(f *excelize.File, sheet string, srcPath, tempPath string) error {
	rows, err := f.GetRows(sheet)
	if err != nil {
		return fmt.Errorf("failed to get rows from sheet %s: %v", sheet, err)
	}

	if len(rows) < 4 {
		return fmt.Errorf("sheet %s must have at least 4 rows", sheet)
	}

	// 컬럼 빌더 생성
	builder := NewColumnBuilder(rows[0], rows[2], rows[1]) // fieldNames, types, tags

	// 컬럼 생성
	columns, err := builder.BuildColumns()
	if err != nil {
		return fmt.Errorf("failed to build columns: %v", err)
	}

	// 테이블 생성
	table := Table{
		Name:       sheet,
		SourceFile: srcPath,
		TempFile:   tempPath,
		SheetName:  sheet,
		Columns:    columns,
	}

	g.mu.Lock()
	g.Tables = append(g.Tables, table)
	g.mu.Unlock()

	return nil
}

func (g *Generator) parseRelations(f *excelize.File) error {
	rows, err := f.GetRows("#Relation")
	if err != nil {
		return err
	}

	if len(rows) < 2 { // 헤더만 있는 경우
		return nil
	}

	// 헤더 검증
	expectedHeaders := map[string]int{
		"SourceTable":  -1,
		"TargetTable":  -1,
		"RelationType": -1,
		"ForeignKey":   -1,
		"ReferenceKey": -1,
	}

	// 헤더 위치 찾기
	for i, cell := range rows[0] {
		if _, exists := expectedHeaders[strings.TrimSpace(cell)]; exists {
			expectedHeaders[cell] = i
		}
	}

	// 모든 필수 컬럼이 있는지 확인
	for header, idx := range expectedHeaders {
		if idx == -1 {
			return fmt.Errorf("required column %s not found in #Relation sheet", header)
		}
	}

	// 관계 데이터 파싱
	for _, row := range rows[1:] {
		if len(row) < 5 || isEmptyRow(row) {
			continue
		}

		relation := Relation{
			SourceTable:  strings.TrimSpace(row[expectedHeaders["SourceTable"]]),
			TargetTable:  strings.TrimSpace(row[expectedHeaders["TargetTable"]]),
			RelationType: strings.TrimSpace(row[expectedHeaders["RelationType"]]),
			ForeignKey:   strings.TrimSpace(row[expectedHeaders["ForeignKey"]]),
			ReferenceKey: strings.TrimSpace(row[expectedHeaders["ReferenceKey"]]),
		}

		// 빈 값 검증
		if relation.SourceTable == "" || relation.TargetTable == "" || relation.RelationType == "" {
			continue
		}

		// ReferenceKey가 비어있으면 기본값 "ID" 사용
		if relation.ReferenceKey == "" {
			relation.ReferenceKey = "ID"
		}

		// ForeignKey가 비어있으면 기본값 생성 (SourceTable + ID)
		if relation.ForeignKey == "" {
			relation.ForeignKey = relation.SourceTable + "ID"
		}

		// 관계 유형 검증
		switch strings.ToLower(relation.RelationType) {
		case "hasone", "has_one", "has-one":
			relation.RelationType = "hasOne"
		case "hasmany", "has_many", "has-many":
			relation.RelationType = "hasMany"
		case "belongsto", "belongs_to", "belongs-to":
			relation.RelationType = "belongsTo"
		default:
			log.Printf("Warning: Invalid relation type %s for %s -> %s\n",
				relation.RelationType, relation.SourceTable, relation.TargetTable)
			continue
		}

		g.mu.Lock()
		g.Relations = append(g.Relations, relation)
		g.mu.Unlock()
	}

	return nil
}

// 빈 행 체크 헬퍼 함수
func isEmptyRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func (g *Generator) PrepareOutputDir() error {
	// 이미 존재하면 삭제
	if _, err := os.Stat(g.OutputDir); err == nil {
		if err := os.RemoveAll(g.OutputDir); err != nil {
			return fmt.Errorf("failed to clean output directory: %v", err)
		}
	}

	// 새로 디렉토리 생성
	if err := os.MkdirAll(g.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	return nil
}
