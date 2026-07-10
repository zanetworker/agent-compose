package compose

import "strings"

// ExpandEnvMapping expands an N-var template map. Each value in
// mapping is either a literal (used as-is) or a template like
// "${endpoint}" or "${model.opus}". Templates are resolved from vars.
// If a template references a var not present in vars, or the var's
// value is empty, that entry is omitted from the result.
func ExpandEnvMapping(mapping map[string]string, vars map[string]string) map[string]string {
	result := make(map[string]string, len(mapping))
	for envVar, tmpl := range mapping {
		val := expandOne(tmpl, vars)
		if val != "" {
			result[envVar] = val
		}
	}
	return result
}

func expandOne(tmpl string, vars map[string]string) string {
	tmpl = strings.TrimSpace(tmpl)
	if strings.HasPrefix(tmpl, "${") && strings.HasSuffix(tmpl, "}") {
		varName := tmpl[2 : len(tmpl)-1]
		return vars[varName]
	}
	return tmpl
}
