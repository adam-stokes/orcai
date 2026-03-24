package bus

import (
	"context"
	"net"
	"strings"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/adam-stokes/orcai/proto/orcai/v1"
)

// Server is the orcai event bus gRPC server.
type Server struct {
	pb.UnimplementedEventBusServer
	mu          sync.RWMutex
	subscribers map[string][]chan *pb.Event
	grpcSrv     *grpc.Server
}

// New creates a new event bus server.
func New() *Server {
	return &Server{
		subscribers: make(map[string][]chan *pb.Event),
	}
}

// Listen starts the gRPC server on addr (e.g. "127.0.0.1:0" for random port).
// Returns the actual address the server is listening on.
func (s *Server) Listen(addr string) (string, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return "", err
	}
	s.grpcSrv = grpc.NewServer()
	pb.RegisterEventBusServer(s.grpcSrv, s)
	go s.grpcSrv.Serve(lis) //nolint:errcheck
	return lis.Addr().String(), nil
}

// Stop shuts down the gRPC server.
func (s *Server) Stop() {
	if s.grpcSrv != nil {
		s.grpcSrv.GracefulStop()
	}
}

func (s *Server) Subscribe(req *pb.SubscribeRequest, stream pb.EventBus_SubscribeServer) error {
	ch := make(chan *pb.Event, 64)
	s.mu.Lock()
	for _, topic := range req.Topics {
		s.subscribers[topic] = append(s.subscribers[topic], ch)
	}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		for _, topic := range req.Topics {
			subs := s.subscribers[topic]
			for i, c := range subs {
				if c == ch {
					s.subscribers[topic] = append(subs[:i], subs[i+1:]...)
					break
				}
			}
		}
		s.mu.Unlock()
	}()

	for {
		select {
		case evt := <-ch:
			if err := stream.Send(evt); err != nil {
				return err
			}
		case <-stream.Context().Done():
			return nil
		}
	}
}

func (s *Server) Publish(ctx context.Context, evt *pb.Event) (*pb.Empty, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for pattern, chs := range s.subscribers {
		if MatchTopic(pattern, evt.Topic) {
			for _, ch := range chs {
				select {
				case ch <- evt:
				default:
					// Subscriber channel full — event dropped intentionally.
					// Slow subscribers do not block the bus.
				}
			}
		}
	}
	return &pb.Empty{}, nil
}

// MatchTopic returns true if pattern matches topic.
// Supports exact match and wildcard suffix ("session.*" matches "session.started").
func MatchTopic(pattern, topic string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == topic {
		return true
	}
	if prefix, ok := strings.CutSuffix(pattern, ".*"); ok {
		return strings.HasPrefix(topic, prefix+".")
	}
	return false
}

// TestSubscribe is a test helper that connects a subscriber to addr.
// Returns a channel of events, a cleanup func, and any error.
func TestSubscribe(ctx context.Context, addr string, topics []string) (<-chan *pb.Event, func(), error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	client := pb.NewEventBusClient(conn)
	stream, err := client.Subscribe(ctx, &pb.SubscribeRequest{Topics: topics})
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	ch := make(chan *pb.Event, 64)
	go func() {
		defer close(ch)
		for {
			evt, err := stream.Recv()
			if err != nil {
				return
			}
			ch <- evt
		}
	}()
	return ch, func() { conn.Close() }, nil
}

// TestPublish is a test helper that publishes an event to addr.
func TestPublish(ctx context.Context, addr string, evt *pb.Event) error {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = pb.NewEventBusClient(conn).Publish(ctx, evt)
	return err
}
