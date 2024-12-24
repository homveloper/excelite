package exporter

import (
	"fmt"
	"strings"
)

// Tag represents a column metadata/constraint
type Tag int

const (
	TagNone          Tag = iota
	TagUnique            // 유니크 제약
	TagIndex             // 인덱스
	TagNotNull           // NULL 불가
	TagAutoIncrement     // 자동 증가
	TagPrimaryKey        // 기본키
	TagDefault           // 기본값
	TagForeignKey        // 외래키
	TagDesign            // 코드 생성 제외
	TagSize              // 크기 제한
	TagIgnore            // 특정 언어/프레임워크에서 제외
	TagReadOnly          // 읽기 전용
	TagWriteOnly         // 쓰기 전용
	TagValidate          // 검증 규칙
)

// TagInfo contains metadata about a tag
type TagInfo struct {
	Name        string            // 태그 이름
	HasValue    bool              // 값 포함 여부
	ValueType   string            // 값의 타입 (있는 경우)
	Description string            // 태그 설명
	Framework   map[string]string // 프레임워크별 매핑 정보
}

// TagValue represents a tag with its optional value
type TagValue struct {
	Tag   Tag
	Value string // Optional value
}

// FrameworkType represents supported frameworks/languages
type FrameworkType string

const (
	FrameworkGorm       FrameworkType = "gorm"
	FrameworkTypeORM    FrameworkType = "typeorm"
	FrameworkSQLAlchemy FrameworkType = "sqlalchemy"
	FrameworkEntity     FrameworkType = "entity"
)

var tagInfoMap = map[Tag]TagInfo{
	TagUnique: {
		Name:        "unique",
		Description: "Unique constraint",
		Framework: map[string]string{
			string(FrameworkGorm):       "unique",
			string(FrameworkTypeORM):    "@Unique()",
			string(FrameworkSQLAlchemy): "unique=True",
			string(FrameworkEntity):     "UNIQUE",
		},
	},
	TagIndex: {
		Name:        "index",
		Description: "Index creation",
		Framework: map[string]string{
			string(FrameworkGorm):       "index",
			string(FrameworkTypeORM):    "@Index()",
			string(FrameworkSQLAlchemy): "index=True",
			string(FrameworkEntity):     "INDEX",
		},
	},
	TagNotNull: {
		Name:        "notnull",
		Description: "Not null constraint",
		Framework: map[string]string{
			string(FrameworkGorm):       "not null",
			string(FrameworkTypeORM):    "@Column({ nullable: false })",
			string(FrameworkSQLAlchemy): "nullable=False",
			string(FrameworkEntity):     "NOT NULL",
		},
	},
	TagDefault: {
		Name:        "default",
		HasValue:    true,
		Description: "Default value",
		Framework: map[string]string{
			string(FrameworkGorm):       "default:%s",
			string(FrameworkTypeORM):    "@Column({ default: %s })",
			string(FrameworkSQLAlchemy): "default=%s",
			string(FrameworkEntity):     "DEFAULT %s",
		},
	},
	TagSize: {
		Name:        "size",
		HasValue:    true,
		ValueType:   "int",
		Description: "Size/length constraint",
		Framework: map[string]string{
			string(FrameworkGorm):       "size:%s",
			string(FrameworkTypeORM):    "@Column({ length: %s })",
			string(FrameworkSQLAlchemy): "length=%s",
			string(FrameworkEntity):     "(%s)",
		},
	},
	TagValidate: {
		Name:        "validate",
		HasValue:    true,
		Description: "Validation rules",
		Framework: map[string]string{
			string(FrameworkGorm):       "validate:%s",
			string(FrameworkTypeORM):    "@Validate(%s)",
			string(FrameworkSQLAlchemy): "validate=%s",
		},
	},
}

// GetFrameworkTag returns the framework-specific tag string
func (tv TagValue) GetFrameworkTag(framework FrameworkType) string {
	info, ok := tagInfoMap[tv.Tag]
	if !ok {
		return ""
	}

	tagFormat, ok := info.Framework[string(framework)]
	if !ok {
		return ""
	}

	if info.HasValue && tv.Value != "" {
		return fmt.Sprintf(tagFormat, tv.Value)
	}
	return tagFormat
}

// Helper functions for tag management
func ParseTag(s string) Tag {
	s = NormalizeTagString(s)
	for tag, info := range tagInfoMap {
		if info.Name == s {
			return tag
		}
	}
	return TagNone
}

func ParseTagWithValue(s string) TagValue {
	parts := strings.SplitN(s, ":", 2)
	tag := ParseTag(parts[0])

	var value string
	if len(parts) > 1 && tagInfoMap[tag].HasValue {
		value = parts[1]
	}

	return TagValue{Tag: tag, Value: value}
}

func NormalizeTagString(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}

// Column tag helper functions
func ParseColumnTags(tagStrs []string) []TagValue {
	var tags []TagValue
	for _, str := range tagStrs {
		if tv := ParseTagWithValue(str); tv.Tag != TagNone {
			tags = append(tags, tv)
		}
	}
	return tags
}

// HasTag checks if tags contain a specific tag
func HasTag(tags []TagValue, tag Tag) bool {
	for _, t := range tags {
		if t.Tag == tag {
			return true
		}
	}
	return false
}

// GetTagValue gets the value of a specific tag if it exists
func GetTagValue(tags []TagValue, tag Tag) (string, bool) {
	for _, t := range tags {
		if t.Tag == tag {
			return t.Value, true
		}
	}
	return "", false
}
