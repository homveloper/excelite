package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/schollz/progressbar/v3"
	"golang.org/x/sync/errgroup"
)

// go run main.go -inputdir=../../Content/Data -output=./generated
func main() {
	// CLI 플래그 정의
	inputDir := flag.String("inputdir", "", "Directory containing Excel files")
	inputFiles := flag.String("inputfiles", "", "Comma-separated list of Excel files")
	outputDir := flag.String("output", "generated", "Output directory for generated files")
	workers := flag.Int("workers", 4, "Number of parallel workers")
	flag.Parse()

	if *inputDir == "" && *inputFiles == "" {
		log.Fatal("Please provide either -input-dir or -input-files flag")
	}

	// 생성기 초기화

	gen := NewGenerator(*outputDir)

	if err := gen.PrepareOutputDir(); err != nil {
		log.Fatal(err)
	}

	printBanner()

	// 파일 목록 수집
	var excelFiles []string
	if *inputDir != "" {
		files, err := collectExcelFiles(*inputDir)
		if err != nil {
			log.Fatal(err)
		}
		excelFiles = files
	} else {
		excelFiles = strings.Split(*inputFiles, ",")
	}

	// 진행바 초기화
	bar := progressbar.Default(int64(len(excelFiles)))

	// 병렬 처리를 위한 error group 생성
	g := new(errgroup.Group)
	fileChan := make(chan string)

	// 워커 생성
	for i := 0; i < *workers; i++ {
		workerID := i
		g.Go(func() error {
			return gen.Worker(workerID, fileChan, bar)
		})
	}

	// 파일들을 채널에 전송
	go func() {
		for _, file := range excelFiles {
			fileChan <- file
		}
		close(fileChan)
	}()

	// 모든 워커가 완료될 때까지 대기
	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}

	// 출력 디렉토리 생성
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatal(err)
	}

	// 파일 생성
	if err := gen.GenerateFiles(); err != nil {
		log.Fatal(err)
	}

	log.Println("\nGeneration completed successfully! 🚀")
}

func printBanner() {
	banner := `
    ███████╗ ██╗   ██╗  █████╗  ███████╗  █████╗  ██████╗ 
    ██╔═══██╗██║   ██║ ██╔══██╗ ██╔════╝ ██╔══██╗ ██╔══██╗
    ██║   ██║██║   ██║ ███████║ ███████╗ ███████║ ██████╔╝
    ██║▄▄ ██║██║   ██║ ██╔══██║ ╚════██║ ██╔══██║ ██╔══██╗
    ╚██████╔╝╚██████╔╝ ██║  ██║ ███████║ ██║  ██║ ██║  ██║
     ╚══▀▀═╝  ╚═════╝  ╚═╝  ╚═╝ ╚══════╝ ╚═╝  ╚═╝ ╚═╝  ╚═╝
                                                v1.0.0
    Excel to Code & DB Generator
    `
	log.Println(banner)
}

func collectExcelFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(info.Name(), ".xlsx") || strings.HasSuffix(info.Name(), ".xls")) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}
