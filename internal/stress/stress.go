package stress

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/go-errors/errors"
	"github.com/kelseyhightower/envconfig"
	"github.com/panjf2000/ants"
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
	pool, err := ants.NewPool(config.Parallelism)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer pool.Release()

	var count, countErrs atomic.Int64
	accessSecretVersionRequest := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf(SecretPathFmt, config.Project, config.Secret),
	}
	stressFunc := func() {
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
			slog.ErrorContext(ctx, "failed to create secret manager client", "error", err)
			return
		}
	}
	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "stress test completed",
				"count", count.Load(),
				"error_count", countErrs.Load(),
			)
			return nil
		default:
			if err = pool.Submit(stressFunc); err != nil {
				return errors.Wrap(err, 0)
			}
		}
	}
}
