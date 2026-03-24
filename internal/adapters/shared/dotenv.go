package shared

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// LoadDotEnv parses a .env file in dir and returns a map of key→value pairs.
// Lines starting with # and empty lines are ignored. Values may optionally be
// quoted with single or double quotes, which are stripped. Missing or
// unreadable .env files return an empty map without error.
func LoadDotEnv(dir string) map[string]string {
	out := make(map[string]string)
	f, err := os.Open(filepath.Join(dir, ".env"))
	if err != nil {
		return out
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		// Strip surrounding quotes.
		if len(v) >= 2 && ((v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'')) {
			v = v[1 : len(v)-1]
		}
		out[k] = v
	}
	return out
}
