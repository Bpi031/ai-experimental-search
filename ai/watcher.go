package ai

import (
	"context"
	"log"
	"time"

	"github.com/go-git/go-git/v6/plumbing"
)

// RepoWatcher monitors a repository for changes and triggers incremental re-indexing.
// This implements the "Background watcher" strategy for keeping indexes fresh.
type RepoWatcher struct {
	repo          *Repository
	interval      time.Duration
	lastIndexHash string
	ticker        *time.Ticker
	stopCh        chan struct{}
	silent        bool
}

// NewRepoWatcher creates a background watcher for the repository.
// It will check for changes every `interval` and incrementally reindex changed files.
func NewRepoWatcher(repo *Repository, interval time.Duration) *RepoWatcher {
	return &RepoWatcher{
		repo:     repo,
		interval: interval,
		stopCh:   make(chan struct{}),
		silent:   false,
	}
}

// Silent suppresses log output from the watcher.
func (w *RepoWatcher) Silent(silent bool) *RepoWatcher {
	w.silent = silent
	return w
}

// Start begins background monitoring. Blocks until Stop() is called or context is cancelled.
func (w *RepoWatcher) Start(ctx context.Context) {
	// Get initial HEAD
	if state, err := w.repo.GetIndexState(); err == nil && state != nil {
		w.lastIndexHash = state.LastCommit
	}
	
	if !w.silent {
		log.Printf("RepoWatcher started (checking every %v)", w.interval)
	}
	
	w.ticker = time.NewTicker(w.interval)
	defer w.ticker.Stop()
	
	for {
		select {
		case <-w.ticker.C:
			if err := w.checkAndReindex(ctx); err != nil && !w.silent {
				log.Printf("RepoWatcher: reindex failed: %v", err)
			}
		case <-w.stopCh:
			if !w.silent {
				log.Println("RepoWatcher stopped")
			}
			return
		case <-ctx.Done():
			if !w.silent {
				log.Println("RepoWatcher cancelled")
			}
			return
		}
	}
}

// Stop halts the background watcher.
func (w *RepoWatcher) Stop() {
	close(w.stopCh)
}

// checkAndReindex checks if HEAD changed and triggers incremental reindex.
func (w *RepoWatcher) checkAndReindex(ctx context.Context) error {
	repo, err := w.repo.getGitRepo()
	if err != nil {
		return err
	}
	
	head, err := repo.Head()
	if err != nil {
		return err
	}
	
	currentHash := head.Hash().String()
	
	// No changes since last check
	if currentHash == w.lastIndexHash {
		return nil
	}
	
	if !w.silent {
		log.Printf("RepoWatcher: detected new commits (%s → %s), re-indexing...", 
			w.lastIndexHash[:7], currentHash[:7])
	}
	
	// Incremental reindex
	if err := w.repo.IndexWithOptions(ctx, IndexOptions{
		Incremental: true,
		Silent:      w.silent,
	}); err != nil {
		return err
	}
	
	w.lastIndexHash = currentHash
	
	if !w.silent {
		log.Println("RepoWatcher: incremental reindex complete")
	}
	
	return nil
}

// GetLastIndexedCommit returns the commit hash that was last indexed.
func (w *RepoWatcher) GetLastIndexedCommit() plumbing.Hash {
	if w.lastIndexHash == "" {
		return plumbing.ZeroHash
	}
	return plumbing.NewHash(w.lastIndexHash)
}
