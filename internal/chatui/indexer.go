package chatui

import (
	"os"
	"path/filepath"
	"strings"
)

// IndexEntry represents a skill, agent, or prompt discovered by the global indexer.
type IndexEntry struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`   // "skill" | "agent" | "prompt"
	Source      string `json:"source"` // "global" | "project" | "project:root" | "stok" | "cli:copilot"
	Path        string `json:"path"`   // absolute file path (dir for skills, file for agents/prompts)
	Description string `json:"description"` // from frontmatter or first content line
	Inject      string `json:"inject"` // "/name" for skills, "@name " for agents, "" for prompts
}

// extractDescription pulls a description from file content.
// It checks frontmatter first (description: value between --- delimiters),
// then falls back to the first non-empty, non-heading, non-frontmatter line.
func extractDescription(content string) string {
	lines := strings.Split(content, "\n")
	inFrontmatter := false
	frontmatterDone := false
	frontmatterCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			frontmatterCount++
			if frontmatterCount == 1 {
				inFrontmatter = true
				continue
			}
			if frontmatterCount == 2 {
				inFrontmatter = false
				frontmatterDone = true
				continue
			}
		}
		if inFrontmatter {
			if k, v, ok := strings.Cut(trimmed, ":"); ok {
				if strings.TrimSpace(k) == "description" {
					desc := strings.TrimSpace(v)
					if desc != "" {
						return desc
					}
				}
			}
			continue
		}
		if !frontmatterDone {
			continue
		}
		// Past frontmatter: take first non-empty, non-heading line.
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		return trimmed
	}

	// No frontmatter: take first non-empty, non-heading line.
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || trimmed == "---" {
			continue
		}
		return trimmed
	}
	return ""
}

func readDescription(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return extractDescription(string(data))
}

// ScanIndex discovers skills and agents from well-known locations relative to
// cwd and homeDir and returns a unified index.
func ScanIndex(cwd, homeDir string) []IndexEntry {
	var entries []IndexEntry

	// ── Skills ──────────────────────────────────────────────────────────────

	type skillSource struct {
		dir    string
		source string
	}
	skillSources := []skillSource{
		{filepath.Join(homeDir, ".claude", "skills"), "global"},
		{filepath.Join(cwd, ".claude", "skills"), "project"},
		{filepath.Join(cwd, "skills"), "project"},
		{filepath.Join(homeDir, ".stok", "skills"), "stok"},
	}
	seenSkillDirs := map[string]bool{}
	for _, ss := range skillSources {
		real, err := filepath.EvalSymlinks(ss.dir)
		if err != nil {
			continue
		}
		abs, err := filepath.Abs(real)
		if err != nil {
			continue
		}
		if seenSkillDirs[abs] {
			continue
		}
		seenSkillDirs[abs] = true

		dirEntries, err := os.ReadDir(abs)
		if err != nil {
			continue
		}
		for _, e := range dirEntries {
			if !e.IsDir() {
				continue
			}
			skillDir := filepath.Join(abs, e.Name())
			skillMD := filepath.Join(skillDir, "SKILL.md")
			if _, err := os.Stat(skillMD); err != nil {
				continue
			}
			entries = append(entries, IndexEntry{
				Name:        e.Name(),
				Kind:        "skill",
				Source:      ss.source,
				Path:        skillDir,
				Description: readDescription(skillMD),
				Inject:      "/" + e.Name(),
			})
		}
	}

	// ── Agents ──────────────────────────────────────────────────────────────

	// Build skill name set for dedup.
	skillNames := map[string]bool{}
	for _, e := range entries {
		if e.Kind == "skill" {
			skillNames[e.Name] = true
		}
	}

	type agentSource struct {
		dir    string
		source string
		ext    string
	}
	agentSources := []agentSource{
		{filepath.Join(homeDir, ".claude", "commands"), "global", ".md"},
		{filepath.Join(cwd, ".claude", "commands"), "project", ".md"},
		{filepath.Join(homeDir, ".copilot", "agents"), "cli:copilot", ".yaml"},
	}
	for _, as := range agentSources {
		dirEntries, err := os.ReadDir(as.dir)
		if err != nil {
			continue
		}
		for _, e := range dirEntries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), as.ext) {
				continue
			}
			name := strings.TrimSuffix(e.Name(), as.ext)
			if skillNames[name] {
				continue
			}
			path := filepath.Join(as.dir, e.Name())
			entries = append(entries, IndexEntry{
				Name:        name,
				Kind:        "agent",
				Source:      as.source,
				Path:        path,
				Description: readDescription(path),
				Inject:      "@" + name + " ",
			})
		}
	}

	// AGENTS.md at repo root.
	if !skillNames["agents"] {
		agentsPath := filepath.Join(cwd, "AGENTS.md")
		if _, err := os.Stat(agentsPath); err == nil {
			entries = append(entries, IndexEntry{
				Name:        "agents",
				Kind:        "agent",
				Source:      "project:root",
				Path:        agentsPath,
				Description: readDescription(agentsPath),
				Inject:      "@agents ",
			})
		}
	}

	return entries
}

// ScanPrompts loads prompt entries from ~/.stok/prompts/ and {cwd}/.stok/prompts/.
func ScanPrompts(cwd, homeDir string) []IndexEntry {
	type promptSource struct {
		dir    string
		source string
	}
	sources := []promptSource{
		{filepath.Join(homeDir, ".stok", "prompts"), "global"},
		{filepath.Join(cwd, ".stok", "prompts"), "project"},
	}
	var prompts []IndexEntry
	for _, ps := range sources {
		dirEntries, err := os.ReadDir(ps.dir)
		if err != nil {
			continue
		}
		for _, e := range dirEntries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			name := strings.TrimSuffix(e.Name(), ".md")
			path := filepath.Join(ps.dir, e.Name())
			prompts = append(prompts, IndexEntry{
				Name:        name,
				Kind:        "prompt",
				Source:      ps.source,
				Path:        path,
				Description: readDescription(path),
				Inject:      "",
			})
		}
	}
	return prompts
}

// IndexByKind returns entries filtered by kind.
func IndexByKind(index []IndexEntry, kind string) []IndexEntry {
	var out []IndexEntry
	for _, e := range index {
		if e.Kind == kind {
			out = append(out, e)
		}
	}
	return out
}

// SourceLabel returns a short label for a source value.
func SourceLabel(source string) string {
	switch source {
	case "global":
		return "[global]"
	case "project":
		return "[project]"
	case "project:root":
		return "[project]"
	case "stok":
		return "[stok]"
	default:
		return "[" + source + "]"
	}
}
