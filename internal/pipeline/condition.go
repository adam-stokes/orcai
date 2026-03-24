package pipeline

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// EvalCondition evaluates a condition expression against output.
// Supported expressions:
//   - "always"         → always true
//   - "contains:<str>" → true if output contains str
//   - "matches:<re>"   → true if output matches the regex
//   - "len > <n>"      → true if len(output) > n
func EvalCondition(expr, output string) bool {
	expr = strings.TrimSpace(expr)
	switch {
	case expr == "always":
		return true
	case strings.HasPrefix(expr, "contains:"):
		sub := strings.TrimPrefix(expr, "contains:")
		return strings.Contains(output, sub)
	case strings.HasPrefix(expr, "matches:"):
		pattern := strings.TrimPrefix(expr, "matches:")
		re, err := regexp.Compile(pattern)
		if err != nil {
			return false
		}
		return re.MatchString(output)
	case strings.HasPrefix(expr, "len > "):
		nStr := strings.TrimPrefix(expr, "len > ")
		n, err := strconv.Atoi(strings.TrimSpace(nStr))
		if err != nil {
			return false
		}
		return len(output) > n
	default:
		fmt.Printf("pipeline: unknown condition %q — defaulting to false\n", expr)
		return false
	}
}
