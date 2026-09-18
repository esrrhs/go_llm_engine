package agent

import (
	"testing"
	"time"
)

func TestRetryDelay_CapsAtMax(t *testing.T) {
	min := time.Second
	max := 30 * time.Second
	var got []time.Duration
	for i := 1; i <= 10; i++ {
		got = append(got, RetryDelay(i, min, max))
	}
	want := []time.Duration{
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
		16 * time.Second,
		30 * time.Second,
		30 * time.Second,
		30 * time.Second,
		30 * time.Second,
		30 * time.Second,
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("failCount=%d delay=%s want=%s", i+1, got[i], want[i])
		}
	}
}
