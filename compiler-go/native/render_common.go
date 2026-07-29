package native

import (
	"fmt"
	"strings"
	"unicode"
)

func toPascal(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		if runes[i] == '.' || runes[i] == '-' {
			runes[i] = '_'
		}
	}
	parts := strings.Split(string(runes), "_")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		pr := []rune(p)
		pr[0] = unicode.ToUpper(pr[0])
		out = append(out, string(pr))
	}
	return strings.Join(out, "")
}

func splitIdentifier(s string) []string {
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.ReplaceAll(s, "-", "_")
	return strings.Split(s, "_")
}

func isFixedFloatType(s string) bool {
	return s == "float" || s == "double"
}

func fixedFloatKindSuffix(kind string) string {
	if kind == "double" {
		return "Double"
	}
	return "Float"
}

func (i *index) fixedFloatKindForType(t TypeExpr) (string, TypeExpr, bool, error) {
	resolved, err := i.resolveAliasType(t)
	if err != nil {
		return "", TypeExpr{}, false, err
	}
	if isFixedFloatType(resolved.Name) {
		return resolved.Name, resolved, true, nil
	}
	return "", resolved, false, nil
}

// toSnakeCase converts a CamelCase or mixed identifier to snake_case.
func toSnakeCase(s string) string {
	s = strings.ReplaceAll(s, "-", "_")
	var out []rune
	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 && (unicode.IsLower(runes[i-1]) ||
				(i+1 < len(runes) && unicode.IsLower(runes[i+1]))) {
				out = append(out, '_')
			}
			out = append(out, unicode.ToLower(r))
		} else {
			out = append(out, r)
		}
	}
	result := string(out)
	for strings.Contains(result, "__") {
		result = strings.ReplaceAll(result, "__", "_")
	}
	return strings.Trim(result, "_")
}

func msgJsonFormatterName(name string) string {
	return "BpNativeJsonFormat" + toPascal(name)
}

func aliasJsonFormatterName(name string) string {
	return "BpNativeJsonFormat" + toPascal(name)
}

func arrayJsonFormatterNameForAlias(name string) string {
	return "BpNativeJsonFormatArray" + toPascal(name)
}

func arrayJsonFormatterNameForField(msg string, fieldNum int) string {
	return fmt.Sprintf("BpNativeJsonFormatArray%s%d", toPascal(msg), fieldNum)
}
