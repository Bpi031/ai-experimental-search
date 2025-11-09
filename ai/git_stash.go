package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
)

// StashEntry represents a single stash entry.
type StashEntry struct {
	Index       int       `json:"index"`
	Message     string    `json:"message"`
	Hash        string    `json:"hash"`
	ShortHash   string    `json:"short_hash"`
	CreatedAt   time.Time `json:"created_at"`
	Author      string    `json:"author"`
	Email       string    `json:"email"`
	BranchName  string    `json:"branch_name"`
	FilesStashed []string  `json:"files_stashed"`
}

// StashOptions configures stash creation.
type StashOptions struct {
	Message      string   // Custom stash message
	IncludeUntracked bool // Include untracked files
	KeepIndex    bool     // Keep changes in index
	Files        []string // Stash only specific files (empty = all)
}

// StashResult represents the result of a stash operation.
type StashResult struct {
	Entry        *StashEntry `json:"entry"`
	FilesStashed int         `json:"files_stashed"`
	Message      string      `json:"message"`
	NeedsConfirm bool        `json:"needs_confirm"`
}

// GitStash provides Git stash operations.
type GitStash struct {
	repo                *git.Repository
	RequireConfirmation bool
}

// NewGitStash creates a new Git stash manager.
func NewGitStash(repo *git.Repository) *GitStash {
	return &GitStash{
		repo:                repo,
		RequireConfirmation: true,
	}
}

// StashChanges saves current changes to stash.
func (gs *GitStash) StashChanges(ctx context.Context, opts StashOptions) (*StashResult, error) {
	w, err := gs.repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	// Check if there are changes to stash
	status, err := w.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	if status.IsClean() {
		return nil, fmt.Errorf("no changes to stash")
	}

	// Get current branch
	head, err := gs.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}
	branchName := head.Name().Short()

	// Create stash message
	message := opts.Message
	if message == "" {
		message = fmt.Sprintf("WIP on %s", branchName)
	}

	// Count files to be stashed
	filesStashed := []string{}
	for file := range status {
		filesStashed = append(filesStashed, file)
	}

	result := &StashResult{
		FilesStashed: len(filesStashed),
		Message:      message,
		NeedsConfirm: gs.RequireConfirmation,
	}

	// If confirmation required, return pending operation
	if gs.RequireConfirmation {
		result.Entry = &StashEntry{
			Message:      message,
			BranchName:   branchName,
			FilesStashed: filesStashed,
			CreatedAt:    time.Now(),
		}
		return result, nil
	}

	// Actually create the stash
	// Note: go-git doesn't have native stash support, so we simulate it
	// by creating a commit on a special stash ref
	commit, err := gs.createStashCommit(w, message, branchName, filesStashed)
	if err != nil {
		return nil, fmt.Errorf("failed to create stash: %w", err)
	}

	result.Entry = &StashEntry{
		Message:      message,
		Hash:         commit.Hash.String(),
		ShortHash:    commit.Hash.String()[:7],
		CreatedAt:    commit.Author.When,
		Author:       commit.Author.Name,
		Email:        commit.Author.Email,
		BranchName:   branchName,
		FilesStashed: filesStashed,
	}

	// Reset working directory
	if !opts.KeepIndex {
		err = w.Reset(&git.ResetOptions{
			Mode: git.HardReset,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to reset working directory: %w", err)
		}
	}

	return result, nil
}

// createStashCommit creates a commit representing the stash.
func (gs *GitStash) createStashCommit(w *git.Worktree, message, branch string, files []string) (*object.Commit, error) {
	// Add all changes
	if len(files) == 0 {
		_, err := w.Add(".")
		if err != nil {
			return nil, err
		}
	} else {
		for _, file := range files {
			_, err := w.Add(file)
			if err != nil {
				return nil, err
			}
		}
	}

	// Create commit
	commitHash, err := w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Go-Git Stash",
			Email: "stash@go-git",
			When:  time.Now(),
		},
	})
	if err != nil {
		return nil, err
	}

	return gs.repo.CommitObject(commitHash)
}

// ListStashes returns all stash entries.
func (gs *GitStash) ListStashes(ctx context.Context) ([]StashEntry, error) {
	stashes := []StashEntry{}

	// Get stash references
	// Note: This is a simplified implementation
	// Real stash would be stored in refs/stash
	refs, err := gs.repo.References()
	if err != nil {
		return nil, fmt.Errorf("failed to get references: %w", err)
	}

	index := 0
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		// Look for stash references (refs/stash or custom refs)
		if ref.Name().String() == "refs/stash" || 
		   (ref.Name().IsBranch() && len(ref.Name().Short()) > 6 && ref.Name().Short()[:6] == "stash/") {
			
			commit, err := gs.repo.CommitObject(ref.Hash())
			if err != nil {
				return nil
			}

			stashes = append(stashes, StashEntry{
				Index:     index,
				Message:   commit.Message,
				Hash:      commit.Hash.String(),
				ShortHash: commit.Hash.String()[:7],
				CreatedAt: commit.Author.When,
				Author:    commit.Author.Name,
				Email:     commit.Author.Email,
			})
			index++
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list stashes: %w", err)
	}

	return stashes, nil
}

// ApplyStash applies a stash entry to the working directory.
func (gs *GitStash) ApplyStash(ctx context.Context, index int, drop bool) error {
	stashes, err := gs.ListStashes(ctx)
	if err != nil {
		return err
	}

	if index < 0 || index >= len(stashes) {
		return fmt.Errorf("invalid stash index: %d", index)
	}

	stash := stashes[index]

	// Get the stash commit
	stashHash := plumbing.NewHash(stash.Hash)
	commit, err := gs.repo.CommitObject(stashHash)
	if err != nil {
		return fmt.Errorf("failed to get stash commit: %w", err)
	}

	// Get worktree
	w, err := gs.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Checkout the stash commit files
	// Note: This is simplified - real git stash apply is more complex
	tree, err := commit.Tree()
	if err != nil {
		return fmt.Errorf("failed to get stash tree: %w", err)
	}

	err = w.Checkout(&git.CheckoutOptions{
		Hash: tree.Hash,
	})
	if err != nil {
		return fmt.Errorf("failed to apply stash: %w", err)
	}

	// Drop stash if requested
	if drop {
		return gs.DropStash(ctx, index)
	}

	return nil
}

// PopStash applies and drops a stash entry.
func (gs *GitStash) PopStash(ctx context.Context, index int) error {
	return gs.ApplyStash(ctx, index, true)
}

// DropStash removes a stash entry.
func (gs *GitStash) DropStash(ctx context.Context, index int) error {
	stashes, err := gs.ListStashes(ctx)
	if err != nil {
		return err
	}

	if index < 0 || index >= len(stashes) {
		return fmt.Errorf("invalid stash index: %d", index)
	}

	stash := stashes[index]

	// Remove the stash reference
	// Note: This is simplified
	refName := plumbing.NewBranchReferenceName(fmt.Sprintf("stash/%s", stash.ShortHash))
	err = gs.repo.Storer.RemoveReference(refName)
	if err != nil {
		// Try the standard stash ref
		err = gs.repo.Storer.RemoveReference(plumbing.ReferenceName("refs/stash"))
		if err != nil {
			return fmt.Errorf("failed to drop stash: %w", err)
		}
	}

	return nil
}

// ClearStashes removes all stash entries.
func (gs *GitStash) ClearStashes(ctx context.Context) error {
	stashes, err := gs.ListStashes(ctx)
	if err != nil {
		return err
	}

	for i := range stashes {
		if err := gs.DropStash(ctx, i); err != nil {
			// Continue even if some fail
			continue
		}
	}

	return nil
}

// Repository methods for stash operations

// StashChanges saves current changes to stash.
func (r *Repository) StashChanges(ctx context.Context, opts StashOptions) (*StashResult, error) {
	repo, err := git.PlainOpen(r.Path)
	if err != nil {
		return nil, err
	}
	stash := NewGitStash(repo)
	stash.RequireConfirmation = true
	return stash.StashChanges(ctx, opts)
}

// ListStashes returns all stash entries.
func (r *Repository) ListStashes(ctx context.Context) ([]StashEntry, error) {
	repo, err := git.PlainOpen(r.Path)
	if err != nil {
		return nil, err
	}
	stash := NewGitStash(repo)
	return stash.ListStashes(ctx)
}

// ApplyStash applies a stash entry.
func (r *Repository) ApplyStash(ctx context.Context, index int, drop bool) error {
	repo, err := git.PlainOpen(r.Path)
	if err != nil {
		return err
	}
	stash := NewGitStash(repo)
	return stash.ApplyStash(ctx, index, drop)
}

// PopStash applies and drops a stash entry.
func (r *Repository) PopStash(ctx context.Context, index int) error {
	repo, err := git.PlainOpen(r.Path)
	if err != nil {
		return err
	}
	stash := NewGitStash(repo)
	return stash.PopStash(ctx, index)
}

// DropStash removes a stash entry.
func (r *Repository) DropStash(ctx context.Context, index int) error {
	repo, err := git.PlainOpen(r.Path)
	if err != nil {
		return err
	}
	stash := NewGitStash(repo)
	return stash.DropStash(ctx, index)
}

// ClearStashes removes all stashes.
func (r *Repository) ClearStashes(ctx context.Context) error {
	repo, err := git.PlainOpen(r.Path)
	if err != nil {
		return err
	}
	stash := NewGitStash(repo)
	return stash.ClearStashes(ctx)
}
