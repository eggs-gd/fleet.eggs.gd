package runtimedb

import (
	"fmt"
	"strings"
	"sync"
)

// Relay is the manager-relay debug log. Writes buffer until 64KB or Close.
type Relay struct {
	mu      sync.Mutex
	root    string
	pending strings.Builder
	closed  bool
}

// OpenRelay appends to the single relay_log row for root.
func OpenRelay(root string) (*Relay, error) {
	if _, err := Open(root); err != nil {
		return nil, err
	}
	return &Relay{root: root}, nil
}

func (r *Relay) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return 0, fmt.Errorf("relay log is closed")
	}
	r.pending.Write(p)
	if r.pending.Len() < logFlushBytes {
		return len(p), nil
	}
	if err := r.flushLocked(); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (r *Relay) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil
	}
	err := r.flushLocked()
	if err != nil {
		return err
	}
	r.closed = true
	return nil
}

func (r *Relay) flushLocked() error {
	if r.pending.Len() == 0 {
		return nil
	}
	chunk := r.pending.String()
	r.pending.Reset()
	if err := appendRelay(r.root, chunk); err != nil {
		r.pending.WriteString(chunk)
		return err
	}
	return nil
}

func appendRelay(root, chunk string) error {
	db, err := Open(root)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
INSERT INTO relay_log(id, body) VALUES(1, ?)
ON CONFLICT(id) DO UPDATE SET body = relay_log.body || excluded.body`, chunk)
	if err != nil {
		return fmt.Errorf("append relay log: %w", err)
	}
	return nil
}

func relayBody(root string) (string, error) {
	db, err := Open(root)
	if err != nil {
		return "", err
	}
	var body string
	err = db.QueryRow(`SELECT body FROM relay_log WHERE id = 1`).Scan(&body)
	if err != nil {
		return "", err
	}
	return body, nil
}
