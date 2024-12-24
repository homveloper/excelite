// exporter/exporter.go
package exporter

import (
	"fmt"
	"os"
)

// BaseExporter provides common functionality for exporters
type BaseExporter struct {
	language string
}

// NewBaseExporter creates a new base exporter
func NewBaseExporter(lang string) BaseExporter {
	return BaseExporter{language: lang}
}

// Language는 지원하는 언어를 반환합니다.
func (b BaseExporter) Language() string {
	return b.language
}

// ParseExcelFiles는 여러 엑셀 파일을 파싱합니다.
func (b BaseExporter) ParseExcelFiles(files []string) ([]Table, error) {
	var allTables []Table
	for _, file := range files {
		tables, err := ParseExcelFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %v", file, err)
		}
		allTables = append(allTables, tables...)
	}
	return allTables, nil
}

// EnsureOutputDir는 출력 디렉토리를 생성합니다.
func (b BaseExporter) EnsureOutputDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

// GetBoolOption은 ExtraOptions에서 bool 값을 가져옵니다.
func (b BaseExporter) GetBoolOption(opts Options, key string, defaultValue bool) bool {
	if val, ok := opts.ExtraOptions[key].(bool); ok {
		return val
	}
	return defaultValue
}

// GetStringOption은 ExtraOptions에서 string 값을 가져옵니다.
func (b BaseExporter) GetStringOption(opts Options, key string, defaultValue string) string {
	if val, ok := opts.ExtraOptions[key].(string); ok {
		return val
	}
	return defaultValue
}
