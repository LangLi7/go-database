//go:build linux

package agent

import (
	"os"
	"syscall"
)

// withLock opens path (creating it), takes an exclusive OS flock (blocking),
// runs fn, then closes. Serializes writers across processes — so multiple
// containers mounting the same memory file won't corrupt it.
func (m *MemoryStore) withLock(fn func(f *os.File) error) error {
	f, err := os.OpenFile(m.path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return fn(f)
}
