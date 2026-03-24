// Package shared provides utilities shared across provider bridge adapters.
package shared

import (
	"io"

	bridgepb "github.com/adam-stokes/orcai/proto/bridgepb"
)

// BidiStream is the common interface for a bidirectional ProviderBridge_SendServer.
type BidiStream interface {
	Send(*bridgepb.SendResponse) error
	Recv() (*bridgepb.SendRequest, error)
}

// RelayStdin starts a background goroutine that reads mid-stream SendRequests
// from the gRPC stream and writes their Input field to the subprocess stdin.
// Call this after starting the subprocess but before entering the output loop.
func RelayStdin(stream BidiStream, stdin io.WriteCloser) {
	go func() {
		for {
			req, err := stream.Recv()
			if err != nil {
				return
			}
			if req.Input != "" {
				_, _ = stdin.Write([]byte(req.Input + "\n"))
			}
		}
	}()
}
