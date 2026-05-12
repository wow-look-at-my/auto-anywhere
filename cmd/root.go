package cmd

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/wow-look-at-my/auto-anywhere/proxy"
)

var (
	port     int
	upstream string
	verbose  bool
)

var rootCmd = &cobra.Command{
	Use:   "auto-anywhere",
	Short: "Anthropic API proxy that enables auto mode and thinking summaries",
	Long: `auto-anywhere is a reverse proxy for the Anthropic API that:
  - Forces adaptive thinking with summaries on all requests
  - Injects auto-mode feature flags for Opus 4.6 on Claude Max

Point Claude Code at it with ANTHROPIC_BASE_URL=http://localhost:PORT`,
	RunE: func(cmd *cobra.Command, args []string) error {
		level := slog.LevelInfo
		if verbose {
			level = slog.LevelDebug
		}
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

		p, err := proxy.New(proxy.Config{Upstream: upstream})
		if err != nil {
			return fmt.Errorf("creating proxy: %w", err)
		}

		addr := fmt.Sprintf(":%d", port)
		slog.Info("starting proxy", "addr", addr, "upstream", upstream)
		return http.ListenAndServe(addr, p)
	},
}

func init() {
	rootCmd.Flags().IntVarP(&port, "port", "p", 18080, "Listen port")
	rootCmd.Flags().StringVarP(&upstream, "upstream", "u", "https://api.anthropic.com", "Upstream API URL")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
