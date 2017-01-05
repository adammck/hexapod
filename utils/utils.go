package utils

import (
	"math"
	"sync"
	"time"
)

func Deg(rads float64) float64 {
	return rads / (math.Pi / 180)
}

func Rad(degrees float64) float64 {
	return (math.Pi / 180) * degrees
}

type FrameCounter struct {
	sync.Mutex

	// Duration at which the counter should be reset.
	interval time.Duration

	// Number of times which Frame has been called during the current interval.
	// TODO: Use a ring here to track the average of the last few intervals.
	count int

	// Number of times which Frame was called during the previous
	prevCount int

	// Time at which Frame was last called, rounded to the interval.
	prevTime time.Time
}

func NewFrameCounter(d time.Duration) *FrameCounter {
	return &FrameCounter{
		interval: d,
	}
}

// Frame increments the frame counter for the interval into which the given time
// falls. Should be called once per frame.
func (fc *FrameCounter) Frame(t time.Time) {
	fc.Lock()
	defer fc.Unlock()

	// If t is a new interval, reset the counter.
	tt := t.Truncate(fc.interval)
	if tt != fc.prevTime {
		fc.prevTime = tt
		fc.prevCount = fc.count
		fc.count = 0
	}

	// Not using sync/atomic since we acquired the lock above anyway.
	fc.count += 1
}

// Count returns the number of times Frame was called during the previous
// interval. Returns junk if Frame is not being called frequently.
func (fc *FrameCounter) Count() int {
	return fc.prevCount
}
