package copilot

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	bridgepb "github.com/adam-stokes/orcai/proto/bridgepb"
)

const (
	version     = "0.1.0"
	waitTimeout = 300 * time.Millisecond
)

// Server implements the ProviderBridge gRPC service for the Copilot CLI.
type Server struct {
	bridgepb.UnimplementedProviderBridgeServer
	cwd string
}

// New creates a new Copilot adapter server.
func New(cwd string) *Server {
	return &Server{cwd: cwd}
}

// Register registers the server with a gRPC server instance.
func Register(s *grpc.Server, cwd string) {
	bridgepb.RegisterProviderBridgeServer(s, New(cwd))
}

// Describe returns provider metadata. Copilot has no shared skills/agents convention.
func (s *Server) Describe(_ context.Context, _ *bridgepb.DescribeRequest) (*bridgepb.DescribeResponse, error) {
	return &bridgepb.DescribeResponse{
		Name:    "Copilot",
		Version: version,
	}, nil
}

// Send executes a copilot CLI subprocess and streams the response.
// The first SendRequest contains the prompt; subsequent SendRequests with a
// non-empty Input field are written to the subprocess stdin.
//
// Waiting detection: Copilot CLI uses plain text output. If the subprocess
// produces a partial line (no trailing newline) and then goes quiet for
// waitTimeout, a WaitingPayload is sent so the user can type a reply.
func (s *Server) Send(stream bridgepb.ProviderBridge_SendServer) error {
	req, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.Internal, "recv initial request: %v", err)
	}

	args := []string{
		"-p", req.Prompt,
		"--yolo",
	}
	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}
	if req.SessionId != "" {
		args = append(args, "--resume="+req.SessionId)
	}

	cmd := exec.CommandContext(stream.Context(), "copilot", args...)
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
		return status.Errorf(codes.Internal, "start copilot: %v", err)
	}

	// inputCh receives mid-stream user input from the gRPC stream.
	inputCh := make(chan string, 4)
	go func() {
		for {
			r, err := stream.Recv()
			if err != nil {
				return
			}
			if r.Input != "" {
				inputCh <- r.Input
			}
		}
	}()

	// readCh receives raw bytes from the subprocess stdout.
	type readResult struct {
		data []byte
		err  error
	}
	readCh := make(chan readResult, 32)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, e := stdout.Read(buf)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				readCh <- readResult{data: data}
			}
			if e != nil {
				readCh <- readResult{err: e}
				return
			}
		}
	}()

	var accumulator []byte
	waitingSent := false
	timer := time.NewTimer(waitTimeout)
	defer timer.Stop()

	for {
		select {
		case result := <-readCh:
			if result.err != nil {
				// EOF — flush pending text.
				if len(accumulator) > 0 {
					_ = stream.Send(&bridgepb.SendResponse{
						Payload: &bridgepb.SendResponse_Chunk{Chunk: string(accumulator)},
					})
				}
				goto done
			}
			accumulator = append(accumulator, result.data...)
			waitingSent = false
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(waitTimeout)
			// Flush complete lines; keep any partial line as a potential prompt.
			for {
				idx := bytes.IndexByte(accumulator, '\n')
				if idx < 0 {
					break
				}
				line := strings.TrimRight(string(accumulator[:idx+1]), "\r\n")
				if line != "" {
					if err := stream.Send(&bridgepb.SendResponse{
						Payload: &bridgepb.SendResponse_Chunk{Chunk: line + "\n"},
					}); err != nil {
						return err
					}
				}
				accumulator = accumulator[idx+1:]
			}

		case <-timer.C:
			// Subprocess is quiet. A pending partial line suggests it's waiting.
			if !waitingSent && len(accumulator) > 0 && cmd.ProcessState == nil {
				hint := strings.TrimSpace(string(accumulator))
				accumulator = nil
				if err := stream.Send(&bridgepb.SendResponse{
					Payload: &bridgepb.SendResponse_Waiting{Waiting: &bridgepb.WaitingPayload{Hint: hint}},
				}); err != nil {
					return err
				}
				waitingSent = true
			}
			timer.Reset(waitTimeout)

		case input := <-inputCh:
			waitingSent = false
			_, _ = stdin.Write([]byte(input + "\n"))
		}
	}

done:
	if err := cmd.Wait(); err != nil {
		return status.Errorf(codes.Internal, "copilot exited: %v", err)
	}

	return stream.Send(&bridgepb.SendResponse{
		Payload: &bridgepb.SendResponse_Done{Done: &bridgepb.DonePayload{
			SessionId: req.SessionId,
		}},
	})
}

// Shutdown is a no-op — the process will exit when the manager kills it.
func (s *Server) Shutdown(_ context.Context, _ *bridgepb.ShutdownRequest) (*bridgepb.ShutdownResponse, error) {
	return &bridgepb.ShutdownResponse{}, nil
}
