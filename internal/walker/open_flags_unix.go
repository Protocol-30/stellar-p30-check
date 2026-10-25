//go:build linux || darwin || freebsd || openbsd || netbsd || dragonfly

package walker

import (
	"os"
	"syscall"
)

// Nonblocking opens prevent a concurrently substituted FIFO from hanging the
// scan. NOFOLLOW rejects final-component symlink replacement; os.Root constrains
// intermediate components to the opened tree.
func sourceOpenFlags() int {
	return os.O_RDONLY | syscall.O_NONBLOCK | syscall.O_NOFOLLOW
}

func directoryOpenFlags() int {
	return sourceOpenFlags() | syscall.O_DIRECTORY
}
