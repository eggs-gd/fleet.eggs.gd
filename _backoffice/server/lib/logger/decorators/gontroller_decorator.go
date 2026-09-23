package decorators

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	l "github.com/eggs-gd/core.eggs.gd/lib/logger"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

const (
	TreeBranch     = "├─"
	TreeLastBranch = "└─"
	TreePipe       = "│ "
)

type GontrollerDecorator struct{}

func (c *GontrollerDecorator) Decorate(buf *buffer.Buffer, fields []zapcore.Field) *buffer.Buffer {
	fields = cleanupFields(fields)
	if len(fields) == 0 {
		return buf
	}

	if hasField(fields, "sql") {
		buf = c.decorateSQL(buf, fields)
	} else {
		buf = c.decorateDefault(buf, fields)
	}

	if hasField(fields, "error") {
		buf = c.decorateError(buf, fields)
	}

	buf.AppendString("\n")
	return buf
}

func (c *GontrollerDecorator) decorateSQL(buf *buffer.Buffer, fields []zapcore.Field) *buffer.Buffer {
	var elapsed, rows, sql string

	for _, field := range fields {
		switch field.Key {
		case "elapsed":
			if field.Integer != 0 {
				elapsed = time.Duration(field.Integer).String()
			}
		case "rows":
			if field.Integer != 0 {
				rows = fmt.Sprintf("%d", field.Integer)
			}
		case "sql":
			if field.String != "" {
				sql = field.String
			}
		}
	}

	if rows != "" || elapsed != "" {
		buf.AppendString(fmt.Sprintf("%s \033[36m%v rows in %v\033[0m\n", TreeBranch, rows, elapsed))
	}
	if sql != "" {
		buf.AppendString(fmt.Sprintf("%s %v\n", TreeLastBranch, formatSQL(sql)))
	}
	return buf
}

func (c *GontrollerDecorator) decorateError(buf *buffer.Buffer, fields []zapcore.Field) *buffer.Buffer {
	for _, field := range fields {
		if field.Key == "error" {
			buf.AppendString(fmt.Sprintf("\033[31m╳ %v\033[0m\n", field.Interface))
			return buf
		}
	}
	return buf
}

func (c *GontrollerDecorator) decorateDefault(buf *buffer.Buffer, fields []zapcore.Field) *buffer.Buffer {
	lastIdx := len(fields) - 1

	for i, field := range fields {
		if field.Key == "error" {
			continue
		}
		value := formatFieldValue(field)
		if value != "" {
			prefix := TreeBranch
			if i == lastIdx {
				prefix = TreeLastBranch
			}
			buf.AppendString(fmt.Sprintf("%s %s: %s\n", prefix, field.Key, value))
		}
	}
	return buf
}

func formatFieldValue(field zapcore.Field) string {
	switch {
	case field.String != "":
		return field.String
	case field.Integer != 0:
		return fmt.Sprintf("%d", field.Integer)
	case field.Interface != nil:
		return fmt.Sprintf("%v", field.Interface)
	}
	return ""
}

func cleanupFields(fields []zapcore.Field) []zapcore.Field {
	validFields := []zapcore.Field{}
	for _, field := range fields {
		if field.Key != "" {
			validFields = append(validFields, field)
		}
	}
	return validFields
}

func hasField(fields []zapcore.Field, key string) bool {
	for _, field := range fields {
		if field.Key == key {
			return true
		}
	}
	return false
}

func formatSQL(sql string) string {
	sql = strings.TrimSpace(sql)
	sql = strings.Join(strings.Fields(sql), " ")

	// Highlight SQL keywords
	keywords := []string{
		"SELECT", "FROM", "WHERE", "INSERT", "INTO", "VALUES", "UPDATE", "SET", "DELETE",
		"JOIN", "LEFT", "RIGHT", "INNER", "OUTER", "GROUP BY",
		"ORDER BY", "HAVING", "LIMIT", "OFFSET", "AND", "OR",
		"IN", "NOT", "NULL", "IS", "AS", "ON", "CONFLICT", "SET", "DO", "RETURNING",
	}

	for _, keyword := range keywords {
		pattern := fmt.Sprintf(`(?i)\b%s\b`, keyword)
		sql = regexp.MustCompile(pattern).ReplaceAllString(sql, l.ColorBrightCyan+keyword+l.ColorReset)
	}

	// Highlight string values
	sql = regexp.MustCompile(`'[^']*'`).ReplaceAllStringFunc(sql, func(s string) string {
		return l.ColorGreen + s + l.ColorReset
	})

	// Highlight numeric values
	sql = regexp.MustCompile(`,\b(\d+\.?\d*)\b,`).ReplaceAllString(sql, ","+l.ColorYellow+"${1}"+l.ColorReset+",")

	// Highlight GUIDs
	sql = regexp.MustCompile(`"[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}"`).ReplaceAllStringFunc(sql, func(s string) string {
		return l.ColorBrightMagenta + s + l.ColorReset
	})

	return sql
}
