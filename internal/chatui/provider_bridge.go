package chatui

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"

	bridgepb "github.com/adam-stokes/orcai/proto/bridgepb"
)

// BridgeProvider implements Provider by forwarding requests over gRPC
// to a ProviderBridge adapter subprocess managed by bridge.Manager.
type BridgeProvider struct {
	mu        sync.Mutex
	client    bridgepb.ProviderBridgeClient
	name      string
	cwd       string
	sessionID string
	model     string
}

// NewBridgeProvider creates a BridgeProvider wrapping the given gRPC client.
func NewBridgeProvider(client bridgepb.ProviderBridgeClient, name, cwd string) *BridgeProvider {
	return &BridgeProvider{client: client, name: name, cwd: cwd}
}

func (p *BridgeProvider) Name() string { return p.name }

func (p *BridgeProvider) SetModel(m string) {
	p.mu.Lock()
	p.model = m
	p.mu.Unlock()
}

func (p *BridgeProvider) Model() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.model
}

// Send opens a bidirectional gRPC Send stream and relays responses via the returned channel.
//
// If the adapter sends a WaitingPayload, a StreamWaiting event is emitted so the caller
// can collect user input. The goroutine then blocks on the returned InputCh until the
// user provides a reply, which is forwarded to the adapter.
func (p *BridgeProvider) Send(ctx context.Context, _ []message, text string) <-chan StreamEvent {
	ch := make(chan StreamEvent, 64)

	p.mu.Lock()
	sid := p.sessionID
	mdl := p.model
	p.mu.Unlock()

	go func() {
		defer close(ch)

		stream, err := p.client.Send(ctx)
		if err != nil {
			ch <- StreamEvent{Err: &StreamErr{Err: err.Error()}}
			return
		}

		// Send the initial request.
		if err := stream.Send(&bridgepb.SendRequest{
			Prompt:    text,
			SessionId: sid,
			Cwd:       p.cwd,
			Model:     mdl,
		}); err != nil {
			ch <- StreamEvent{Err: &StreamErr{Err: err.Error()}}
			return
		}

		var done StreamDone
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				ch <- StreamEvent{Err: &StreamErr{Err: err.Error()}}
				return
			}

			switch payload := resp.Payload.(type) {
			case *bridgepb.SendResponse_Chunk:
				chunkText := payload.Chunk
				if strings.HasPrefix(chunkText, "\x02") {
					// Status event encoded as \x02{"tool":"...","input":"..."}
					var ev struct {
						Tool  string `json:"tool"`
						Input string `json:"input"`
					}
					if err := json.Unmarshal([]byte(chunkText[1:]), &ev); err == nil {
						ch <- StreamEvent{Status: &StreamStatus{Tool: ev.Tool, Input: ev.Input}}
					}
					continue
				}
				ch <- StreamEvent{Chunk: &StreamChunk{Text: chunkText}}

			case *bridgepb.SendResponse_Done:
				d := payload.Done
				done = StreamDone{
					SessionID:     d.SessionId,
					Model:         d.Model,
					InputTokens:   int(d.InputTokens),
					CacheTokens:   int(d.CacheTokens),
					OutputTokens:  int(d.OutputTokens),
					ContextWindow: int(d.ContextWindow),
				}
				p.mu.Lock()
				p.sessionID = d.SessionId
				p.mu.Unlock()

			case *bridgepb.SendResponse_Waiting:
				// Adapter subprocess needs user input. Signal the caller to
				// collect a reply, then block until it arrives.
				inputCh := make(chan string, 1)
				ch <- StreamEvent{Waiting: &StreamWaiting{Hint: payload.Waiting.Hint, InputCh: inputCh}}
				input := <-inputCh
				// Forward the reply to the adapter.
				if err := stream.Send(&bridgepb.SendRequest{Input: input}); err != nil {
					ch <- StreamEvent{Err: &StreamErr{Err: err.Error()}}
					return
				}

			case *bridgepb.SendResponse_Error:
				ch <- StreamEvent{Err: &StreamErr{Err: payload.Error}}
				return
			}
		}
		ch <- StreamEvent{Done: &done}
	}()

	return ch
}

// SetSession sets the session ID to resume on the next Send.
func (p *BridgeProvider) SetSession(id string) {
	p.mu.Lock()
	p.sessionID = id
	p.mu.Unlock()
}

// SessionID returns the current session ID.
func (p *BridgeProvider) SessionID() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.sessionID
}

func (p *BridgeProvider) Close() {}
