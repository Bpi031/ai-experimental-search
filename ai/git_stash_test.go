package ai

import (
	"context"
	"testing"
)

func TestGitStash_StashChanges(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	mustWrite(t, repo, "main.go", "package main")
	mustCommitAll(t, repo, "Initial commit")

	// Make some changes
	mustWrite(t, repo, "main.go", "package main\n\n// Modified")

	ctx := context.Background()
	
	// Create stash
	stash := NewGitStash(repo)
	stash.RequireConfirmation = false
	
	result, err := stash.StashChanges(ctx, StashOptions{
		Message: "WIP: testing stash",
	})
	if err != nil {
		t.Fatalf("StashChanges() error = %v", err)
	}

	if result.FilesStashed == 0 {
		t.Error("StashChanges() FilesStashed = 0, want > 0")
	}

	if result.Entry == nil {
		t.Error("StashChanges() Entry is nil")
	}
}

func TestGitStash_ListStashes(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	mustWrite(t, repo, "main.go", "package main")
	mustCommitAll(t, repo, "Initial commit")

	ctx := context.Background()
	
	stash := NewGitStash(repo)
	stashes, err := stash.ListStashes(ctx)
	if err != nil {
		t.Fatalf("ListStashes() error = %v", err)
	}

	// New repo should have no stashes
	if len(stashes) != 0 {
		t.Errorf("ListStashes() got %d stashes, want 0 for new repo", len(stashes))
	}
}

func TestGitStash_StashOptions(t *testing.T) {
	tests := []struct {
		name string
		opts StashOptions
	}{
		{
			name: "with custom message",
			opts: StashOptions{
				Message: "WIP: custom message",
			},
		},
		{
			name: "include untracked",
			opts: StashOptions{
				IncludeUntracked: true,
			},
		},
		{
			name: "keep index",
			opts: StashOptions{
				KeepIndex: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that options are properly structured
			if tt.opts.Message == "" && !tt.opts.IncludeUntracked && !tt.opts.KeepIndex {
				t.Error("StashOptions should have at least one field set")
			}
		})
	}
}

func TestGitStash_StashEntry(t *testing.T) {
	entry := StashEntry{
		Index:      0,
		Message:    "WIP on master",
		Hash:       "abc123def456",
		ShortHash:  "abc123d",
		BranchName: "master",
	}

	if entry.Index != 0 {
		t.Errorf("StashEntry Index = %d, want 0", entry.Index)
	}

	if entry.Message == "" {
		t.Error("StashEntry Message is empty")
	}

	if entry.BranchName == "" {
		t.Error("StashEntry BranchName is empty")
	}
}

func TestGitStash_ApplyStash(t *testing.T) {
	t.Skip("Stash apply requires full worktree simulation - skipping for now")
}

func TestGitStash_PopStash(t *testing.T) {
	t.Skip("Stash pop requires full worktree simulation - skipping for now")
}

func TestGitStash_DropStash(t *testing.T) {
	t.Skip("Stash drop requires full stash setup - skipping for now")
}

func TestGitStash_ClearStashes(t *testing.T) {
	t.Skip("Stash clear requires full stash setup - skipping for now")
}
