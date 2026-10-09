package execution

import (
	"errors"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

// ParseSchedule accepts "@every <duration>" (e.g. "@every 2h", "@every 30m")
// or a plain integer number of seconds.
func ParseSchedule(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("execution: schedule is required")
	}
	if rest, ok := strings.CutPrefix(s, "@every "); ok {
		d, err := time.ParseDuration(strings.TrimSpace(rest))
		if err != nil {
			return 0, err
		}
		if d <= 0 {
			return 0, errors.New("execution: schedule must be positive")
		}
		return d, nil
	}
	secs, err := strconv.ParseInt(s, 10, 64)
	if err != nil || secs <= 0 {
		return 0, errors.New("execution: schedule must be '@every <duration>' or seconds")
	}
	return time.Duration(secs) * time.Second, nil
}

// NextRun returns the next run time: interval from `from`, plus/minus jitter.
// Jitter spreads bots so scheduled activity does not land in one burst.
func NextRun(rng *rand.Rand, from time.Time, interval, jitter time.Duration) time.Time {
	if jitter < 0 {
		jitter = 0
	}
	var delta time.Duration
	if jitter > 0 && rng != nil {
		delta = time.Duration(rng.Int63n(int64(2*jitter)+1)) - jitter
	}
	next := from.Add(interval + delta)
	if next.Before(from) {
		return from
	}
	return next
}
