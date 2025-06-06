package stress

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync/atomic"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/go-errors/errors"
	"github.com/kelseyhightower/envconfig"
	"github.com/sourcegraph/conc/pool"
	"google.golang.org/api/option"
)

type (
	StressorConfig struct {
		Parallelism int    `default:"10"`
		Project     string `default:"media-monetization-p"`
		Secret      string `default:"human_mediaguard_key"`
	}
	Stressor interface {
		Stress(ctx context.Context, config *StressorConfig) error
	}
	StressorImpl struct {
	}
)

const (
	SecretPathFmt = "projects/%s/secrets/%s/versions/latest"
)

var (
	_ Stressor = (*StressorImpl)(nil)
)

// MustDefaultStressorConfig returns a default configuration for the stressor.
func MustDefaultStressorConfig() *StressorConfig {
	ret, err := DefaultStressorConfig()
	if err != nil {
		slog.Error("failed to get default stressor config", "error", err)
		panic(errors.Wrap(err, 0))
	}
	return ret
}

// DefaultStressorConfig returns a default configuration for the stressor.
func DefaultStressorConfig() (*StressorConfig, error) {
	ret := &StressorConfig{}
	if err := envconfig.Process("stress", ret); err != nil {
		return nil, errors.Wrap(err, 0)
	}
	return ret, nil
}

func (s *StressorImpl) Stress(ctx context.Context, config *StressorConfig) error {
	p := pool.New().
		WithMaxGoroutines(config.Parallelism)

	var count, countErrs, duration atomic.Int64
	accessSecretVersionRequest := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf(SecretPathFmt, config.Project, config.Secret),
	}
	stressFunc := func() {
		start := time.Now()
		defer func() {
			duration.Add(time.Since(start).Milliseconds())
		}()
		ctx := context.Background()
		count.Add(1)
		client, err := secretmanager.NewClient(ctx, option.WithEndpoint("dns:///secretmanager.googleapis.com:443"))
		if err != nil {
			countErrs.Add(1)
			slog.ErrorContext(ctx, "failed to create secret manager client", "error", err)
			return
		}
		defer client.Close()
		if _, err = client.AccessSecretVersion(ctx, accessSecretVersionRequest); err != nil {
			countErrs.Add(1)
			slog.ErrorContext(ctx, "failed to access secret", "error", err)
			return
		}
	}
	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "Waiting for all goroutines to finish")
			p.Wait()
			averageDuration := float64(duration.Load()) / float64(count.Load())
			averageDuration = math.Round(averageDuration*100) / 100
			slog.InfoContext(ctx, "stress test completed",
				"count", count.Load(),
				"error_count", countErrs.Load(),
				"duration_ms", duration.Load(),
				"average_duration_ms", averageDuration,
			)
			return nil
		default:
			p.Go(stressFunc)
		}
	}
}
