package taskprovider

import (
	"os"
	"time"
)

type cardStamp struct {
	modTime time.Time
	size    int64
}

// stampCard records path's current file state and reports whether it differs
// from the last recorded one (a card seen for the first time counts as
// changed). An unreadable path reports false and records nothing.
func (b *Board) stampCard(path string) bool {
	if b == nil || path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	stamp := cardStamp{modTime: info.ModTime(), size: info.Size()}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cardStamps == nil {
		b.cardStamps = map[string]cardStamp{}
	}
	if previous, ok := b.cardStamps[path]; ok && previous == stamp {
		return false
	}
	b.cardStamps[path] = stamp
	return true
}
