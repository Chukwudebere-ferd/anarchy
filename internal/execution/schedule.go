package execution

import (
	"errors"
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
