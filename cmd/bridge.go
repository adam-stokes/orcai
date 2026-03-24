package cmd

import (
	"fmt"
	"net"
	"os"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/adam-stokes/orcai/internal/adapters/claude"
	"github.com/adam-stokes/orcai/internal/adapters/copilot"
	"github.com/adam-stokes/orcai/internal/adapters/gemini"
)

var bridgeCmd = &cobra.Command{
	Use:    "bridge",
	Short:  "Provider bridge subcommands (internal)",
	Hidden: true,
}

var bridgeClaudeCmd = &cobra.Command{
	Use:   "claude",
	Short: "Run the Claude provider gRPC adapter",
	RunE:  runBridgeClaude,
}

var bridgeGeminiCmd = &cobra.Command{
	Use:   "gemini",
	Short: "Run the Gemini provider gRPC adapter",
	RunE:  runBridgeGemini,
}

var bridgeCopilotCmd = &cobra.Command{
	Use:   "copilot",
	Short: "Run the Copilot provider gRPC adapter",
	RunE:  runBridgeCopilot,
}

var bridgeSocket string
var bridgeCwd string

func init() {
	for _, cmd := range []*cobra.Command{bridgeClaudeCmd, bridgeGeminiCmd, bridgeCopilotCmd} {
		cmd.Flags().StringVar(&bridgeSocket, "socket", "", "Unix socket path (required)")
		cmd.Flags().StringVar(&bridgeCwd, "cwd", "", "Working directory (required)")
		_ = cmd.MarkFlagRequired("socket")
		_ = cmd.MarkFlagRequired("cwd")
	}

	bridgeCmd.AddCommand(bridgeClaudeCmd, bridgeGeminiCmd, bridgeCopilotCmd)
	rootCmd.AddCommand(bridgeCmd)
}

func runBridgeClaude(_ *cobra.Command, _ []string) error {
	lis, err := net.Listen("unix", bridgeSocket)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", bridgeSocket, err)
	}
	defer os.Remove(bridgeSocket)

	s := grpc.NewServer()
	claude.Register(s, bridgeCwd)

	return s.Serve(lis)
}

func runBridgeGemini(_ *cobra.Command, _ []string) error {
	lis, err := net.Listen("unix", bridgeSocket)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", bridgeSocket, err)
	}
	defer os.Remove(bridgeSocket)

	s := grpc.NewServer()
	gemini.Register(s, bridgeCwd)

	return s.Serve(lis)
}

func runBridgeCopilot(_ *cobra.Command, _ []string) error {
	lis, err := net.Listen("unix", bridgeSocket)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", bridgeSocket, err)
	}
	defer os.Remove(bridgeSocket)

	s := grpc.NewServer()
	copilot.Register(s, bridgeCwd)

	return s.Serve(lis)
}
