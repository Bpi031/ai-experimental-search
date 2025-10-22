package ai

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
)

// IndexState tracks the indexing status of a repository.
// It's persisted to .git/ai-index.json to detect staleness across runs.
type IndexState struct {
	LastCommit  string    `json:"last_commit"`  // Hash of HEAD when last indexed
	LastIndexed time.Time `json:"last_indexed"` // Timestamp of last indexing
	FileCount   int       `json:"file_count"`   // Number of indexed files
	ChunkCount  int       `json:"chunk_count"`  // Number of indexed chunks
	ModelName   string    `json:"model_name"`   // Embedding model used
}

// indexStatePath returns the path to the index state file.
func (r *Repository) indexStatePath() string {
	return filepath.Join(r.Path, ".git", "ai-index.json")
}

// GetIndexState reads the persisted index state.
// Returns nil if never indexed.
func (r *Repository) GetIndexState() (*IndexState, error) {
	data, err := os.ReadFile(r.indexStatePath())
	if os.IsNotExist(err) {
		return nil, nil // Not yet indexed
	}
	if err != nil {
		return nil, err
	}
	
	var state IndexState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// SaveIndexState persists the current index state.
func (r *Repository) SaveIndexState(state *IndexState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	
	// Ensure .git directory exists
	gitDir := filepath.Join(r.Path, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		return err
	}
	
	return os.WriteFile(r.indexStatePath(), data, 0644)
}

// IsIndexed returns true if the repository has been indexed at least once.
func (r *Repository) IsIndexed() bool {
	state, err := r.GetIndexState()
	return err == nil && state != nil && state.FileCount > 0
}

// NeedsReindex determines if the index is stale and needs updating.
// Returns true if:
// - Never indexed
// - HEAD commit changed since last index
// - More than maxAge has passed since last index
func (r *Repository) NeedsReindex(maxAge time.Duration) (bool, error) {
	state, err := r.GetIndexState()
	if err != nil || state == nil {
		return true, nil // Never indexed
	}
	
	// Check if too old
	if maxAge > 0 && time.Since(state.LastIndexed) > maxAge {
		return true, nil
	}
	
	// Check if HEAD changed
	repo, err := r.getGitRepo()
	if err != nil {
		return false, err
	}
	
	head, err := repo.Head()
	if err != nil {
		return false, err
	}
	
	return head.Hash().String() != state.LastCommit, nil
}

// getGitRepo opens the underlying go-git repository.
func (r *Repository) getGitRepo() (*gogit.Repository, error) {
	return gogit.PlainOpen(r.Path)
}

// GetChangedFiles returns files that changed between the last indexed commit and HEAD.
// Returns empty slice if no previous index or on error.
func (r *Repository) GetChangedFiles() ([]string, error) {
	state, err := r.GetIndexState()
	if err != nil || state == nil {
		return nil, nil // No previous index
	}
	
	repo, err := r.getGitRepo()
	if err != nil {
		return nil, err
	}
	
	head, err := repo.Head()
	if err != nil {
		return nil, err
	}
	
	// If same commit, no changes
	if head.Hash().String() == state.LastCommit {
		return []string{}, nil
	}
	
	// Get commits between last index and HEAD
	lastHash := plumbing.NewHash(state.LastCommit)
	lastCommit, err := repo.CommitObject(lastHash)
	if err != nil {
		// If last commit not found, do full reindex
		return nil, nil
	}
	
	headCommit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return nil, err
	}
	
	// Get diff between commits
	patch, err := lastCommit.Patch(headCommit)
	if err != nil {
		return nil, err
	}
	
	var changed []string
	for _, fileStat := range patch.Stats() {
		changed = append(changed, fileStat.Name)
	}
	
	return changed, nil
}
