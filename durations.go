package fisk

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var (
	durationMatcher    = regexp.MustCompile(`([-+]?)(([\d\.]+)([a-zA-Z]+))`)
	errInvalidDuration = fmt.Errorf("invalid duration")
)

// ParseDuration parse durations with additional units over those from
// standard go parser.
//
// In addition to normal go parser time units it also supports
// these.
//
// The reason these are not in go standard lib is due to precision around
// how many days in a month and about leap years and leap seconds. This
// function does nothing to try and correct for those.
//
// * "w", "W" - a week based on 7 days of exactly 24 hours
// * "d", "D" - a day based on 24 hours
// * "M" - a month made of 30 days of 24 hours
// * "y", "Y" - a year made of 365 days of 24 hours each
//
// Valid duration strings can be -1y1d1µs
func ParseDuration(d string) (time.Duration, error) {
	var (
		r   time.Duration
		neg = 1
	)

	if len(d) == 0 {
		return r, errInvalidDuration
	}

	parts := durationMatcher.FindAllStringSubmatch(d, -1)
	if len(parts) == 0 {
		return r, errInvalidDuration
	}

	for i, p := range parts {
		if len(p) != 5 {
			return 0, errInvalidDuration
		}

		if i == 0 && p[1] == "-" {
			neg = -1
		}

		switch p[4] {
		case "w", "W":
			val, err := strconv.ParseFloat(p[3], 32)
			if err != nil {
				return 0, fmt.Errorf("%w: %v", errInvalidDuration, err)
			}

			r += time.Duration(val*7*24) * time.Hour

		case "d", "D":
			val, err := strconv.ParseFloat(p[3], 32)
			if err != nil {
				return 0, fmt.Errorf("%w: %v", errInvalidDuration, err)
			}

			r += time.Duration(val*24) * time.Hour

		case "M":
			val, err := strconv.ParseFloat(p[3], 32)
			if err != nil {
				return 0, fmt.Errorf("%w: %v", errInvalidDuration, err)
			}

			r += time.Duration(val*24*30) * time.Hour

		case "Y", "y":
			val, err := strconv.ParseFloat(p[3], 32)
			if err != nil {
				return 0, fmt.Errorf("%w: %v", errInvalidDuration, err)
			}

			r += time.Duration(val*24*365) * time.Hour

		case "ns", "us", "µs", "ms", "s", "m", "h":
			dur, err := time.ParseDuration(p[2])
			if err != nil {
				return 0, fmt.Errorf("%w: %v", errInvalidDuration, err)
			}

			r += dur
		default:
			return 0, fmt.Errorf("%w: invalid unit %v", errInvalidDuration, p[4])
		}
	}

	return time.Duration(neg) * r, nil
}
