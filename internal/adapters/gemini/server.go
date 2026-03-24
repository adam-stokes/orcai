package gemini

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/oauth2/google"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/adam-stokes/orcai/internal/adapters/shared"
	bridgepb "github.com/adam-stokes/orcai/proto/bridgepb"
)

const version = "0.1.0"

// Server implements the ProviderBridge gRPC service using Vertex AI.
type Server struct {
	bridgepb.UnimplementedProviderBridgeServer
	cwd string
}

func New(cwd string) *Server { return &Server{cwd: cwd} }
func Register(s *grpc.Server, cwd string) {
	bridgepb.RegisterProviderBridgeServer(s, New(cwd))
}

func (s *Server) Describe(_ context.Context, _ *bridgepb.DescribeRequest) (*bridgepb.DescribeResponse, error) {
	return &bridgepb.DescribeResponse{Name: "Gemini", Version: version}, nil
}

// convMsg is one stored conversation turn.
type convMsg struct {
	Role    string `json:"role"`    // "user" or "assistant"
	Content string `json:"content"`
}

func convHistoryPath(cwd, sessionID string) string {
	home, _ := os.UserHomeDir()
	encoded := strings.ReplaceAll(cwd, "/", "-")
	return filepath.Join(home, ".stok", "sessions", "history", encoded, sessionID+".json")
}

func loadConvHistory(cwd, sessionID string) []convMsg {
	if sessionID == "" {
		return nil
	}
	data, err := os.ReadFile(convHistoryPath(cwd, sessionID))
	if err != nil {
		return nil
	}
	var msgs []convMsg
	_ = json.Unmarshal(data, &msgs)
	return msgs
}

func saveConvHistory(cwd, sessionID string, msgs []convMsg) {
	if sessionID == "" {
		return
	}
	path := convHistoryPath(cwd, sessionID)
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	data, _ := json.MarshalIndent(msgs, "", "  ")
	_ = os.WriteFile(path, data, 0644)
}

func generateID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s-%x", prefix, b)
}

// Send dispatches to sendAnthropic or sendGemini based on the model name.
//
// Config is read from the project .env:
//
//	GOOGLE_CLOUD_PROJECT  — Vertex AI project ID
//	GOOGLE_CLOUD_LOCATION — region (default: us-central1)
//	GEMINI_MODEL          — model name (default: claude-sonnet-4-6)
func (s *Server) Send(stream bridgepb.ProviderBridge_SendServer) error {
	req, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.Internal, "recv initial request: %v", err)
	}

	dotenv := shared.LoadDotEnv(req.Cwd)

	project := dotenv["GOOGLE_CLOUD_PROJECT"]
	if project == "" {
		return status.Error(codes.InvalidArgument, "GOOGLE_CLOUD_PROJECT not set in .env")
	}
	location := dotenv["GOOGLE_CLOUD_LOCATION"]
	if location == "" {
		location = "us-central1"
	}
	model := req.Model
	if model == "" {
		model = dotenv["GEMINI_MODEL"]
	}
	if model == "" {
		model = "claude-sonnet-4-6"
	}

	if strings.HasPrefix(model, "gemini") {
		return s.sendGemini(stream, req, project, location, model)
	}
	return s.sendAnthropic(stream, req, project, location, model)
}

// sendAnthropic sends to the Vertex AI Anthropic endpoint with full conversation history.
func (s *Server) sendAnthropic(stream bridgepb.ProviderBridge_SendServer, req *bridgepb.SendRequest, project, location, model string) error {
	type contentBlock struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	type apiMessage struct {
		Role    string         `json:"role"`
		Content []contentBlock `json:"content"`
	}

	history := loadConvHistory(req.Cwd, req.SessionId)

	messages := make([]apiMessage, 0, len(history)+1)
	for _, h := range history {
		messages = append(messages, apiMessage{
			Role:    h.Role,
			Content: []contentBlock{{Type: "text", Text: h.Content}},
		})
	}
	messages = append(messages, apiMessage{
		Role:    "user",
		Content: []contentBlock{{Type: "text", Text: req.Prompt}},
	})

	endpoint := fmt.Sprintf(
		"https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/anthropic/models/%s:streamRawPredict",
		location, project, location, model,
	)

	payload := map[string]interface{}{
		"anthropic_version": "vertex-2023-10-16",
		"max_tokens":        8192,
		"stream":            true,
		"messages":          messages,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return status.Errorf(codes.Internal, "marshal request: %v", err)
	}

	ts, err := google.DefaultTokenSource(stream.Context(), "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return status.Errorf(codes.Internal, "ADC token source: %v", err)
	}
	token, err := ts.Token()
	if err != nil {
		return status.Errorf(codes.Internal, "get token: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(stream.Context(), "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return status.Errorf(codes.Internal, "build request: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return status.Errorf(codes.Internal, "http request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return status.Errorf(codes.Internal, "vertex AI %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var done bridgepb.DonePayload
	if req.SessionId != "" {
		done.SessionId = req.SessionId
	}

	var assistantContent strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		data, ok := strings.CutPrefix(line, "data: ")
		if !ok || data == "[DONE]" {
			continue
		}

		var ev map[string]json.RawMessage
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			continue
		}
		var typ string
		if err := json.Unmarshal(ev["type"], &typ); err != nil {
			continue
		}

		switch typ {
		case "message_start":
			var ms struct {
				Message struct {
					ID    string `json:"id"`
					Model string `json:"model"`
				} `json:"message"`
			}
			if err := json.Unmarshal([]byte(data), &ms); err == nil {
				if done.SessionId == "" {
					done.SessionId = ms.Message.ID
				}
				done.Model = ms.Message.Model
				if done.Model == "" {
					done.Model = model
				}
			}

		case "content_block_delta":
			var cbd struct {
				Delta struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"delta"`
			}
			if err := json.Unmarshal([]byte(data), &cbd); err != nil {
				continue
			}
			if cbd.Delta.Type == "text_delta" && cbd.Delta.Text != "" {
				assistantContent.WriteString(cbd.Delta.Text)
				if err := stream.Send(&bridgepb.SendResponse{
					Payload: &bridgepb.SendResponse_Chunk{Chunk: cbd.Delta.Text},
				}); err != nil {
					return err
				}
			}

		case "message_delta":
			var md struct {
				Usage struct {
					OutputTokens int `json:"output_tokens"`
				} `json:"usage"`
			}
			if err := json.Unmarshal([]byte(data), &md); err == nil {
				done.OutputTokens = int32(md.Usage.OutputTokens)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return status.Errorf(codes.Internal, "stream read: %v", err)
	}

	// Persist conversation history for session continuity and resume display.
	if done.SessionId != "" && assistantContent.Len() > 0 {
		updated := make([]convMsg, len(history), len(history)+2)
		copy(updated, history)
		updated = append(updated, convMsg{Role: "user", Content: req.Prompt}, convMsg{Role: "assistant", Content: assistantContent.String()})
		go saveConvHistory(req.Cwd, done.SessionId, updated)
	}

	return stream.Send(&bridgepb.SendResponse{
		Payload: &bridgepb.SendResponse_Done{Done: &done},
	})
}

// sendGemini sends to the Vertex AI Gemini endpoint with full conversation history.
func (s *Server) sendGemini(stream bridgepb.ProviderBridge_SendServer, req *bridgepb.SendRequest, project, location, model string) error {
	type part struct {
		Text string `json:"text"`
	}
	type content struct {
		Role  string `json:"role"`
		Parts []part `json:"parts"`
	}

	history := loadConvHistory(req.Cwd, req.SessionId)

	contents := make([]content, 0, len(history)+1)
	for _, h := range history {
		role := h.Role
		if role == "assistant" {
			role = "model"
		}
		contents = append(contents, content{Role: role, Parts: []part{{Text: h.Content}}})
	}
	contents = append(contents, content{Role: "user", Parts: []part{{Text: req.Prompt}}})

	endpoint := fmt.Sprintf(
		"https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:streamGenerateContent?alt=sse",
		location, project, location, model,
	)

	payload := map[string]interface{}{
		"contents": contents,
		"generationConfig": map[string]interface{}{
			"maxOutputTokens": 8192,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return status.Errorf(codes.Internal, "marshal request: %v", err)
	}

	ts, err := google.DefaultTokenSource(stream.Context(), "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return status.Errorf(codes.Internal, "ADC token source: %v", err)
	}
	token, err := ts.Token()
	if err != nil {
		return status.Errorf(codes.Internal, "get token: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(stream.Context(), "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return status.Errorf(codes.Internal, "build request: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return status.Errorf(codes.Internal, "http request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return status.Errorf(codes.Internal, "vertex AI gemini %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	sessionID := req.SessionId
	if sessionID == "" {
		sessionID = generateID("gemini")
	}

	var done bridgepb.DonePayload
	done.SessionId = sessionID
	done.Model = model

	var assistantContent strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		data, ok := strings.CutPrefix(line, "data: ")
		if !ok || data == "[DONE]" {
			continue
		}

		var ev struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
			UsageMetadata struct {
				PromptTokenCount     int `json:"promptTokenCount"`
				CandidatesTokenCount int `json:"candidatesTokenCount"`
			} `json:"usageMetadata"`
		}
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			continue
		}

		for _, candidate := range ev.Candidates {
			for _, p := range candidate.Content.Parts {
				if p.Text != "" {
					assistantContent.WriteString(p.Text)
					if err := stream.Send(&bridgepb.SendResponse{
						Payload: &bridgepb.SendResponse_Chunk{Chunk: p.Text},
					}); err != nil {
						return err
					}
				}
			}
		}

		if ev.UsageMetadata.PromptTokenCount > 0 {
			done.InputTokens = int32(ev.UsageMetadata.PromptTokenCount)
			done.OutputTokens = int32(ev.UsageMetadata.CandidatesTokenCount)
		}
	}

	if err := scanner.Err(); err != nil {
		return status.Errorf(codes.Internal, "stream read: %v", err)
	}

	if assistantContent.Len() > 0 {
		updated := make([]convMsg, len(history), len(history)+2)
		copy(updated, history)
		updated = append(updated, convMsg{Role: "user", Content: req.Prompt}, convMsg{Role: "assistant", Content: assistantContent.String()})
		go saveConvHistory(req.Cwd, sessionID, updated)
	}

	return stream.Send(&bridgepb.SendResponse{
		Payload: &bridgepb.SendResponse_Done{Done: &done},
	})
}

func (s *Server) Shutdown(_ context.Context, _ *bridgepb.ShutdownRequest) (*bridgepb.ShutdownResponse, error) {
	return &bridgepb.ShutdownResponse{}, nil
}
