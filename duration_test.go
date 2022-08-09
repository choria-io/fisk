package fisk

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDurationParser(t *testing.T) {
	cases := []struct {
		s   string
		d   time.Duration
		err error
	}{
		{"-1m1d", (time.Minute + (24 * time.Hour)) * -1, nil},
		{"1m1.1w", (184 * time.Hour) + time.Minute, nil},
		{"1M", 24 * 30 * time.Hour, nil},
		{"1Y1M", (365 * 24 * time.Hour) + (24 * 30 * time.Hour), nil},
		{"1xX", 0, fmt.Errorf("%w: invalid unit xX", errInvalidDuration)},
		{"-1", 0, fmt.Errorf("invalid duration")},
		{"0", 0, nil},
	}

	for _, c := range cases {
		d, err := ParseDuration(c.s)
		if c.err == nil {
			assert.NoError(t, err, c.s)
		} else {
			assert.ErrorContains(t, err, c.err.Error(), c.s)
		}
		assert.Equal(t, c.d, d, c.s)
	}
}
