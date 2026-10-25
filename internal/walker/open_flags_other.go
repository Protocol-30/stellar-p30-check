//go:build !linux && !darwin && !freebsd && !openbsd && !netbsd && !dragonfly

package walker

import "os"

// Other platforms retain rooted access and type checks, but do not provide the
// Unix no-follow/nonblocking flags through this helper. Do not scan concurrently
// modified trees on these platforms.
func sourceOpenFlags() int    { return os.O_RDONLY }
func directoryOpenFlags() int { return os.O_RDONLY }
