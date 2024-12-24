// exporter/registry.go
package exporter

import (
	"fmt"
	"sync"
)

// Registry는 모든 exporter들을 관리하는 중앙 레지스트리입니다.
type Registry struct {
	mu        sync.RWMutex
	factories map[string]FactoryFunc
	options   map[string]Options
}

// NewRegistry는 새로운 Registry 인스턴스를 생성합니다.
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]FactoryFunc),
		options:   make(map[string]Options),
	}
}

// Register는 새로운 exporter factory를 등록합니다.
func (r *Registry) Register(lang string, factory FactoryFunc, defaultOpts Options) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.factories[lang] = factory
	r.options[lang] = defaultOpts
}

// Get은 지정된 언어의 exporter 인스턴스를 반환합니다.
func (r *Registry) Get(lang string) (Exporter, error) {
	r.mu.RLock()
	factory, exists := r.factories[lang]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no exporter registered for language: %s", lang)
	}

	return factory(), nil
}

// GetOptions는 지정된 언어의 기본 옵션을 반환합니다.
func (r *Registry) GetOptions(lang string) (Options, error) {
	r.mu.RLock()
	opts, exists := r.options[lang]
	r.mu.RUnlock()

	if !exists {
		return Options{}, fmt.Errorf("no options found for language: %s", lang)
	}

	return opts, nil
}

// Languages는 지원되는 모든 언어 목록을 반환합니다.
func (r *Registry) Languages() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	langs := make([]string, 0, len(r.factories))
	for lang := range r.factories {
		langs = append(langs, lang)
	}
	return langs
}

// ExportTables는 지정된 언어로 테이블들을 내보냅니다.
func (r *Registry) Export(lang string, tables []Table, opts Options) error {
	exp, err := r.Get(lang)
	if err != nil {
		return err
	}

	// 기본 옵션과 사용자 옵션을 병합
	defaultOpts, _ := r.GetOptions(lang)
	mergedOpts := mergeOptions(defaultOpts, opts)

	return exp.Export(tables, mergedOpts)
}

// 옵션 병합을 위한 헬퍼 함수
func mergeOptions(defaultOpts, userOpts Options) Options {
	result := defaultOpts

	// 사용자 지정 값이 있는 경우 덮어쓰기
	if userOpts.OutputDir != "" {
		result.OutputDir = userOpts.OutputDir
	}
	if userOpts.PackageName != "" {
		result.PackageName = userOpts.PackageName
	}
	if userOpts.TemplateDir != "" {
		result.TemplateDir = userOpts.TemplateDir
	}
	if userOpts.DBDriver != "" {
		result.DBDriver = userOpts.DBDriver
	}
	if userOpts.DBName != "" {
		result.DBName = userOpts.DBName
	}

	// ExtraOptions 병합
	if result.ExtraOptions == nil {
		result.ExtraOptions = make(map[string]interface{})
	}
	for k, v := range userOpts.ExtraOptions {
		result.ExtraOptions[k] = v
	}

	return result
}

// DefaultRegistry는 전역 레지스트리 인스턴스입니다.
var DefaultRegistry = NewRegistry()

// Register는 기본 레지스트리에 exporter를 등록합니다.
func Register(lang string, factory FactoryFunc, defaultOpts Options) {
	DefaultRegistry.Register(lang, factory, defaultOpts)
}

// Get은 기본 레지스트리에서 exporter를 가져옵니다.
func Get(lang string) (Exporter, error) {
	return DefaultRegistry.Get(lang)
}

// ExportTables는 기본 레지스트리를 사용하여 테이블들을 내보냅니다.
func Export(lang string, tables []Table, opts Options) error {
	return DefaultRegistry.Export(lang, tables, opts)
}
