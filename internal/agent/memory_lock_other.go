//go:build !linux

package agent

import "os"

// withLock opens path (creating it) and runs fn. On non-Linux (dev/Windows)
// there is no cross-process flock; the per-process mutex in Remember/
// ContextPrompt still serializes, and multi-container deployments target Linux
// (where memory_lock_linux.go provides the OS-level lock).
func (m *MemoryStore) withLock(fn func(f *os.File) error) error {
	f, err := os.OpenFile(m.path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	return fn(f)
}
