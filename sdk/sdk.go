// Package sdk provides the building blocks for orcai plugins.
// Plugin authors embed BasePlugin and call sdk.Serve(impl) from main().
package sdk

import (
	"context"

	goplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/adam-stokes/orcai/proto/orcai/v1"
)

var handshake = goplugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "ORCAI_PLUGIN",
	MagicCookieValue: "orcai",
}

// OrcaiPlugin is the interface every native plugin must implement.
// Embed BasePlugin to get default no-op implementations.
type OrcaiPlugin interface {
	GetInfo(context.Context, *pb.Empty) (*pb.PluginInfo, error)
	Start(context.Context, *pb.StartRequest) (*pb.StartResponse, error)
	Stop(context.Context, *pb.Empty) (*pb.Empty, error)
	GetStatus(context.Context, *pb.Empty) (*pb.StatusResponse, error)
}

// BasePlugin provides no-op defaults. Embed it in your plugin struct.
type BasePlugin struct{}

func (b *BasePlugin) Start(_ context.Context, _ *pb.StartRequest) (*pb.StartResponse, error) {
	return &pb.StartResponse{}, nil
}
func (b *BasePlugin) Stop(_ context.Context, _ *pb.Empty) (*pb.Empty, error) {
	return &pb.Empty{}, nil
}
func (b *BasePlugin) GetStatus(_ context.Context, _ *pb.Empty) (*pb.StatusResponse, error) {
	return &pb.StatusResponse{State: "idle"}, nil
}

// Serve starts the go-plugin gRPC server. Call from your plugin's main().
func Serve(impl OrcaiPlugin) {
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: handshake,
		Plugins: map[string]goplugin.Plugin{
			"plugin": &grpcBridge{impl: impl},
		},
		GRPCServer: goplugin.DefaultGRPCServer,
	})
}

// grpcBridge adapts OrcaiPlugin for go-plugin.
type grpcBridge struct {
	goplugin.Plugin
	impl OrcaiPlugin
}

func (p *grpcBridge) GRPCServer(_ *goplugin.GRPCBroker, s *grpc.Server) error {
	pb.RegisterOrcaiPluginServer(s, &grpcAdapter{impl: p.impl})
	return nil
}

func (p *grpcBridge) GRPCClient(_ context.Context, _ *goplugin.GRPCBroker, conn *grpc.ClientConn) (any, error) {
	return pb.NewOrcaiPluginClient(conn), nil
}

type grpcAdapter struct {
	pb.UnimplementedOrcaiPluginServer
	impl OrcaiPlugin
}

func (a *grpcAdapter) GetInfo(ctx context.Context, e *pb.Empty) (*pb.PluginInfo, error) {
	return a.impl.GetInfo(ctx, e)
}
func (a *grpcAdapter) Start(ctx context.Context, r *pb.StartRequest) (*pb.StartResponse, error) {
	return a.impl.Start(ctx, r)
}
func (a *grpcAdapter) Stop(ctx context.Context, e *pb.Empty) (*pb.Empty, error) {
	return a.impl.Stop(ctx, e)
}
func (a *grpcAdapter) GetStatus(ctx context.Context, e *pb.Empty) (*pb.StatusResponse, error) {
	return a.impl.GetStatus(ctx, e)
}

// BusClient lets plugins publish/subscribe to the orcai event bus.
type BusClient struct {
	client pb.EventBusClient
	conn   *grpc.ClientConn
}

// NewBusClient connects to the event bus at addr (provided in StartRequest.BusAddress).
func NewBusClient(addr string) (*BusClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &BusClient{client: pb.NewEventBusClient(conn), conn: conn}, nil
}

// Close releases the bus connection.
func (b *BusClient) Close() error { return b.conn.Close() }

// Publish sends an event to the bus.
func (b *BusClient) Publish(ctx context.Context, topic, source string, payload []byte) error {
	_, err := b.client.Publish(ctx, &pb.Event{Topic: topic, Source: source, Payload: payload})
	return err
}

// Subscribe returns a channel of events matching the given topics.
// The channel closes when ctx is cancelled.
func (b *BusClient) Subscribe(ctx context.Context, topics []string) (<-chan *pb.Event, error) {
	stream, err := b.client.Subscribe(ctx, &pb.SubscribeRequest{Topics: topics})
	if err != nil {
		return nil, err
	}
	ch := make(chan *pb.Event, 64)
	go func() {
		defer close(ch)
		for {
			evt, err := stream.Recv()
			if err != nil {
				return
			}
			select {
			case ch <- evt:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}
