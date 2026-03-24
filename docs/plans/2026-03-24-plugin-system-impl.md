# Plugin System & Prompt Builder Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a two-tier plugin system (native go-plugin + CLI wrappers), a pipeline engine (YAML-defined, composable, event-bus-driven), and a charm BBS-style prompt builder TUI that saves pipelines as discoverable plugins.

**Architecture:** Every CLI tool becomes a plugin via hashicorp/go-plugin (Tier 1 native) or a CliAdapter subprocess wrapper (Tier 2). Pipelines are YAML files interpreted at runtime and registered as first-class plugins via the existing discovery system. A bubbletea 80% modal lets users compose, branch, and save pipelines interactively.

**Tech Stack:** Go 1.25, hashicorp/go-plugin v1.7.0, gRPC/protobuf, charmbracelet/bubbletea + bubbles + lipgloss, gopkg.in/yaml.v3 (new dep), existing internal/bus event bus.

---

## Phase 1: Proto Extensions

### Task 1: Extend plugin.proto with Execute and Capabilities RPCs

**Files:**
- Modify: `proto/orcai/v1/plugin.proto`
- Regenerate: `proto/orcai/v1/plugin.pb.go`, `proto/orcai/v1/plugin_grpc.pb.go`

**Step 1: Add messages and RPCs to plugin.proto**

Open `proto/orcai/v1/plugin.proto` and append after the existing `StatusResponse` message:

```protobuf
message ExecuteRequest {
  string input            = 1;
  map<string, string> vars = 2;
}

message ExecuteResponse {
  string chunk = 1;
  bool   done  = 2;
  string error = 3;
}

message Capability {
  string name          = 1;
  string input_schema  = 2;
  string output_schema = 3;
}

message CapabilityList {
  repeated Capability items = 1;
}
```

Add two RPCs to the `OrcaiPlugin` service:

```protobuf
rpc Execute(ExecuteRequest)  returns (stream ExecuteResponse);
rpc Capabilities(Empty)      returns (CapabilityList);
```

**Step 2: Regenerate protobuf code**

```bash
make proto
```

Expected: no errors, `plugin.pb.go` and `plugin_grpc.pb.go` updated with new types.

**Step 3: Verify build still compiles**

```bash
go build ./...
```

Expected: PASS (new RPCs have generated `Unimplemented` stubs automatically).

**Step 4: Commit**

```bash
git add proto/orcai/v1/plugin.proto proto/orcai/v1/plugin.pb.go proto/orcai/v1/plugin_grpc.pb.go
git commit -m "feat(proto): add Execute and Capabilities RPCs to OrcaiPlugin service"
```

---

## Phase 2: Universal Plugin Interface

### Task 2: Create internal/plugin package with Plugin interface and Manager

**Files:**
- Create: `internal/plugin/plugin.go`
- Create: `internal/plugin/manager.go`
- Create: `internal/plugin/plugin_test.go`

**Step 1: Write the failing test**

Create `internal/plugin/plugin_test.go`:

```go
package plugin_test

import (
	"testing"

	"github.com/adam-stokes/orcai/internal/plugin"
)

func TestManager_Empty(t *testing.T) {
	m := plugin.NewManager()
	if len(m.List()) != 0 {
		t.Errorf("expected empty manager, got %d plugins", len(m.List()))
	}
}

func TestManager_Register(t *testing.T) {
	m := plugin.NewManager()
	p := &plugin.StubPlugin{PluginName: "test"}
	m.Register(p)
	plugins := m.List()
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0].Name() != "test" {
		t.Errorf("expected name 'test', got %q", plugins[0].Name())
	}
}

func TestManager_Get(t *testing.T) {
	m := plugin.NewManager()
	m.Register(&plugin.StubPlugin{PluginName: "alpha"})
	m.Register(&plugin.StubPlugin{PluginName: "beta"})

	p, ok := m.Get("alpha")
	if !ok {
		t.Fatal("expected to find 'alpha'")
	}
	if p.Name() != "alpha" {
		t.Errorf("got wrong plugin: %q", p.Name())
	}

	_, ok = m.Get("missing")
	if ok {
		t.Error("expected not found for 'missing'")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/plugin/... -v
```

Expected: FAIL — `plugin` package does not exist yet.

**Step 3: Create internal/plugin/plugin.go**

```go
package plugin

import (
	"context"
	"io"
)

// Plugin is the universal interface all orcai plugins implement,
// regardless of whether they are native go-plugins (Tier 1) or CLI wrappers (Tier 2).
type Plugin interface {
	// Name returns the unique plugin identifier.
	Name() string
	// Description returns a human-readable summary.
	Description() string
	// Capabilities returns what this plugin can do.
	Capabilities() []Capability
	// Execute runs the plugin with the given input and template vars.
	// Output is streamed to w. Returns when the plugin finishes or ctx is cancelled.
	Execute(ctx context.Context, input string, vars map[string]string, w io.Writer) error
	// Close releases any resources held by the plugin.
	Close() error
}

// Capability describes one thing a plugin can do.
type Capability struct {
	Name         string
	InputSchema  string
	OutputSchema string
}

// StubPlugin is a test double that satisfies the Plugin interface.
type StubPlugin struct {
	PluginName  string
	PluginDesc  string
	PluginCaps  []Capability
	ExecuteFn   func(ctx context.Context, input string, vars map[string]string, w io.Writer) error
}

func (s *StubPlugin) Name() string               { return s.PluginName }
func (s *StubPlugin) Description() string         { return s.PluginDesc }
func (s *StubPlugin) Capabilities() []Capability  { return s.PluginCaps }
func (s *StubPlugin) Close() error                { return nil }
func (s *StubPlugin) Execute(ctx context.Context, input string, vars map[string]string, w io.Writer) error {
	if s.ExecuteFn != nil {
		return s.ExecuteFn(ctx, input, vars, w)
	}
	return nil
}
```

**Step 4: Create internal/plugin/manager.go**

```go
package plugin

import "sync"

// Manager holds the registry of all active plugins.
type Manager struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
}

// NewManager returns an empty plugin manager.
func NewManager() *Manager {
	return &Manager{plugins: make(map[string]Plugin)}
}

// Register adds a plugin. Silently replaces any existing plugin with the same name.
func (m *Manager) Register(p Plugin) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.plugins[p.Name()] = p
}

// Get returns a plugin by name.
func (m *Manager) Get(name string) (Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.plugins[name]
	return p, ok
}

// List returns all registered plugins in no guaranteed order.
func (m *Manager) List() []Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Plugin, 0, len(m.plugins))
	for _, p := range m.plugins {
		out = append(out, p)
	}
	return out
}
```

**Step 5: Run test to verify it passes**

```bash
go test ./internal/plugin/... -v
```

Expected: PASS — 3 tests passing.

**Step 6: Commit**

```bash
git add internal/plugin/
git commit -m "feat(plugin): add universal Plugin interface and Manager"
```

---

### Task 3: Tier 2 CliAdapter (wraps any CLI subprocess)

**Files:**
- Create: `internal/plugin/cli_adapter.go`
- Create: `internal/plugin/cli_adapter_test.go`

**Step 1: Write the failing test**

Create `internal/plugin/cli_adapter_test.go`:

```go
package plugin_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/adam-stokes/orcai/internal/plugin"
)

func TestCliAdapter_Name(t *testing.T) {
	a := plugin.NewCliAdapter("echo-tool", "A simple echo tool", "echo")
	if a.Name() != "echo-tool" {
		t.Errorf("expected 'echo-tool', got %q", a.Name())
	}
}

func TestCliAdapter_Execute(t *testing.T) {
	// Use `echo` as a trivial CLI that outputs its args and exits.
	// Input is passed via stdin; args are the command arguments.
	a := plugin.NewCliAdapter("echo-tool", "echoes input", "cat")
	var buf bytes.Buffer
	err := a.Execute(context.Background(), "hello world\n", nil, &buf)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(buf.String(), "hello world") {
		t.Errorf("expected output to contain 'hello world', got %q", buf.String())
	}
}

func TestCliAdapter_Execute_ContextCancel(t *testing.T) {
	// `sleep 10` will be cancelled by ctx before it finishes.
	a := plugin.NewCliAdapter("sleep-tool", "sleeps", "sleep", "10")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	var buf bytes.Buffer
	err := a.Execute(ctx, "", nil, &buf)
	if err == nil {
		t.Error("expected an error when context is cancelled")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/plugin/... -run TestCliAdapter -v
```

Expected: FAIL — `NewCliAdapter` undefined.

**Step 3: Create internal/plugin/cli_adapter.go**

```go
package plugin

import (
	"context"
	"io"
	"os/exec"
	"strings"
)

// CliAdapter wraps an arbitrary CLI tool as a Plugin.
// Input is written to the subprocess stdin; stdout is streamed to the writer.
// Extra args (beyond the command name) are passed as command-line arguments.
type CliAdapter struct {
	name string
	desc string
	cmd  string
	args []string
}

// NewCliAdapter creates a Tier 2 plugin that wraps cmd.
// args are fixed command-line arguments prepended to every Execute call.
func NewCliAdapter(name, description, cmd string, args ...string) *CliAdapter {
	return &CliAdapter{name: name, desc: description, cmd: cmd, args: args}
}

func (c *CliAdapter) Name() string              { return c.name }
func (c *CliAdapter) Description() string        { return c.desc }
func (c *CliAdapter) Capabilities() []Capability { return nil }
func (c *CliAdapter) Close() error               { return nil }

// Execute spawns the subprocess, writes input to stdin, and streams stdout to w.
func (c *CliAdapter) Execute(ctx context.Context, input string, _ map[string]string, w io.Writer) error {
	cmd := exec.CommandContext(ctx, c.cmd, c.args...)
	cmd.Stdin = strings.NewReader(input)
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/plugin/... -v
```

Expected: PASS — all 6 tests passing.

**Step 5: Commit**

```bash
git add internal/plugin/cli_adapter.go internal/plugin/cli_adapter_test.go
git commit -m "feat(plugin): add Tier 2 CliAdapter for CLI subprocess wrapping"
```

---

## Phase 3: Pipeline Engine

### Task 4: Add yaml dependency and pipeline YAML types + loader

**Files:**
- Modify: `go.mod`, `go.sum`
- Create: `internal/pipeline/pipeline.go`
- Create: `internal/pipeline/pipeline_test.go`

**Step 1: Add yaml dependency**

```bash
go get gopkg.in/yaml.v3
```

Expected: `go.mod` and `go.sum` updated.

**Step 2: Write the failing test**

Create `internal/pipeline/pipeline_test.go`:

```go
package pipeline_test

import (
	"strings"
	"testing"

	"github.com/adam-stokes/orcai/internal/pipeline"
)

const sampleYAML = `
name: test-pipeline
version: "1.0"
steps:
  - id: step1
    type: input
    prompt: "Enter topic:"
  - id: step2
    plugin: claude
    model: claude-sonnet-4-6
    prompt: "Summarize: {{step1.out}}"
    condition:
      if: "contains:spec"
      then: step3a
      else: step3b
  - id: step3a
    plugin: openspec
    input: "{{step2.out}}"
  - id: output
    type: output
    publish_to: "pipeline.test-pipeline.done"
`

func TestLoad_Valid(t *testing.T) {
	p, err := pipeline.Load(strings.NewReader(sampleYAML))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if p.Name != "test-pipeline" {
		t.Errorf("expected name 'test-pipeline', got %q", p.Name)
	}
	if len(p.Steps) != 4 {
		t.Errorf("expected 4 steps, got %d", len(p.Steps))
	}
	if p.Steps[0].ID != "step1" {
		t.Errorf("expected first step id 'step1', got %q", p.Steps[0].ID)
	}
	if p.Steps[1].Condition.If != "contains:spec" {
		t.Errorf("expected condition 'contains:spec', got %q", p.Steps[1].Condition.If)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	_, err := pipeline.Load(strings.NewReader(":::bad yaml:::"))
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoad_MissingName(t *testing.T) {
	_, err := pipeline.Load(strings.NewReader("version: '1.0'\nsteps: []"))
	if err == nil {
		t.Error("expected error when name is missing")
	}
}
```

**Step 3: Run test to verify it fails**

```bash
go test ./internal/pipeline/... -v
```

Expected: FAIL — `pipeline` package does not exist.

**Step 4: Create internal/pipeline/pipeline.go**

```go
package pipeline

import (
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// Pipeline is the top-level definition loaded from a .pipeline.yaml file.
type Pipeline struct {
	Name    string  `yaml:"name"`
	Version string  `yaml:"version"`
	Steps   []Step  `yaml:"steps"`
}

// Step is one unit of work in a pipeline.
type Step struct {
	ID        string    `yaml:"id"`
	Type      string    `yaml:"type"`   // "input", "output", or empty (plugin step)
	Plugin    string    `yaml:"plugin"` // plugin name for non-input/output steps
	Model     string    `yaml:"model"`
	Prompt    string    `yaml:"prompt"`
	Input     string    `yaml:"input"`
	PublishTo string    `yaml:"publish_to"`
	Condition Condition `yaml:"condition"`
}

// Condition describes a branch: if the expression is true, go to Then, else go to Else.
type Condition struct {
	If   string `yaml:"if"`
	Then string `yaml:"then"`
	Else string `yaml:"else"`
}

// Load reads and validates a Pipeline from r.
func Load(r io.Reader) (*Pipeline, error) {
	var p Pipeline
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("pipeline yaml: %w", err)
	}
	if p.Name == "" {
		return nil, fmt.Errorf("pipeline yaml: name is required")
	}
	return &p, nil
}
```

**Step 5: Run tests to verify they pass**

```bash
go test ./internal/pipeline/... -v
```

Expected: PASS — 3 tests passing.

**Step 6: Commit**

```bash
git add go.mod go.sum internal/pipeline/
git commit -m "feat(pipeline): add YAML types and loader with validation"
```

---

### Task 5: Template engine — interpolate {{stepN.out}} variables

**Files:**
- Create: `internal/pipeline/template.go`
- Modify: `internal/pipeline/pipeline_test.go` (add template tests)

**Step 1: Write the failing tests**

Append to `internal/pipeline/pipeline_test.go`:

```go
func TestInterpolate_Simple(t *testing.T) {
	vars := map[string]string{"step1.out": "golang plugins"}
	result := pipeline.Interpolate("Summarize: {{step1.out}}", vars)
	if result != "Summarize: golang plugins" {
		t.Errorf("got %q", result)
	}
}

func TestInterpolate_Multiple(t *testing.T) {
	vars := map[string]string{"a.out": "foo", "b.out": "bar"}
	result := pipeline.Interpolate("{{a.out}} and {{b.out}}", vars)
	if result != "foo and bar" {
		t.Errorf("got %q", result)
	}
}

func TestInterpolate_Missing(t *testing.T) {
	vars := map[string]string{}
	result := pipeline.Interpolate("hello {{missing.out}}", vars)
	// Missing vars are left as-is.
	if result != "hello {{missing.out}}" {
		t.Errorf("got %q", result)
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/pipeline/... -run TestInterpolate -v
```

Expected: FAIL — `Interpolate` undefined.

**Step 3: Create internal/pipeline/template.go**

```go
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
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/pipeline/... -v
```

Expected: PASS — all tests passing.

**Step 5: Commit**

```bash
git add internal/pipeline/template.go internal/pipeline/pipeline_test.go
git commit -m "feat(pipeline): add template interpolation for {{stepN.out}} variables"
```

---

### Task 6: Condition evaluator

**Files:**
- Create: `internal/pipeline/condition.go`
- Modify: `internal/pipeline/pipeline_test.go` (add condition tests)

**Step 1: Write the failing tests**

Append to `internal/pipeline/pipeline_test.go`:

```go
func TestEvalCondition_Contains(t *testing.T) {
	if !pipeline.EvalCondition("contains:spec", "openspec output here") {
		t.Error("expected true for contains:spec")
	}
	if pipeline.EvalCondition("contains:spec", "nothing here") {
		t.Error("expected false for contains:spec")
	}
}

func TestEvalCondition_Always(t *testing.T) {
	if !pipeline.EvalCondition("always", "anything") {
		t.Error("expected always to be true")
	}
}

func TestEvalCondition_LenGt(t *testing.T) {
	if !pipeline.EvalCondition("len > 5", "hello world") {
		t.Error("expected true for len > 5 on 11-char string")
	}
	if pipeline.EvalCondition("len > 5", "hi") {
		t.Error("expected false for len > 5 on 2-char string")
	}
}

func TestEvalCondition_Matches(t *testing.T) {
	if !pipeline.EvalCondition("matches:^go", "golang is great") {
		t.Error("expected true for matches:^go")
	}
	if pipeline.EvalCondition("matches:^go", "python is great") {
		t.Error("expected false for matches:^go")
	}
}

func TestEvalCondition_Unknown(t *testing.T) {
	// Unknown expressions default to false.
	if pipeline.EvalCondition("unknown-expr", "anything") {
		t.Error("expected false for unknown expression")
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/pipeline/... -run TestEvalCondition -v
```

Expected: FAIL — `EvalCondition` undefined.

**Step 3: Create internal/pipeline/condition.go**

```go
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
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/pipeline/... -v
```

Expected: PASS — all tests passing.

**Step 5: Commit**

```bash
git add internal/pipeline/condition.go internal/pipeline/pipeline_test.go
git commit -m "feat(pipeline): add condition evaluator (contains, matches, len, always)"
```

---

### Task 7: Pipeline interpreter/runner

**Files:**
- Create: `internal/pipeline/runner.go`
- Create: `internal/pipeline/runner_test.go`

**Step 1: Write the failing tests**

Create `internal/pipeline/runner_test.go`:

```go
package pipeline_test

import (
	"context"
	"strings"
	"testing"

	"github.com/adam-stokes/orcai/internal/pipeline"
	"github.com/adam-stokes/orcai/internal/plugin"
)

func TestRunner_LinearPipeline(t *testing.T) {
	// A simple two-step pipeline: echo input through a stub plugin.
	p := &pipeline.Pipeline{
		Name:    "linear-test",
		Version: "1.0",
		Steps: []pipeline.Step{
			{ID: "s1", Type: "input", Prompt: "Enter:"},
			{ID: "s2", Plugin: "echo"},
			{ID: "s3", Type: "output"},
		},
	}

	mgr := plugin.NewManager()
	mgr.Register(&plugin.StubPlugin{
		PluginName: "echo",
		ExecuteFn: func(ctx context.Context, input string, vars map[string]string, w interface{ Write([]byte) (int, error) }) error {
			_, err := w.Write([]byte("echoed: " + input))
			return err
		},
	})

	result, err := pipeline.Run(context.Background(), p, mgr, "hello world")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(result, "echoed: hello world") {
		t.Errorf("expected 'echoed: hello world' in output, got %q", result)
	}
}

func TestRunner_ConditionalBranch(t *testing.T) {
	p := &pipeline.Pipeline{
		Name: "branch-test",
		Steps: []pipeline.Step{
			{ID: "s1", Type: "input"},
			{
				ID:     "s2",
				Plugin: "classifier",
				Condition: pipeline.Condition{
					If:   "contains:go",
					Then: "golang-step",
					Else: "other-step",
				},
			},
			{ID: "golang-step", Plugin: "go-handler"},
			{ID: "other-step", Plugin: "other-handler"},
			{ID: "out", Type: "output"},
		},
	}

	mgr := plugin.NewManager()
	mgr.Register(&plugin.StubPlugin{
		PluginName: "classifier",
		ExecuteFn:  func(_ context.Context, input string, _ map[string]string, w interface{ Write([]byte) (int, error) }) error {
			_, err := w.Write([]byte(input))
			return err
		},
	})
	mgr.Register(&plugin.StubPlugin{
		PluginName: "go-handler",
		ExecuteFn:  func(_ context.Context, _ string, _ map[string]string, w interface{ Write([]byte) (int, error) }) error {
			_, err := w.Write([]byte("handled by go"))
			return err
		},
	})
	mgr.Register(&plugin.StubPlugin{
		PluginName: "other-handler",
		ExecuteFn:  func(_ context.Context, _ string, _ map[string]string, w interface{ Write([]byte) (int, error) }) error {
			_, err := w.Write([]byte("handled by other"))
			return err
		},
	})

	result, err := pipeline.Run(context.Background(), p, mgr, "golang rocks")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(result, "handled by go") {
		t.Errorf("expected 'handled by go', got %q", result)
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/pipeline/... -run TestRunner -v
```

Expected: FAIL — `Run` undefined.

**Step 3: Create internal/pipeline/runner.go**

```go
package pipeline

import (
	"bytes"
	"context"
	"fmt"

	"github.com/adam-stokes/orcai/internal/plugin"
)

// Run executes a pipeline against the given plugin manager.
// userInput is the initial value injected for the first input step.
// Returns the final output string.
func Run(ctx context.Context, p *Pipeline, mgr *plugin.Manager, userInput string) (string, error) {
	vars := make(map[string]string)

	// Index steps by ID for O(1) branch lookups.
	byID := make(map[string]*Step, len(p.Steps))
	for i := range p.Steps {
		byID[p.Steps[i].ID] = &p.Steps[i]
	}

	var finalOutput string

	// Walk steps in order; conditional branches jump by ID.
	visited := make(map[string]bool)
	queue := make([]string, 0, len(p.Steps))
	for _, s := range p.Steps {
		queue = append(queue, s.ID)
	}

	i := 0
	for i < len(queue) {
		id := queue[i]
		i++

		if visited[id] {
			continue
		}
		visited[id] = true

		step, ok := byID[id]
		if !ok {
			return "", fmt.Errorf("pipeline: unknown step id %q", id)
		}

		switch step.Type {
		case "input":
			if userInput != "" {
				vars[step.ID+".out"] = userInput
			}
		case "output":
			finalOutput = vars[lastPluginStepOutput(p, byID, id)]
		default:
			// Plugin step.
			pl, ok := mgr.Get(step.Plugin)
			if !ok {
				return "", fmt.Errorf("pipeline: plugin %q not found", step.Plugin)
			}

			promptOrInput := Interpolate(step.Prompt+step.Input, vars)
			stepVars := copyVars(vars)
			stepVars["model"] = step.Model

			var buf bytes.Buffer
			if err := pl.Execute(ctx, promptOrInput, stepVars, &buf); err != nil {
				return "", fmt.Errorf("pipeline: step %q: %w", step.ID, err)
			}
			output := buf.String()
			vars[step.ID+".out"] = output

			// Evaluate branch condition if present.
			if step.Condition.If != "" {
				if EvalCondition(step.Condition.If, output) {
					if step.Condition.Then != "" {
						queue = append([]string{step.Condition.Then}, queue[i:]...)
						i = 0
					}
				} else {
					if step.Condition.Else != "" {
						queue = append([]string{step.Condition.Else}, queue[i:]...)
						i = 0
					}
				}
			}
		}
	}

	return finalOutput, nil
}

func copyVars(vars map[string]string) map[string]string {
	out := make(map[string]string, len(vars))
	for k, v := range vars {
		out[k] = v
	}
	return out
}

// lastPluginStepOutput finds the most recent plugin step output var for the output step.
func lastPluginStepOutput(p *Pipeline, byID map[string]*Step, outputID string) string {
	// Walk steps in reverse; find the last non-output step before this one.
	for i := len(p.Steps) - 1; i >= 0; i-- {
		s := &p.Steps[i]
		if s.ID == outputID {
			continue
		}
		if s.Type == "" && s.Plugin != "" {
			return s.ID + ".out"
		}
	}
	return ""
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/pipeline/... -v
```

Expected: PASS — all tests passing.

**Step 5: Commit**

```bash
git add internal/pipeline/runner.go internal/pipeline/runner_test.go
git commit -m "feat(pipeline): add pipeline runner with linear execution and conditional branching"
```

---

## Phase 4: Extended Discovery

### Task 8: Extend discovery to scan ~/.config/orcai/pipelines/

**Files:**
- Modify: `internal/discovery/discovery.go`
- Modify: `internal/discovery/discovery_test.go`

**Step 1: Write the failing test**

Append to `internal/discovery/discovery_test.go`:

```go
func TestScanPipelines_FindsYAML(t *testing.T) {
	dir := t.TempDir()
	pipelinesDir := filepath.Join(dir, "pipelines")
	if err := os.MkdirAll(pipelinesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	yamlPath := filepath.Join(pipelinesDir, "my-pipeline.pipeline.yaml")
	content := "name: my-pipeline\nversion: \"1.0\"\nsteps: []\n"
	if err := os.WriteFile(yamlPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	plugins, err := discovery.Discover(dir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	found := false
	for _, p := range plugins {
		if p.Name == "my-pipeline" && p.Type == discovery.TypePipeline {
			found = true
		}
	}
	if !found {
		t.Error("expected to discover 'my-pipeline' as TypePipeline")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/discovery/... -run TestScanPipelines -v
```

Expected: FAIL — `TypePipeline` undefined.

**Step 3: Modify internal/discovery/discovery.go**

Add `TypePipeline` constant and `scanPipelines` function:

```go
// Add to the PluginType constants:
TypePipeline  // .pipeline.yaml files in ~/.config/orcai/pipelines/
```

Add `PipelineFile` field to `Plugin` struct:

```go
PipelineFile string // only set for TypePipeline; absolute path to .pipeline.yaml
```

Update `Discover` to call `scanPipelines`:

```go
func Discover(configDir string) ([]Plugin, error) {
	native, err := scanNative(filepath.Join(configDir, "plugins"))
	if err != nil {
		return nil, err
	}
	pipelines, err := scanPipelines(filepath.Join(configDir, "pipelines"))
	if err != nil {
		return nil, err
	}

	nativeNames := make(map[string]bool, len(native)+len(pipelines))
	for _, p := range native {
		nativeNames[p.Name] = true
	}
	for _, p := range pipelines {
		nativeNames[p.Name] = true
	}

	plugins := append(native, pipelines...)
	for _, tool := range knownCLITools {
		if nativeNames[tool.Name] {
			continue
		}
		if _, err := exec.LookPath(tool.Command); err == nil {
			t := tool
			t.Type = TypeCLIWrapper
			plugins = append(plugins, t)
		}
	}
	return plugins, nil
}

func scanPipelines(dir string) ([]Plugin, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var plugins []Plugin
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".pipeline.yaml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".pipeline.yaml")
		fullPath := filepath.Join(dir, e.Name())
		plugins = append(plugins, Plugin{
			Name:         name,
			Command:      fullPath,
			Type:         TypePipeline,
			PipelineFile: fullPath,
		})
	}
	return plugins, nil
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/discovery/... -v
```

Expected: PASS — all tests passing.

**Step 5: Commit**

```bash
git add internal/discovery/discovery.go internal/discovery/discovery_test.go
git commit -m "feat(discovery): scan ~/.config/orcai/pipelines/ for TypePipeline plugins"
```

---

## Phase 5: Prompt Builder TUI

### Task 9: Prompt builder model and step list pane

**Files:**
- Create: `internal/promptbuilder/model.go`
- Create: `internal/promptbuilder/steplist.go`
- Create: `internal/promptbuilder/model_test.go`

**Step 1: Write the failing tests**

Create `internal/promptbuilder/model_test.go`:

```go
package promptbuilder_test

import (
	"testing"

	"github.com/adam-stokes/orcai/internal/pipeline"
	"github.com/adam-stokes/orcai/internal/promptbuilder"
)

func TestModel_New(t *testing.T) {
	m := promptbuilder.New(nil)
	if m == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestModel_AddStep(t *testing.T) {
	m := promptbuilder.New(nil)
	m.AddStep(pipeline.Step{ID: "s1", Type: "input", Prompt: "Enter:"})
	m.AddStep(pipeline.Step{ID: "s2", Plugin: "claude"})
	if len(m.Steps()) != 2 {
		t.Errorf("expected 2 steps, got %d", len(m.Steps()))
	}
}

func TestModel_SelectStep(t *testing.T) {
	m := promptbuilder.New(nil)
	m.AddStep(pipeline.Step{ID: "s1"})
	m.AddStep(pipeline.Step{ID: "s2"})
	m.SelectStep(1)
	if m.SelectedIndex() != 1 {
		t.Errorf("expected selected index 1, got %d", m.SelectedIndex())
	}
}

func TestModel_SetName(t *testing.T) {
	m := promptbuilder.New(nil)
	m.SetName("my-pipeline")
	if m.Name() != "my-pipeline" {
		t.Errorf("expected 'my-pipeline', got %q", m.Name())
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/promptbuilder/... -v
```

Expected: FAIL — package does not exist.

**Step 3: Create internal/promptbuilder/model.go**

```go
package promptbuilder

import (
	"github.com/adam-stokes/orcai/internal/pipeline"
	"github.com/adam-stokes/orcai/internal/plugin"
)

// Model is the state for the prompt builder TUI.
type Model struct {
	name          string
	steps         []pipeline.Step
	selectedIndex int
	pluginMgr     *plugin.Manager
}

// New creates a new prompt builder model.
// pluginMgr may be nil in tests.
func New(pluginMgr *plugin.Manager) *Model {
	return &Model{pluginMgr: pluginMgr}
}

func (m *Model) Name() string                { return m.name }
func (m *Model) SetName(name string)          { m.name = name }
func (m *Model) Steps() []pipeline.Step       { return m.steps }
func (m *Model) SelectedIndex() int            { return m.selectedIndex }

// AddStep appends a step to the pipeline.
func (m *Model) AddStep(s pipeline.Step) {
	m.steps = append(m.steps, s)
}

// SelectStep sets the active step by index (clamped to valid range).
func (m *Model) SelectStep(i int) {
	if i < 0 {
		i = 0
	}
	if i >= len(m.steps) {
		i = len(m.steps) - 1
	}
	m.selectedIndex = i
}

// ToPipeline converts the current model state to a Pipeline.
func (m *Model) ToPipeline() *pipeline.Pipeline {
	steps := make([]pipeline.Step, len(m.steps))
	copy(steps, m.steps)
	return &pipeline.Pipeline{
		Name:    m.name,
		Version: "1.0",
		Steps:   steps,
	}
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/promptbuilder/... -v
```

Expected: PASS — 4 tests passing.

**Step 5: Commit**

```bash
git add internal/promptbuilder/
git commit -m "feat(promptbuilder): add Model with step management and pipeline conversion"
```

---

### Task 10: Prompt builder bubbletea view (modal, step list, config form)

**Files:**
- Create: `internal/promptbuilder/view.go`
- Create: `internal/promptbuilder/keys.go`

> This task is pure TUI rendering — no unit tests for rendering output (bubbletea views are integration-tested by running the app). We verify by running the builder interactively.

**Step 1: Create internal/promptbuilder/keys.go**

```go
package promptbuilder

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Tab      key.Binding
	AddStep  key.Binding
	Run      key.Binding
	Save     key.Binding
	Quit     key.Binding
	Help     key.Binding
}

var keys = keyMap{
	Up:      key.NewBinding(key.WithKeys("up", "k"),   key.WithHelp("↑/k", "prev step")),
	Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "next step")),
	Tab:     key.NewBinding(key.WithKeys("tab"),        key.WithHelp("tab", "next field")),
	AddStep: key.NewBinding(key.WithKeys("+"),          key.WithHelp("+", "add step")),
	Run:     key.NewBinding(key.WithKeys("r"),          key.WithHelp("r", "run")),
	Save:    key.NewBinding(key.WithKeys("s"),          key.WithHelp("s", "save")),
	Quit:    key.NewBinding(key.WithKeys("esc", "q"),   key.WithHelp("esc", "quit")),
	Help:    key.NewBinding(key.WithKeys("?"),          key.WithHelp("?", "help")),
}
```

**Step 2: Create internal/promptbuilder/view.go**

```go
package promptbuilder

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/adam-stokes/orcai/internal/pipeline"
)

// Styles — BBS aesthetic: border panels, muted palette.
var (
	borderStyle  = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(lipgloss.Color("63"))
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	selectedStep = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	dimStep      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	statusBar    = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
	labelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
)

// BubbleModel wraps Model and implements tea.Model.
type BubbleModel struct {
	inner  *Model
	width  int
	height int
	output string // result of last run
}

// NewBubble creates a bubbletea-compatible model.
func NewBubble(m *Model) *BubbleModel {
	return &BubbleModel{inner: m}
}

func (b *BubbleModel) Init() tea.Cmd { return nil }

func (b *BubbleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.width = msg.Width
		b.height = msg.Height
	case tea.KeyMsg:
		switch {
		case keys.Quit.Matches(msg):
			return b, tea.Quit
		case keys.Up.Matches(msg):
			b.inner.SelectStep(b.inner.SelectedIndex() - 1)
		case keys.Down.Matches(msg):
			b.inner.SelectStep(b.inner.SelectedIndex() + 1)
		case keys.AddStep.Matches(msg):
			id := fmt.Sprintf("step%d", len(b.inner.Steps())+1)
			b.inner.AddStep(pipeline.Step{ID: id, Plugin: "claude"})
		}
	}
	return b, nil
}

func (b *BubbleModel) View() string {
	if b.width == 0 {
		return "Loading..."
	}

	w := b.width * 80 / 100
	h := b.height * 80 / 100

	// Left pane: step list (30% of modal width).
	leftW := w * 30 / 100
	rightW := w - leftW - 4 // 4 for borders

	leftContent := titleStyle.Render("STEPS") + "\n" + strings.Repeat("─", leftW-2) + "\n"
	for i, s := range b.inner.Steps() {
		label := fmt.Sprintf("[%d] %s", i+1, stepLabel(s))
		if i == b.inner.SelectedIndex() {
			leftContent += selectedStep.Render("→ "+label) + "\n"
		} else {
			leftContent += dimStep.Render("  "+label) + "\n"
		}
	}
	leftContent += "\n" + dimStep.Render("[+] add step")

	// Right pane: config form for selected step.
	rightContent := ""
	steps := b.inner.Steps()
	if len(steps) > 0 {
		sel := steps[b.inner.SelectedIndex()]
		rightContent = titleStyle.Render(fmt.Sprintf("STEP %d — CONFIG", b.inner.SelectedIndex()+1)) + "\n"
		rightContent += strings.Repeat("─", rightW-2) + "\n"
		rightContent += labelStyle.Render("ID:      ") + sel.ID + "\n"
		rightContent += labelStyle.Render("Plugin:  ") + sel.Plugin + "\n"
		rightContent += labelStyle.Render("Model:   ") + sel.Model + "\n"
		rightContent += labelStyle.Render("Prompt:  ") + sel.Prompt + "\n"
		if sel.Condition.If != "" {
			rightContent += labelStyle.Render("Cond:    ") + sel.Condition.If + "\n"
			rightContent += labelStyle.Render("  then→  ") + sel.Condition.Then + "\n"
			rightContent += labelStyle.Render("  else→  ") + sel.Condition.Else + "\n"
		}
	}

	left := lipgloss.NewStyle().Width(leftW).Height(h - 6).Render(leftContent)
	right := lipgloss.NewStyle().Width(rightW).Height(h - 6).Render(rightContent)
	panes := lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)

	header := titleStyle.Render("PIPELINE BUILDER") +
		lipgloss.NewStyle().Width(w-20).Render("") +
		dimStep.Render("[?] help  [x]")
	nameRow := labelStyle.Render("NAME: ") + b.inner.Name()
	footer := statusBar.Render("[r] run  [s] save  [tab] next field  [↑↓] steps  [esc] quit")

	modal := borderStyle.Width(w).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			header,
			nameRow,
			strings.Repeat("═", w-4),
			panes,
			strings.Repeat("═", w-4),
			footer,
		),
	)

	// Center the modal.
	marginLeft := (b.width - w) / 2
	marginTop := (b.height - h) / 2
	return lipgloss.NewStyle().
		MarginLeft(marginLeft).
		MarginTop(marginTop).
		Render(modal)
}

func stepLabel(s pipeline.Step) string {
	if s.Type != "" {
		return s.Type
	}
	if s.Plugin != "" {
		return s.Plugin
	}
	return s.ID
}
```

**Step 3: Verify it compiles**

```bash
go build ./internal/promptbuilder/...
```

Expected: PASS.

**Step 4: Commit**

```bash
git add internal/promptbuilder/view.go internal/promptbuilder/keys.go
git commit -m "feat(promptbuilder): add bubbletea modal view with step list and config pane"
```

---

### Task 11: Pipeline save function

**Files:**
- Create: `internal/promptbuilder/save.go`
- Modify: `internal/promptbuilder/model_test.go` (add save test)

**Step 1: Write the failing test**

Append to `internal/promptbuilder/model_test.go`:

```go
import (
	"os"
	"path/filepath"
	"testing"

	"github.com/adam-stokes/orcai/internal/pipeline"
	"github.com/adam-stokes/orcai/internal/promptbuilder"
)

func TestSave_WritesYAML(t *testing.T) {
	dir := t.TempDir()
	m := promptbuilder.New(nil)
	m.SetName("my-test-pipeline")
	m.AddStep(pipeline.Step{ID: "s1", Type: "input", Prompt: "Enter:"})
	m.AddStep(pipeline.Step{ID: "s2", Plugin: "claude", Model: "claude-sonnet-4-6"})

	outPath := filepath.Join(dir, "my-test-pipeline.pipeline.yaml")
	if err := promptbuilder.Save(m, outPath); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "my-test-pipeline") {
		t.Errorf("expected pipeline name in YAML, got: %s", string(data))
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/promptbuilder/... -run TestSave -v
```

Expected: FAIL — `Save` undefined.

**Step 3: Create internal/promptbuilder/save.go**

```go
package promptbuilder

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Save writes the pipeline to path as YAML.
func Save(m *Model, path string) error {
	p := m.ToPipeline()
	data, err := yaml.Marshal(p)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/promptbuilder/... -v
```

Expected: PASS.

**Step 5: Commit**

```bash
git add internal/promptbuilder/save.go internal/promptbuilder/model_test.go
git commit -m "feat(promptbuilder): add Save to write pipeline YAML to disk"
```

---

## Phase 6: Wire Up — Cobra Commands

### Task 12: Add `pipeline` cobra command with `build` and `run` subcommands

**Files:**
- Create: `cmd/pipeline.go`

**Step 1: Create cmd/pipeline.go**

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/adam-stokes/orcai/internal/pipeline"
	"github.com/adam-stokes/orcai/internal/plugin"
	"github.com/adam-stokes/orcai/internal/promptbuilder"
)

func init() {
	rootCmd.AddCommand(pipelineCmd)
	pipelineCmd.AddCommand(pipelineBuildCmd)
	pipelineCmd.AddCommand(pipelineRunCmd)
}

var pipelineCmd = &cobra.Command{
	Use:   "pipeline",
	Short: "Manage and run AI pipelines",
}

var pipelineBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Open the interactive pipeline builder",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := plugin.NewManager()
		// Register known CLI tools as Tier 2 adapters.
		for _, name := range []string{"claude", "gemini", "openspec", "openclaw"} {
			mgr.Register(plugin.NewCliAdapter(name, name+" CLI", name))
		}

		m := promptbuilder.New(mgr)
		m.SetName("new-pipeline")
		m.AddStep(pipeline.Step{ID: "input", Type: "input", Prompt: "Enter your prompt:"})
		m.AddStep(pipeline.Step{ID: "step1", Plugin: "claude", Model: "claude-sonnet-4-6"})
		m.AddStep(pipeline.Step{ID: "output", Type: "output"})

		bubble := promptbuilder.NewBubble(m)
		p := tea.NewProgram(bubble, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("pipeline builder: %w", err)
		}
		return nil
	},
}

var pipelineRunCmd = &cobra.Command{
	Use:   "run <name>",
	Short: "Run a saved pipeline by name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		configDir, err := orcaiConfigDir()
		if err != nil {
			return err
		}

		yamlPath := filepath.Join(configDir, "pipelines", name+".pipeline.yaml")
		f, err := os.Open(yamlPath)
		if err != nil {
			return fmt.Errorf("pipeline %q not found: %w", name, err)
		}
		defer f.Close()

		p, err := pipeline.Load(f)
		if err != nil {
			return err
		}

		mgr := plugin.NewManager()
		for _, n := range []string{"claude", "gemini", "openspec", "openclaw"} {
			mgr.Register(plugin.NewCliAdapter(n, n+" CLI", n))
		}

		result, err := pipeline.Run(cmd.Context(), p, mgr, "")
		if err != nil {
			return err
		}
		fmt.Println(result)
		return nil
	},
}

func orcaiConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "orcai"), nil
}
```

**Step 2: Verify it compiles**

```bash
go build ./...
```

Expected: PASS.

**Step 3: Smoke test the new commands exist**

```bash
./bin/orcai pipeline --help
```

Expected: shows `build` and `run` subcommands.

**Step 4: Commit**

```bash
git add cmd/pipeline.go
git commit -m "feat(cmd): add pipeline build and pipeline run cobra commands"
```

---

### Task 13: Full test suite pass + final build

**Step 1: Run all tests**

```bash
go test ./... -v
```

Expected: PASS — all tests across `internal/plugin`, `internal/pipeline`, `internal/promptbuilder`, `internal/discovery`, `internal/bus` passing.

**Step 2: Build final binary**

```bash
make build
```

Expected: `bin/orcai` built successfully.

**Step 3: Commit if any cleanup needed**

```bash
git add -u
git commit -m "chore: final cleanup and full test pass for plugin system"
```

---

## Quick Reference

| Command | Purpose |
|---------|---------|
| `make proto` | Regenerate protobuf Go files after editing `.proto` |
| `go test ./...` | Run all tests |
| `make build` | Build `bin/orcai` |
| `./bin/orcai pipeline build` | Open the interactive prompt builder |
| `./bin/orcai pipeline run <name>` | Run a saved pipeline |

## Config Directory Layout

```
~/.config/orcai/
  plugins/          ← Tier 1 native plugin binaries
  pipelines/        ← *.pipeline.yaml files (Tier 3)
  wrappers/         ← Optional sidecar YAML for Tier 2 CLI wrappers
```
