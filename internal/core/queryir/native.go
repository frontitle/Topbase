package queryir

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var optionalBlock = regexp.MustCompile(`\[\[([\s\S]*?)\]\]`)
var placeholder = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)

func ApplyNative(sql string, params []Parameter, values map[string]any) (string, []any, error) {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return "", nil, fmt.Errorf("sql is required")
	}
	byName := map[string]Parameter{}
	for _, param := range params {
		byName[param.Name] = param
	}
	sql = optionalBlock.ReplaceAllStringFunc(sql, func(block string) string {
		inner := optionalBlock.FindStringSubmatch(block)
		if len(inner) < 2 {
			return ""
		}
		matches := placeholder.FindAllStringSubmatch(inner[1], -1)
		for _, match := range matches {
			if _, ok := values[match[1]]; !ok {
				return ""
			}
		}
		return inner[1]
	})
	args := []any{}
	var err error
	sql = placeholder.ReplaceAllStringFunc(sql, func(token string) string {
		if err != nil {
			return token
		}
		name := placeholder.FindStringSubmatch(token)[1]
		value, ok := values[name]
		param := byName[name]
		if param.Type == "field" || param.Field != "" {
			field := param.Field
			if field == "" {
				field = name
			}
			if e := checkPath("native field", field); e != nil {
				err = e
				return token
			}
			frag, next, e := nativeFieldSQL(field, param.Type, value, args)
			if e != nil {
				err = e
				return token
			}
			args = next
			return frag
		}
		if !ok {
			err = fmt.Errorf("missing parameter %s", name)
			return token
		}
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	})
	if err != nil {
		return "", nil, err
	}
	return strings.TrimSpace(sql), args, nil
}

func nativeFieldSQL(field, paramType string, value any, args []any) (string, []any, error) {
	quoted := QuotePath(field)
	if value == nil {
		return "TRUE", args, nil
	}
	if strings.EqualFold(paramType, "date") {
		start, end, err := parseDateValue(value, time.Now().UTC())
		if err != nil {
			return "", args, err
		}
		parts := []string{}
		if start != "" {
			args = append(args, start)
			parts = append(parts, fmt.Sprintf("%s >= $%d", quoted, len(args)))
		}
		if end != "" {
			args = append(args, end)
			parts = append(parts, fmt.Sprintf("%s <= $%d", quoted, len(args)))
		}
		if len(parts) == 0 {
			return "TRUE", args, nil
		}
		return "(" + strings.Join(parts, " AND ") + ")", args, nil
	}
	args = append(args, value)
	return fmt.Sprintf("%s = $%d", quoted, len(args)), args, nil
}
