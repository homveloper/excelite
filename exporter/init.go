// exporter/init.go
package exporter

// init 함수는 패키지가 로드될 때 기본 Exporter들을 등록합니다.
func init() {
	// Go Exporter 등록
	Register("go", func() Exporter {
		return NewGORMExporter()
	}, Options{
		PackageName: "models",
		ExtraOptions: map[string]interface{}{
			"useGorm":      true,
			"useSQLite":    true,
			"generateRepo": true,
		},
	})

	// // C++ Exporter 등록
	// Register("cpp", func() Exporter {
	// 	return NewCppExporter()
	// }, Options{
	// 	ExtraOptions: map[string]interface{}{
	// 		"useSQLite":      true,
	// 		"usePointers":    true,
	// 		"headerGuards":   true,
	// 		"namespaceStyle": "nested", // nested or flat
	// 	},
	// })

	// // Node.js Exporter 등록
	// Register("nodejs", func() Exporter {
	// 	return NewNodeJSExporter()
	// }, Options{
	// 	PackageName: "models",
	// 	ExtraOptions: map[string]interface{}{
	// 		"useTypeORM":         true,
	// 		"useTypeScript":      true,
	// 		"generateMigrations": true,
	// 		"decoratorStyle":     "experimental", // experimental or legacy
	// 	},
	// })
}

// 언어별 기능을 쉽게 켜고 끌 수 있는 옵션 상수들
const (
	// Go options
	OptGoUseGorm      = "useGorm"
	OptGoUseSQLite    = "useSQLite"
	OptGoGenerateRepo = "generateRepo"

	// C++ options
	OptCppUseSQLite    = "useSQLite"
	OptCppUsePointers  = "usePointers"
	OptCppHeaderGuards = "headerGuards"

	// Node.js options
	OptNodeUseTypeORM = "useTypeORM"
	OptNodeTypeScript = "useTypeScript"
	OptNodeMigrations = "generateMigrations"
)

// GenerateAll은 모든 지원 언어에 대해 코드를 생성합니다.
func GenerateAll(tables []Table, baseOpts Options) map[string]error {
	results := make(map[string]error)

	for _, lang := range DefaultRegistry.Languages() {
		// 각 언어별 기본 옵션 가져오기
		langOpts, err := DefaultRegistry.GetOptions(lang)
		if err != nil {
			results[lang] = err
			continue
		}

		// 기본 옵션과 사용자 옵션 병합
		opts := mergeOptions(langOpts, baseOpts)

		// 해당 언어로 생성 시도
		if err := Export(lang, tables, opts); err != nil {
			results[lang] = err
		}
	}

	return results
}
