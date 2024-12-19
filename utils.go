package main

import (
	"fmt"
	"strings"
)

// SQLite 예약어 목록
var sqliteReservedWords = map[string]bool{
	"abort": true, "action": true, "add": true, "after": true, "all": true,
	"alter": true, "analyze": true, "and": true, "as": true, "asc": true,
	"attach": true, "autoincrement": true, "before": true, "begin": true,
	"between": true, "by": true, "cascade": true, "case": true, "cast": true,
	"check": true, "collate": true, "column": true, "commit": true, "conflict": true,
	"constraint": true, "create": true, "cross": true, "current": true,
	"current_date": true, "current_time": true, "current_timestamp": true,
	"database": true, "default": true, "deferrable": true, "deferred": true,
	"delete": true, "desc": true, "detach": true, "distinct": true, "drop": true,
	"each": true, "else": true, "end": true, "escape": true, "except": true,
	"exclusive": true, "exists": true, "explain": true, "fail": true, "for": true,
	"foreign": true, "from": true, "full": true, "glob": true, "group": true,
	"having": true, "if": true, "ignore": true, "immediate": true, "in": true,
	"index": true, "indexed": true, "initially": true, "inner": true, "insert": true,
	"instead": true, "intersect": true, "into": true, "is": true, "isnull": true,
	"join": true, "key": true, "left": true, "like": true, "limit": true,
	"match": true, "natural": true, "no": true, "not": true, "notnull": true,
	"null": true, "of": true, "offset": true, "on": true, "or": true, "order": true,
	"outer": true, "plan": true, "pragma": true, "primary": true, "query": true,
	"raise": true, "recursive": true, "references": true, "regexp": true,
	"reindex": true, "release": true, "rename": true, "replace": true,
	"restrict": true, "right": true, "rollback": true, "row": true, "savepoint": true,
	"select": true, "set": true, "table": true, "temp": true, "temporary": true,
	"then": true, "to": true, "transaction": true, "trigger": true, "union": true,
	"unique": true, "update": true, "using": true, "vacuum": true, "values": true,
	"view": true, "virtual": true, "when": true, "where": true, "with": true,
	"without": true,
}

// QuoteIdentifier checks if the given identifier is a reserved word and quotes it if necessary
func QuoteIdentifier(name string) string {
	// 대소문자 구분 없이 검사
	if sqliteReservedWords[strings.ToLower(name)] {
		return fmt.Sprintf(`"%s"`, name)
	}
	return name
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
