//go:build appengine || (!linux && !freebsd && !darwin && !dragonfly && !netbsd && !openbsd)
// +build appengine !linux,!freebsd,!darwin,!dragonfly,!netbsd,!openbsd

package fisk

import "io"

func guessWidth(w io.Writer) int {
	return 80
}
