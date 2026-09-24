package markdown

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/lib/chain"
)

// FileEvent is a Markdown/filesystem change notification.
type FileEvent struct {
	Path    string
	ModTime time.Time
	Size    int64
	Kind    FileEventKind
}

// FileEventKind classifies walker emissions.
type FileEventKind string

const (
	FileEventInitial  FileEventKind = "initial"
	FileEventCreated  FileEventKind = "created"
	FileEventModified FileEventKind = "modified"
)

type fileSignature struct {
	modTime time.Time
	size    int64
}

// NewFsWalker is Markdown-provider filesystem change detection only.
// It must never be wired into the execution processor chain.
func NewFsWalker(root string, interval time.Duration, chout chan<- FileEvent) chain.Processor {
	walker := &FsWalker{
		root:     root,
		interval: interval,
		seen:     map[string]fileSignature{},
	}
	return chain.NewEntryPoint(chout, walker)
}

// FsWalker polls Work/ and _registry/ for Markdown/task/workspace changes.
type FsWalker struct {
	root     string
	interval time.Duration
	seen     map[string]fileSignature
	started  bool
}

func (walker *FsWalker) Start(chout chan<- FileEvent, ctx context.Context) {
	if walker.interval <= 0 {
		walker.interval = 2 * time.Second
	}
	walker.scan(chout, ctx)
	ticker := time.NewTicker(walker.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			walker.scan(chout, ctx)
		}
	}
}

func (walker *FsWalker) Decorate(event FileEvent) (FileEvent, error) { return event, nil }
func (walker *FsWalker) Stop()                                       {}

func (walker *FsWalker) scan(chout chan<- FileEvent, ctx context.Context) {
	walker.scanDir(filepath.Join(walker.root, "Work"), chout, ctx)
	walker.scanDir(filepath.Join(walker.root, "_registry"), chout, ctx)
	walker.started = true
}

func (walker *FsWalker) scanDir(root string, chout chan<- FileEvent, ctx context.Context) {
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !walker.isWatchedFile(path) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		signature := fileSignature{modTime: info.ModTime(), size: info.Size()}
		previous, seen := walker.seen[path]
		if seen && previous == signature {
			return nil
		}
		walker.seen[path] = signature
		kind := FileEventModified
		if !seen {
			kind = FileEventCreated
		}
		if !walker.started {
			kind = FileEventInitial
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case chout <- FileEvent{Path: path, ModTime: signature.modTime, Size: signature.size, Kind: kind}:
			return nil
		}
	})
}

func isWorkspaceFile(path string) bool {
	return filepath.Base(path) == "PROJECT.md" &&
		filepath.Base(filepath.Dir(filepath.Dir(path))) == "Work"
}

func (walker *FsWalker) isWatchedFile(path string) bool {
	if IsTaskFile(path) || isWorkspaceFile(path) {
		return true
	}
	rel, err := filepath.Rel(walker.root, path)
	if err != nil {
		return false
	}
	return filepath.Dir(rel) == "_registry" && filepath.Ext(path) == ".json"
}
