package agent

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"
)

// MemoryEntry is one persisted fact/correction the agent remembers across
// sessions. type "fact" = user preference/info; "correction" = user told the
// agent it was wrong / wants something else.
type MemoryEntry struct {
	Type      string    `json:"type"` // "fact" | "correction"
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	SessionID string    `json:"session_id,omitempty"`
}

// MemoryStore persists agent memory to a JSON file so it survives restarts and
// spans sessions. ponytail: single-file JSON, no DB; swap for internaldb if you
// need multi-user scoping or queries.
// Cross-process safe: os-level flock (Linux) so multiple containers writing the
// same mounted file won't corrupt it. On non-Linux (dev/Windows) the per-process
// mutex still serializes within one process.
type MemoryStore struct {
	mu   sync.Mutex
	path string
}

// NewMemoryStore opens (or creates) the memory file at path.
func NewMemoryStore(path string) *MemoryStore {
	return &MemoryStore{path: path}
}

func (m *MemoryStore) load() []MemoryEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, err := os.ReadFile(m.path)
	if err != nil {
		return nil
	}
	var entries []MemoryEntry
	if json.Unmarshal(data, &entries) == nil {
		return entries
	}
	return nil
}

// Remember appends an entry.
func (m *MemoryStore) Remember(typ, content, sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_ = m.withLock(func(f *os.File) error {
		var entries []MemoryEntry
		if data, err := os.ReadFile(m.path); err == nil {
			json.Unmarshal(data, &entries)
		}
		entries = append(entries, MemoryEntry{
			Type:      typ,
			Content:   content,
			CreatedAt: time.Now(),
			SessionID: sessionID,
		})
		// ponytail: keep last 200 entries; trim oldest.
		if len(entries) > 200 {
			entries = entries[len(entries)-200:]
		}
		data, _ := json.MarshalIndent(entries, "", "  ")
		return os.WriteFile(m.path, data, 0644)
	})
}

// ContextPrompt renders the last N entries as a system-context block for the
// LLM. Empty string when no memory yet.
func (m *MemoryStore) ContextPrompt(n int) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var entries []MemoryEntry
	_ = m.withLock(func(f *os.File) error {
		data, err := os.ReadFile(m.path)
		if err != nil {
			return err
		}
		return json.Unmarshal(data, &entries)
	})
	if len(entries) == 0 {
		return ""
	}
	if n > len(entries) {
		n = len(entries)
	}
	recent := entries[len(entries)-n:]
	var b strings.Builder
	b.WriteString("\n---\nMEMORY (persistent, across sessions):\n")
	for _, e := range recent {
		b.WriteString("- [" + e.Type + "] " + e.Content + "\n")
	}
	return b.String()
}

// isCorrection heuristically detects a user correction/self-improvement signal.
func isCorrection(msg string) bool {
	low := strings.ToLower(msg)
	for _, kw := range []string{"falsch", "wrong", "nein,", "stattdessen", "korrigier", "correct", "actually", "instead", "that's not", "das stimmt nicht", "nicht so"} {
		if strings.Contains(low, kw) {
			return true
		}
	}
	return false
}
