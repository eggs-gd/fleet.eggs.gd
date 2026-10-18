package taskprovider

import (
	"os"
	"time"
)

type cardStamp struct {
	modTime time.Time
	size    int64
}

// cardChanged reports whether path's file differs from the state recorded at
// its last successful load (a card never loaded counts as changed), and
// returns that state so the caller can record it once the load succeeds. An
// unreadable path reports unchanged.
func (b *Board) cardChanged(path string) (cardStamp, bool) {
	if b == nil || path == "" {
		return cardStamp{}, false
	}
	info, err := os.Stat(path)
	if err != nil {
		return cardStamp{}, false
	}
	stamp := cardStamp{modTime: info.ModTime(), size: info.Size()}
	b.mu.Lock()
	defer b.mu.Unlock()
	previous, ok := b.cardStamps[path]
	return stamp, !ok || previous != stamp
}

// recordCard remembers the file state a card was successfully loaded at.
func (b *Board) recordCard(path string, stamp cardStamp) {
	if b == nil || path == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cardStamps == nil {
		b.cardStamps = map[string]cardStamp{}
	}
	b.cardStamps[path] = stamp
}

// recordCardFile records path's current state; for loads that have no earlier
// cardChanged check (Hydrate, an explicit reload).
func (b *Board) recordCardFile(path string) {
	if stamp, _ := b.cardChanged(path); stamp != (cardStamp{}) {
		b.recordCard(path, stamp)
	}
}
