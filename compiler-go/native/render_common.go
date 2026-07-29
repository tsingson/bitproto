package native

import (
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
