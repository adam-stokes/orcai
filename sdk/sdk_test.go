package sdk_test

import (
	"context"
	"testing"

	"github.com/adam-stokes/orcai/sdk"
	pb "github.com/adam-stokes/orcai/proto/orcai/v1"
)

type testPlugin struct {
	sdk.BasePlugin
}

func (p *testPlugin) GetInfo(_ context.Context, _ *pb.Empty) (*pb.PluginInfo, error) {
	return &pb.PluginInfo{Name: "test", Version: "0.1.0"}, nil
}

func TestBasePlugin_DefaultStop(t *testing.T) {
	p := &testPlugin{}
	resp, err := p.Stop(context.Background(), &pb.Empty{})
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if resp == nil {
		t.Error("expected non-nil response")
	}
}

func TestBasePlugin_DefaultStatus(t *testing.T) {
	p := &testPlugin{}
	resp, err := p.GetStatus(context.Background(), &pb.Empty{})
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if resp.State != "idle" {
		t.Errorf("expected state 'idle', got %q", resp.State)
	}
}
