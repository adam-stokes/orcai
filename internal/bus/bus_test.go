package bus_test

import (
	"context"
	"testing"
	"time"

	"github.com/adam-stokes/orcai/internal/bus"
	pb "github.com/adam-stokes/orcai/proto/orcai/v1"
)

func TestMatchTopic(t *testing.T) {
	cases := []struct {
		pattern string
		topic   string
		want    bool
	}{
		{"*", "session.started", true},
		{"session.*", "session.started", true},
		{"session.*", "session.stopped", true},
		{"session.*", "git.commit", false},
		{"git.commit", "git.commit", true},
		{"git.commit", "git.push", false},
	}
	for _, tc := range cases {
		got := bus.MatchTopic(tc.pattern, tc.topic)
		if got != tc.want {
			t.Errorf("MatchTopic(%q, %q) = %v, want %v", tc.pattern, tc.topic, got, tc.want)
		}
	}
}

func TestPublishSubscribe(t *testing.T) {
	srv := bus.New()
	addr, err := srv.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Connect a subscriber
	ch, cleanup, err := bus.TestSubscribe(ctx, addr, []string{"session.*"})
	if err != nil {
		t.Fatalf("TestSubscribe: %v", err)
	}
	defer cleanup()

	// Publish an event
	if err := bus.TestPublish(ctx, addr, &pb.Event{
		Topic:   "session.started",
		Source:  "test",
		Payload: []byte(`{"name":"claude-1"}`),
	}); err != nil {
		t.Fatalf("TestPublish: %v", err)
	}

	select {
	case evt := <-ch:
		if evt.Topic != "session.started" {
			t.Errorf("got topic %q, want %q", evt.Topic, "session.started")
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for event")
	}
}
