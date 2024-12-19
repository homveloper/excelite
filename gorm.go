package main

import (
	"fmt"
	"strings"
)

// Tag 정보를 파싱하고 GORM 태그로 변환하는 함수
func parseTagToGORMTag(tagStr string) (string, map[string]string) {
	tags := make(map[string]string)
	var gormTags []string

	// 쉼표로 구분된 태그들을 처리
	for _, tag := range strings.Split(tagStr, ",") {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}

		// key=value 형태의 태그 처리
		if strings.Contains(tag, "=") {
			parts := strings.SplitN(tag, "=", 2)
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			tags[key] = value

			// GORM 태그로 변환
			switch key {
			case TagSize:
				gormTags = append(gormTags, fmt.Sprintf("size:%s", value))
			case TagDefault:
				gormTags = append(gormTags, fmt.Sprintf("default:%s", value))
			case TagForeignKey:
				gormTags = append(gormTags, fmt.Sprintf("foreignKey:%s", value))
			}
		} else {
			// 단일 태그 처리
			tags[tag] = "true"

			// GORM 태그로 변환
			switch tag {
			case TagPrimaryKey:
				gormTags = append(gormTags, "primaryKey")
			case TagUnique:
				gormTags = append(gormTags, "unique")
			case TagIndex:
				gormTags = append(gormTags, "index")
			case TagNotNull:
				gormTags = append(gormTags, "not null")
			case TagAutoInc:
				gormTags = append(gormTags, "autoIncrement")
			}
		}
	}

	gormTag := ""
	if len(gormTags) > 0 {
		gormTag = fmt.Sprintf(`gorm:"%s"`, strings.Join(gormTags, ";"))
	}

	return gormTag, tags
}
