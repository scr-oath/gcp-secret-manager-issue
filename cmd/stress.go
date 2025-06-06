package cmd

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/scr-oath/gcp-secret-manager-issue/internal/stress"
	"github.com/spf13/cobra"
)

var stressorConfig = stress.MustDefaultStressorConfig()

// stressCmd represents the stress command
var stressCmd = &cobra.Command{
	Use:   "stress",
	Short: "Stress test the secret manager",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
		defer cancel()
		slog.DebugContext(ctx, "stress called")
		stressor := stress.NewStressor()
		if err := stressor.Stress(ctx, stressorConfig); err != nil {
			slog.ErrorContext(ctx, "stress test failed", "error", err)
			os.Exit(1)
		}
		slog.InfoContext(ctx, "stress test completed successfully")
	},
}

func init() {
	rootCmd.AddCommand(stressCmd)

	flags := stressCmd.Flags()
	flags.IntVarP(&stressorConfig.Parallelism, "parallelism", "p", stressorConfig.Parallelism, "parallelism")
	flags.StringVarP(&stressorConfig.Project, "project", "P", stressorConfig.Project, "GCP project ID")
	flags.StringVarP(&stressorConfig.Secret, "secret", "s", stressorConfig.Secret, "GCP secret name")
}
