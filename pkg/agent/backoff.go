package agent

import (
	"context"
	"time"
)

const (
	defaultRetryMinInterval = time.Second
	defaultRetryMaxInterval = 30 * time.Second
)

// RetryDelay is exponential backoff: min, 2min, 4min, ... capped at max.
// failCount is 1 after the first failure.
func RetryDelay(failCount int, min, max time.Duration) time.Duration {
	if min <= 0 {
		min = defaultRetryMinInterval
	}
	if max <= 0 {
		max = defaultRetryMaxInterval
	}
	if max < min {
		max = min
	}
	if failCount < 1 {
		failCount = 1
	}
	d := min
	for i := 1; i < failCount; i++ {
		if d > max/2 {
			return max
		}
		d *= 2
	}
	if d > max {
		return max
	}
	return d
}

func waitBackoff(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (c Config) retryMin() time.Duration {
	if c.RetryMinInterval > 0 {
		return c.RetryMinInterval
	}
	return defaultRetryMinInterval
}

func (c Config) retryMax() time.Duration {
	if c.RetryMaxInterval > 0 {
		return c.RetryMaxInterval
	}
	return defaultRetryMaxInterval
}
