package ollama

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// LocalModel represents a model returned by the Ollama API.
type LocalModel struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type tagsResponse struct {
	Models []LocalModel `json:"models"`
}

// IsAvailable returns true if the Ollama API is reachable.
func IsAvailable() bool {
	resp, err := http.Get("http://localhost:11434/api/tags")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}

// ListModels returns all locally available Ollama models.
func ListModels() ([]LocalModel, error) {
	resp, err := http.Get("http://localhost:11434/api/tags")
	if err != nil {
		return nil, fmt.Errorf("ollama unavailable: %w", err)
	}
	defer resp.Body.Close()

	var r tagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return r.Models, nil
}

// IsCtxVariant returns true if name matches the <base>-ctx<N>k:<tag> pattern.
func IsCtxVariant(name string) bool {
	base, _, _ := splitName(name)
	idx := strings.LastIndex(base, "-ctx")
	if idx == -1 {
		return false
	}
	suffix := base[idx+4:]
	return strings.HasSuffix(suffix, "k") && len(suffix) > 1
}

// CtxVariantName builds the extended-context model name for baseModel at ctxK * 1024 tokens.
func CtxVariantName(baseModel string, ctxK int) string {
	base, tag, _ := splitName(baseModel)
	// Strip any existing ctx suffix so we don't double-nest.
	if idx := strings.LastIndex(base, "-ctx"); idx != -1 {
		base = base[:idx]
	}
	return fmt.Sprintf("%s-ctx%dk:%s", base, ctxK, tag)
}

// BuildCtxModel creates an Ollama model derived from baseModel with num_ctx = ctxK * 1024.
func BuildCtxModel(baseModel string, ctxK int) error {
	variantName := CtxVariantName(baseModel, ctxK)
	content := fmt.Sprintf("FROM %s\nPARAMETER num_ctx %d\n", baseModel, ctxK*1024)

	tmp, err := os.CreateTemp("", "Modelfile-*")
	if err != nil {
		return fmt.Errorf("create modelfile: %w", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()

	out, err := exec.Command("ollama", "create", variantName, "-f", tmp.Name()).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ollama create: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// UpdateOpencodeConfig adds modelName to ~/.config/opencode/opencode.json.
func UpdateOpencodeConfig(modelName string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	cfgPath := filepath.Join(home, ".config", "opencode", "opencode.json")

	var cfg map[string]interface{}

	data, err := os.ReadFile(cfgPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("parse opencode config: %w", err)
		}
	} else {
		cfg = map[string]interface{}{
			"$schema": "https://opencode.ai/config.json",
		}
	}

	provider := mapAt(cfg, "provider")
	ollama := mapAt(provider, "ollama")
	if ollama["name"] == nil {
		ollama["name"] = "Ollama (local)"
		ollama["npm"] = "@ai-sdk/openai-compatible"
		ollama["options"] = map[string]interface{}{"baseURL": "http://localhost:11434/v1"}
		provider["ollama"] = ollama
		cfg["provider"] = provider
	}
	models := mapAt(ollama, "models")
	models[modelName] = map[string]interface{}{"name": modelName}
	ollama["models"] = models

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfgPath, out, 0o644)
}

// splitName splits "name:tag" into (name, tag, true). Tag defaults to "latest".
func splitName(model string) (base, tag string, ok bool) {
	parts := strings.SplitN(model, ":", 2)
	base = parts[0]
	tag = "latest"
	if len(parts) == 2 {
		tag = parts[1]
		ok = true
	}
	return
}

// mapAt retrieves or creates a nested map[string]interface{} key.
func mapAt(parent map[string]interface{}, key string) map[string]interface{} {
	if v, ok := parent[key].(map[string]interface{}); ok {
		return v
	}
	m := map[string]interface{}{}
	parent[key] = m
	return m
}
