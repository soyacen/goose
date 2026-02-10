package bbr

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestBBR_Allow(t *testing.T) {
	t.Run("initially_allowed", func(t *testing.T) {
		l := defaultOptions().init().newLimiter()
		done, err := l.Allow()
		assert.NoError(t, err)
		assert.NotNil(t, done)
		done(DoneInfo{})
	})

	t.Run("limit_exceeded", func(t *testing.T) {
		// Mock CPU threshold to trigger dropping
		l := &bbrLimiter{
			conf: &options{
				Window:       time.Second,
				Buckets:      10,
				CPUThreshold: 0.5,
			},
			passStat: newRollingCounter(time.Second, 10, false),
			rtStat:   newRollingCounter(time.Second, 10, true),
			cpu:      func() float64 { return 0.8 }, // Above threshold
		}

		// Fill some successes to avoid initial "allow"
		for i := 0; i < 100; i++ {
			done, _ := l.Allow()
			if done != nil {
				done(DoneInfo{})
			}
		}

		// Artificially increase inflight
		l.inflight = 1000

		_, err := l.Allow()
		assert.Error(t, err)
		assert.Equal(t, codes.ResourceExhausted, status.Code(err))
	})
}

func TestRollingCounter(t *testing.T) {
	c := newRollingCounter(time.Millisecond*100, 10, false)
	c.Add(10)
	assert.Equal(t, int64(10), c.Max())

	time.Sleep(time.Millisecond * 150)
	c.Add(5)
	// Old data should be rotated out
	assert.Equal(t, int64(5), c.Max())
}
