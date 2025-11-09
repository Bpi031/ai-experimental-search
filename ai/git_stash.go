package ai

import (
	"context"
	"fmt"
	"io"
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

	// Get current HEAD commit
	headCommit, err := gs.repo.CommitObject(head.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD commit: %w", err)
	}

	// Stage all changes
	err = w.AddGlob(".")
	if err != nil {
		return nil, fmt.Errorf("failed to stage changes: %w", err)
	}

	// Create a temporary commit to capture the stashed state
	stashHash, err := w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Go-Git Stash",
			Email: "stash@go-git",
			When:  time.Now(),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create stash commit: %w", err)
	}

	// Get the stash commit
	stashCommit, err := gs.repo.CommitObject(stashHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get stash commit: %w", err)
	}

	// Update refs/stash reference to point to the stash commit
	stashRef := plumbing.NewHashReference(plumbing.ReferenceName("refs/stash"), stashHash)
	err = gs.repo.Storer.SetReference(stashRef)
	if err != nil {
		return nil, fmt.Errorf("failed to update stash reference: %w", err)
	}

	result.Entry = &StashEntry{
		Index:        0,
		Message:      message,
		Hash:         stashHash.String(),
		ShortHash:    stashHash.String()[:7],
		CreatedAt:    stashCommit.Author.When,
		Author:       stashCommit.Author.Name,
		Email:        stashCommit.Author.Email,
		BranchName:   branchName,
		FilesStashed: filesStashed,
	}

	// Reset HEAD back to original commit (but keep the stash ref)
	err = gs.repo.Storer.SetReference(plumbing.NewHashReference(head.Name(), headCommit.Hash))
	if err != nil {
		return nil, fmt.Errorf("failed to reset HEAD: %w", err)
	}

	// Reset working directory to clean state
	if !opts.KeepIndex {
		err = w.Reset(&git.ResetOptions{
			Mode:   git.HardReset,
			Commit: headCommit.Hash,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to reset working directory: %w", err)
		}
	}

	return result, nil
}

// ListStashes returns all stash entries.
func (gs *GitStash) ListStashes(ctx context.Context) ([]StashEntry, error) {
	stashes := []StashEntry{}

	// Try to get the stash reference
	stashRef, err := gs.repo.Reference(plumbing.ReferenceName("refs/stash"), true)
	if err != nil {
		// No stash exists yet
		if err == plumbing.ErrReferenceNotFound {
			return stashes, nil
		}
		return nil, fmt.Errorf("failed to get stash reference: %w", err)
	}

	// Get the stash commit
	commit, err := gs.repo.CommitObject(stashRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get stash commit: %w", err)
	}

	// Parse branch name from commit message
	branchName := ""
	if len(commit.Message) > 7 && commit.Message[:7] == "WIP on " {
		// Extract branch name from "WIP on branch"
		branchName = commit.Message[7:]
		if idx := len(branchName); idx > 0 {
			// Remove any trailing text after branch name
			for i, ch := range branchName {
				if ch == ':' || ch == '\n' {
					branchName = branchName[:i]
					break
				}
			}
		}
	}

	stashes = append(stashes, StashEntry{
		Index:      0,
		Message:    commit.Message,
		Hash:       commit.Hash.String(),
		ShortHash:  commit.Hash.String()[:7],
		CreatedAt:  commit.Author.When,
		Author:     commit.Author.Name,
		Email:      commit.Author.Email,
		BranchName: branchName,
	})

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

	// Get the stash tree
	tree, err := commit.Tree()
	if err != nil {
		return fmt.Errorf("failed to get stash tree: %w", err)
	}

	// Apply stash by restoring files from the stash tree
	err = tree.Files().ForEach(func(f *object.File) error {
		// Read file content from stash
		reader, err := f.Reader()
		if err != nil {
			return err
		}
		defer reader.Close()

		// Write to worktree
		file, err := w.Filesystem.Create(f.Name)
		if err != nil {
			return err
		}
		defer file.Close()

		// Copy content using io.Copy
		_, err = io.Copy(file, reader)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to apply stash files: %w", err)
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

	// For now, we only support a single stash (index 0)
	// Full stash reflog support would require more complex implementation
	if index != 0 {
		return fmt.Errorf("only dropping the most recent stash (index 0) is supported")
	}

	// Remove the stash reference
	err = gs.repo.Storer.RemoveReference(plumbing.ReferenceName("refs/stash"))
	if err != nil {
		return fmt.Errorf("failed to drop stash: %w", err)
	}

	return nil
}

// ClearStashes removes all stash entries.
func (gs *GitStash) ClearStashes(ctx context.Context) error {
	// Simply remove the stash reference (simplified implementation)
	err := gs.repo.Storer.RemoveReference(plumbing.ReferenceName("refs/stash"))
	if err != nil {
		// If no stash exists, that's fine
		if err == plumbing.ErrReferenceNotFound {
			return nil
		}
		return fmt.Errorf("failed to clear stashes: %w", err)
	}

	return nil
}

// clearStashesOld is the old implementation (kept for reference)
func (gs *GitStash) clearStashesOld(ctx context.Context) error {
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
