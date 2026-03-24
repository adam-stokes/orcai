package pipeline

import "strings"

// Interpolate replaces all {{key}} placeholders in s with values from vars.
// Unknown keys are left unchanged.
func Interpolate(s string, vars map[string]string) string {
	for k, v := range vars {
		s = strings.ReplaceAll(s, "{{"+k+"}}", v)
	}
	return s
}
