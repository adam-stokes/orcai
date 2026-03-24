package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/adam-stokes/orcai/internal/adapters/shared"
	bridgepb "github.com/adam-stokes/orcai/proto/bridgepb"
)

const version = "0.1.0"

// Server implements the ProviderBridge gRPC service for the Claude CLI.
type Server struct {
	bridgepb.UnimplementedProviderBridgeServer
	cwd string
}

// New creates a new Claude adapter server.
func New(cwd string) *Server {
	return &Server{cwd: cwd}
}

// Register registers the server with a gRPC server instance.
func Register(s *grpc.Server, cwd string) {
	bridgepb.RegisterProviderBridgeServer(s, New(cwd))
}

// Describe returns provider metadata and capabilities scanned from the filesystem.
func (s *Server) Describe(_ context.Context, _ *bridgepb.DescribeRequest) (*bridgepb.DescribeResponse, error) {
	caps, err := scanCapabilities(s.cwd)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "scan capabilities: %v", err)
	}
	return &bridgepb.DescribeResponse{
		Name:         "Claude",
		Version:      version,
		Capabilities: caps,
	}, nil
}

// scanCapabilities scans skills and agents from the cwd filesystem.
func scanCapabilities(cwd string) ([]*bridgepb.Capability, error) {
	var caps []*bridgepb.Capability

	// ── Skills ──────────────────────────────────────────────────────────────
	// Scan .claude/skills/ and skills/ (dedup by real path).
	seen := make(map[string]bool)
	for _, rel := range []string{filepath.Join(".claude", "skills"), "skills"} {
		dir := filepath.Join(cwd, rel)
		real, err := filepath.EvalSymlinks(dir)
		if err != nil {
			continue
		}
		real, err = filepath.Abs(real)
		if err != nil {
			continue
		}
		if seen[real] {
			continue
		}
		seen[real] = true

		entries, err := os.ReadDir(real)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			skillMD := filepath.Join(real, e.Name(), "SKILL.md")
			if _, err := os.Stat(skillMD); err != nil {
				continue
			}
			caps = append(caps, &bridgepb.Capability{
				Name:   e.Name(),
				Kind:   "skill",
				Inject: "/" + e.Name(),
			})
		}
	}

	// ── Agents ──────────────────────────────────────────────────────────────
	// Build skill name set for dedup.
	skillNames := make(map[string]bool)
	for _, c := range caps {
		if c.Kind == "skill" {
			skillNames[c.Name] = true
		}
	}

	// Scan .claude/commands/*.md — skip if name matches a skill.
	commandsDir := filepath.Join(cwd, ".claude", "commands")
	if entries, err := os.ReadDir(commandsDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			name := strings.TrimSuffix(e.Name(), ".md")
			if skillNames[name] {
				continue
			}
			caps = append(caps, &bridgepb.Capability{
				Name:   name,
				Kind:   "agent",
				Inject: "@" + name + " ",
			})
		}
	}

	// AGENTS.md at repo root.
	if _, err := os.Stat(filepath.Join(cwd, "AGENTS.md")); err == nil {
		caps = append(caps, &bridgepb.Capability{
			Name:   "agents",
			Kind:   "agent",
			Inject: "@agents ",
		})
	}

	return caps, nil
}

// Send executes a claude CLI subprocess and streams the response.
// It accepts a bidirectional stream: the first SendRequest contains the prompt,
// and any subsequent SendRequests with a non-empty Input field are written to
// the subprocess stdin (for mid-stream interactive prompts).
func (s *Server) Send(stream bridgepb.ProviderBridge_SendServer) error {
	req, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.Internal, "recv initial request: %v", err)
	}

	args := []string{
		"-p",
		"--output-format", "stream-json",
		"--verbose",
		"--include-partial-messages",
		"--dangerously-skip-permissions",
	}
	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}
	if req.SessionId != "" {
		args = append(args, "--resume", req.SessionId)
	}
	args = append(args, req.Prompt)

	cmd := exec.CommandContext(stream.Context(), "claude", args...)
	if req.Cwd != "" {
		cmd.Dir = req.Cwd
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return status.Errorf(codes.Internal, "stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return status.Errorf(codes.Internal, "stdout pipe: %v", err)
	}
	defer stdout.Close()

	if err := cmd.Start(); err != nil {
		return status.Errorf(codes.Internal, "start claude: %v", err)
	}

	// Relay any mid-stream user input to the subprocess stdin.
	shared.RelayStdin(stream, stdin)

	var done bridgepb.DonePayload
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		var typ string
		if err := json.Unmarshal(raw["type"], &typ); err != nil {
			continue
		}

		switch typ {
		case "system":
			var subtype string
			json.Unmarshal(raw["subtype"], &subtype)
			if subtype == "init" {
				json.Unmarshal(raw["session_id"], &done.SessionId)
				json.Unmarshal(raw["model"], &done.Model)
				if req.SessionId != "" {
					done.SessionId = req.SessionId
				}
			}
		case "assistant":
			var msg struct {
				Message struct {
					Content []struct {
						Type  string          `json:"type"`
						Text  string          `json:"text"`
						Name  string          `json:"name"`  // for tool_use
						Input json.RawMessage `json:"input"` // for tool_use
					} `json:"content"`
				} `json:"message"`
			}
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				continue
			}
			for _, block := range msg.Message.Content {
				switch block.Type {
				case "text":
					if block.Text != "" {
						if err := stream.Send(&bridgepb.SendResponse{
							Payload: &bridgepb.SendResponse_Chunk{Chunk: block.Text},
						}); err != nil {
							return err
						}
					}
				case "tool_use":
					if block.Name != "" {
						// Compact the input JSON for display, truncate to 120 chars
						inputStr := string(block.Input)
						if inputStr == "" || inputStr == "null" {
							inputStr = ""
						} else {
							// Try to extract a meaningful string from the input
							var m map[string]interface{}
							if json.Unmarshal(block.Input, &m) == nil {
								// Common patterns: command, path, content, query
								for _, k := range []string{"command", "path", "file_path", "query", "content", "pattern"} {
									if v, ok := m[k]; ok {
										if s, ok := v.(string); ok {
											inputStr = s
											break
										}
									}
								}
							}
						}
						if len(inputStr) > 120 {
							inputStr = inputStr[:120] + "…"
						}
						statusJSON, _ := json.Marshal(map[string]string{"tool": block.Name, "input": inputStr})
						if err := stream.Send(&bridgepb.SendResponse{
							Payload: &bridgepb.SendResponse_Chunk{Chunk: "\x02" + string(statusJSON)},
						}); err != nil {
							return err
						}
					}
				}
			}
		case "result":
			var res struct {
				Usage struct {
					InputTokens     int `json:"input_tokens"`
					CacheReadTokens int `json:"cache_read_input_tokens"`
					OutputTokens    int `json:"output_tokens"`
				} `json:"usage"`
				ModelUsage map[string]struct {
					ContextWindow int `json:"contextWindow"`
				} `json:"modelUsage"`
			}
			if err := json.Unmarshal([]byte(line), &res); err != nil {
				continue
			}
			done.InputTokens = int32(res.Usage.InputTokens)
			done.CacheTokens = int32(res.Usage.CacheReadTokens)
			done.OutputTokens = int32(res.Usage.OutputTokens)
			if u, ok := res.ModelUsage[done.Model]; ok {
				done.ContextWindow = int32(u.ContextWindow)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return status.Errorf(codes.Internal, "stream read: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		return status.Errorf(codes.Internal, "claude exited: %v", err)
	}

	return stream.Send(&bridgepb.SendResponse{
		Payload: &bridgepb.SendResponse_Done{Done: &done},
	})
}

// Shutdown is a no-op for the Claude adapter — the process will exit when the manager kills it.
func (s *Server) Shutdown(_ context.Context, _ *bridgepb.ShutdownRequest) (*bridgepb.ShutdownResponse, error) {
	return &bridgepb.ShutdownResponse{}, nil
}
