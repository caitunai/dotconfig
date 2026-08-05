// Package configformat renders published configuration values in formats used
// by people and machine clients while preserving configuration descriptions.
package configformat

import (
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type Format string

const (
	JSON Format = "json"
	YAML Format = "yaml"
	TOML Format = "toml"
	ENV  Format = "env"
	INI  Format = "ini"
)

var (
	ErrUnsupportedFormat = errors.New("unsupported configuration output format")
	ErrFormatConfig      = errors.New("format configuration output failed")
)

func Parse(value string) (Format, error) {
	normalized := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), ".")
	switch normalized {
	case "", string(JSON):
		return JSON, nil
	case string(YAML), "yml":
		return YAML, nil
	case string(TOML):
		return TOML, nil
	case string(ENV), "dotenv":
		return ENV, nil
	case string(INI):
		return INI, nil
	default:
		return "", ErrUnsupportedFormat
	}
}

func (format Format) ContentType() string {
	switch format {
	case YAML:
		return "application/yaml; charset=utf-8"
	case TOML:
		return "application/toml; charset=utf-8"
	case ENV, INI:
		return "text/plain; charset=utf-8"
	default:
		return "application/json; charset=utf-8"
	}
}

func (format Format) Extension() string {
	if format == "" {
		return string(JSON)
	}
	return string(format)
}

// Render emits the same logical document as the configuration-center preview:
// JSON carries descriptions in a sibling object, while text formats use
// comments so parsers still receive the configuration at their root.
func Render(config map[string]any, descriptions map[string]string, format Format) ([]byte, error) {
	switch format {
	case JSON:
		raw, err := json.MarshalIndent(map[string]any{
			"config":       config,
			"descriptions": descriptions,
		}, "", "  ")
		if err != nil {
			return nil, errors.Join(ErrFormatConfig, err)
		}
		return append(raw, '\n'), nil
	case YAML:
		return renderYAML(config, descriptions)
	case TOML:
		return renderTOML(config, descriptions)
	case ENV:
		return renderENV(config, descriptions)
	case INI:
		return renderINI(config, descriptions)
	default:
		return nil, ErrUnsupportedFormat
	}
}

func renderYAML(config map[string]any, descriptions map[string]string) ([]byte, error) {
	if len(config) == 0 {
		return []byte("{}\n"), nil
	}
	lines, err := yamlObject(config, descriptions, nil, 0)
	if err != nil {
		return nil, err
	}
	return linesBytes(lines), nil
}

func yamlObject(object map[string]any, descriptions map[string]string, path []any, indent int) ([]string, error) {
	lines := make([]string, 0, len(object))
	for _, key := range sortedKeys(object) {
		value := object[key]
		childPath := appendPath(path, key)
		lines = appendDescription(lines, descriptions[pathToPointer(childPath)], indent, "# ")
		prefix := strings.Repeat(" ", indent) + yamlKey(key) + ":"
		switch typed := value.(type) {
		case map[string]any:
			if len(typed) == 0 {
				lines = append(lines, prefix+" {}")
				continue
			}
			lines = append(lines, prefix)
			children, err := yamlObject(typed, descriptions, childPath, indent+2)
			if err != nil {
				return nil, err
			}
			lines = append(lines, children...)
		case []any:
			if len(typed) == 0 {
				lines = append(lines, prefix+" []")
				continue
			}
			lines = append(lines, prefix)
			children, err := yamlArray(typed, descriptions, childPath, indent+2)
			if err != nil {
				return nil, err
			}
			lines = append(lines, children...)
		default:
			scalar, err := scalarValue(value)
			if err != nil {
				return nil, err
			}
			lines = append(lines, prefix+" "+scalar)
		}
	}
	return lines, nil
}

func yamlArray(array []any, descriptions map[string]string, path []any, indent int) ([]string, error) {
	lines := make([]string, 0, len(array))
	for index, value := range array {
		childPath := appendPath(path, index)
		lines = appendDescription(lines, descriptions[pathToPointer(childPath)], indent, "# ")
		prefix := strings.Repeat(" ", indent) + "-"
		switch typed := value.(type) {
		case map[string]any:
			if len(typed) == 0 {
				lines = append(lines, prefix+" {}")
				continue
			}
			lines = append(lines, prefix)
			children, err := yamlObject(typed, descriptions, childPath, indent+2)
			if err != nil {
				return nil, err
			}
			lines = append(lines, children...)
		case []any:
			if len(typed) == 0 {
				lines = append(lines, prefix+" []")
				continue
			}
			lines = append(lines, prefix)
			children, err := yamlArray(typed, descriptions, childPath, indent+2)
			if err != nil {
				return nil, err
			}
			lines = append(lines, children...)
		default:
			scalar, err := scalarValue(value)
			if err != nil {
				return nil, err
			}
			lines = append(lines, prefix+" "+scalar)
		}
	}
	return lines, nil
}

func renderTOML(config map[string]any, descriptions map[string]string) ([]byte, error) {
	lines := descriptionIndex(descriptions, "# ")
	formatted, err := appendTOMLTable(lines, config, nil)
	if err != nil {
		return nil, err
	}
	return linesBytes(formatted), nil
}

func appendTOMLTable(lines []string, object map[string]any, path []string) ([]string, error) {
	if len(path) > 0 {
		lines = appendBlankLine(lines)
		parts := make([]string, len(path))
		for index, value := range path {
			parts[index] = tomlKey(value)
		}
		lines = append(lines, "["+strings.Join(parts, ".")+"]")
	}
	keys := sortedKeys(object)
	for _, key := range keys {
		if _, nested := object[key].(map[string]any); nested {
			continue
		}
		value, err := tomlValue(object[key])
		if err != nil {
			return nil, err
		}
		lines = append(lines, tomlKey(key)+" = "+value)
	}
	for _, key := range keys {
		nested, ok := object[key].(map[string]any)
		if !ok {
			continue
		}
		var err error
		lines, err = appendTOMLTable(lines, nested, appendStringPath(path, key))
		if err != nil {
			return nil, err
		}
	}
	return lines, nil
}

func tomlValue(value any) (string, error) {
	switch typed := value.(type) {
	case []any:
		values := make([]string, len(typed))
		for index, child := range typed {
			formatted, err := tomlValue(child)
			if err != nil {
				return "", err
			}
			values[index] = formatted
		}
		return "[" + strings.Join(values, ", ") + "]", nil
	case map[string]any:
		values := make([]string, 0, len(typed))
		for _, key := range sortedKeys(typed) {
			formatted, err := tomlValue(typed[key])
			if err != nil {
				return "", err
			}
			values = append(values, tomlKey(key)+" = "+formatted)
		}
		return "{ " + strings.Join(values, ", ") + " }", nil
	default:
		return scalarValue(value)
	}
}

func renderENV(config map[string]any, descriptions map[string]string) ([]byte, error) {
	lines := descriptionIndex(descriptions, "# ")
	values := flattenValues(config, nil, nil)
	for _, value := range values {
		encoded, err := envValue(value.Value)
		if err != nil {
			return nil, err
		}
		lines = append(lines, envKey(value.Path)+"="+encoded)
	}
	return linesBytes(lines), nil
}

func renderINI(config map[string]any, descriptions map[string]string) ([]byte, error) {
	lines := descriptionIndex(descriptions, "; ")
	for _, key := range sortedKeys(config) {
		if _, nested := config[key].(map[string]any); nested {
			continue
		}
		value, err := envValue(config[key])
		if err != nil {
			return nil, err
		}
		lines = append(lines, key+"="+value)
	}
	for _, section := range sortedKeys(config) {
		object, ok := config[section].(map[string]any)
		if !ok {
			continue
		}
		lines = appendBlankLine(lines)
		lines = append(lines, "["+section+"]")
		for _, value := range flattenValues(object, nil, nil) {
			encoded, err := envValue(value.Value)
			if err != nil {
				return nil, err
			}
			lines = append(lines, joinPath(value.Path, ".")+"="+encoded)
		}
	}
	return linesBytes(lines), nil
}

type flattenedValue struct {
	Value any
	Path  []any
}

func flattenValues(value any, path []any, result []flattenedValue) []flattenedValue {
	if object, ok := value.(map[string]any); ok {
		for _, key := range sortedKeys(object) {
			result = flattenValues(object[key], appendPath(path, key), result)
		}
		return result
	}
	return append(result, flattenedValue{Path: path, Value: value})
}

func envValue(value any) (string, error) {
	switch value.(type) {
	case string, []any, map[string]any:
		raw, err := json.Marshal(value)
		if err != nil {
			return "", errors.Join(ErrFormatConfig, err)
		}
		return string(raw), nil
	default:
		return scalarValue(value)
	}
}

func scalarValue(value any) (string, error) {
	switch typed := value.(type) {
	case string:
		raw, err := json.Marshal(typed)
		if err != nil {
			return "", errors.Join(ErrFormatConfig, err)
		}
		return string(raw), nil
	case json.Number:
		if _, err := typed.Int64(); err == nil {
			return typed.String(), nil
		}
		if _, err := typed.Float64(); err != nil {
			return "", errors.Join(ErrFormatConfig, err)
		}
		return typed.String(), nil
	case bool:
		return strconv.FormatBool(typed), nil
	case int:
		return strconv.Itoa(typed), nil
	case int64:
		return strconv.FormatInt(typed, 10), nil
	case float64:
		return strconv.FormatFloat(typed, 'g', -1, 64), nil
	default:
		return "", ErrFormatConfig
	}
}

func yamlKey(key string) string {
	if isBareYAMLKey(key) {
		return key
	}
	return strconv.Quote(key)
}

func tomlKey(key string) string {
	if isBareTOMLKey(key) {
		return key
	}
	return strconv.Quote(key)
}

func isBareYAMLKey(value string) bool {
	for index, current := range value {
		if current != '_' && current != '-' && !unicode.IsLetter(current) && (index == 0 || !unicode.IsDigit(current)) {
			return false
		}
	}
	return value != ""
}

func isBareTOMLKey(value string) bool {
	for _, current := range value {
		if current != '_' && current != '-' && !unicode.IsLetter(current) && !unicode.IsDigit(current) {
			return false
		}
	}
	return value != ""
}

func envKey(path []any) string {
	parts := make([]string, len(path))
	for index, segment := range path {
		var builder strings.Builder
		for _, current := range toString(segment) {
			if current >= 'a' && current <= 'z' || current >= 'A' && current <= 'Z' || current >= '0' && current <= '9' {
				builder.WriteRune(unicode.ToUpper(current))
			} else {
				builder.WriteByte('_')
			}
		}
		parts[index] = builder.String()
	}
	return strings.Join(parts, "__")
}

func descriptionIndex(descriptions map[string]string, marker string) []string {
	if len(descriptions) == 0 {
		return nil
	}
	pointers := make([]string, 0, len(descriptions))
	for pointer := range descriptions {
		pointers = append(pointers, pointer)
	}
	sort.Strings(pointers)
	lines := []string{marker + "配置项描述"}
	for _, pointer := range pointers {
		for line := range strings.SplitSeq(descriptions[pointer], "\n") {
			lines = append(lines, marker+pointer+": "+line)
		}
	}
	return append(lines, "")
}

func appendDescription(lines []string, description string, indent int, marker string) []string {
	if description == "" {
		return lines
	}
	prefix := strings.Repeat(" ", indent) + marker
	for line := range strings.SplitSeq(description, "\n") {
		lines = append(lines, prefix+line)
	}
	return lines
}

func pathToPointer(path []any) string {
	var builder strings.Builder
	for _, segment := range path {
		builder.WriteByte('/')
		value := strings.ReplaceAll(toString(segment), "~", "~0")
		builder.WriteString(strings.ReplaceAll(value, "/", "~1"))
	}
	return builder.String()
}

func joinPath(path []any, separator string) string {
	parts := make([]string, len(path))
	for index, segment := range path {
		parts[index] = toString(segment)
	}
	return strings.Join(parts, separator)
}

func toString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case int:
		return strconv.Itoa(typed)
	default:
		return ""
	}
}

func sortedKeys(object map[string]any) []string {
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func appendPath(path []any, value any) []any {
	result := make([]any, len(path), len(path)+1)
	copy(result, path)
	return append(result, value)
}

func appendStringPath(path []string, value string) []string {
	result := make([]string, len(path), len(path)+1)
	copy(result, path)
	return append(result, value)
}

func appendBlankLine(lines []string) []string {
	if len(lines) > 0 && lines[len(lines)-1] != "" {
		return append(lines, "")
	}
	return lines
}

func linesBytes(lines []string) []byte {
	return []byte(strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n")
}
