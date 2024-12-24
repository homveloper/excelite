package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"excelite/exporter"
)

// go run main.go -inputdir=./data -output=./generated -lang="go,nodejs" -package=models
// go run main.go -inputfiles=game_data.xlsx -output=./generated -lang="all" -package=models
func main() {
	// CLI 플래그 정의
	inputDir := flag.String("inputdir", "", "Directory containing Excel files")
	inputFiles := flag.String("inputfiles", "", "Comma-separated list of Excel files")
	outputDir := flag.String("output", "generated", "Output directory for generated files")
	languages := flag.String("lang", "all", "Comma-separated list of target languages (go,cpp,nodejs,all)")
	packageName := flag.String("package", "models", "Package name for generated code")
	flag.Parse()

	if *inputDir == "" && *inputFiles == "" {
		log.Fatal("Either -inputdir or -inputfiles must be provided")
	}

	printBanner()

	// Excel 파일 목록 수집
	var excelFiles []string
	if *inputDir != "" {
		files, err := collectExcelFiles(*inputDir)
		if err != nil {
			log.Fatalf("Failed to collect Excel files: %v", err)
		}
		excelFiles = files
	} else {
		excelFiles = strings.Split(*inputFiles, ",")
	}

	// Excel 파일들을 파싱하여 테이블 정의 수집
	var allTables []exporter.Table
	for _, file := range excelFiles {
		tables, err := exporter.ParseExcelFile(file)
		if err != nil {
			log.Printf("Warning: Failed to parse %s: %v", file, err)
			continue
		}
		allTables = append(allTables, tables...)
	}

	// Registry에 exporter들 등록
	registry := exporter.NewRegistry()

	// Go exporter 등록
	// registry.Register("go", exporter.NewGORMExporter, exporter.Options{
	// 	PackageName: *packageName,
	// 	ExtraOptions: map[string]interface{}{
	// 		"useGorm":      true,
	// 		"useSQLite":    true,
	// 		"generateRepo": true,
	// 	},
	// })

	// sqlite exporter 등록
	registry.Register("sqlite", exporter.NewSQLiteExporter, exporter.Options{
		PackageName: *packageName,
	})

	// // Node.js exporter 등록
	// registry.Register("nodejs", exporter.NewNodeJSExporter, exporter.Options{
	// 	PackageName: *packageName,
	// 	ExtraOptions: map[string]interface{}{
	// 		"useTypeScript": true,
	// 		"useTypeORM":    true,
	// 	},
	// })

	// 요청된 언어들로 export
	requestedLangs := []string{}
	if *languages == "all" {
		requestedLangs = registry.Languages()
	} else {
		requestedLangs = strings.Split(*languages, ",")
	}

	// 각 언어별로 Export 실행
	for _, lang := range requestedLangs {
		opts := exporter.Options{
			OutputDir:   filepath.Join(*outputDir, lang),
			PackageName: *packageName,
			DBDriver:    "sqlite",
			DBName:      "app.db",
		}

		if err := registry.Export(lang, allTables, opts); err != nil {
			log.Printf("Failed to export %s code: %v", lang, err)
			continue
		}
		log.Printf("Successfully exported %s code", lang)
	}
}

// Excel 파일 수집 함수
func collectExcelFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			// Excel 파일 확장자 확인 (.xlsx, .xls)
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".xlsx" || ext == ".xls" {
				// 임시 파일 제외 (~$로 시작하는 파일)
				if !strings.HasPrefix(filepath.Base(path), "~$") {
					files = append(files, path)
				}
			}
		}
		return nil
	})

	return files, err
}

func printBanner() {
	banner := `
███████╗██╗  ██╗ ██████╗███████╗██╗     ██╗████████╗███████╗
██╔════╝╚██╗██╔╝██╔════╝██╔════╝██║     ██║╚══██╔══╝██╔════╝
█████╗   ╚███╔╝ ██║     █████╗  ██║     ██║   ██║   █████╗  
██╔══╝   ██╔██╗ ██║     ██╔══╝  ██║     ██║   ██║   ██╔══╝  
███████╗██╔╝ ██╗╚██████╗███████╗███████╗██║   ██║   ███████╗
╚══════╝╚═╝  ╚═╝ ╚═════╝╚══════╝╚══════╝╚═╝   ╚═╝   ╚══════╝
                                                  v0.0.1
Excel to Code & DB Generator
    `
	log.Println(banner)
}
