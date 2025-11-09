package ai

import (
	"context"
	"testing"
	"time"
)

func TestRecentFilesTracker_GetRecentFiles(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	// Create initial files
	mustWrite(t, repo, "file1.go", "package main")
	mustCommitAll(t, repo, "Add file1")

	time.Sleep(100 * time.Millisecond)

	mustWrite(t, repo, "file2.go", "package main")
	mustCommitAll(t, repo, "Add file2")

	time.Sleep(100 * time.Millisecond)

	mustWrite(t, repo, "file3.go", "package main")
	mustCommitAll(t, repo, "Add file3")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	tracker := NewRecentFilesTracker(repo, repoPath)

	ctx := context.Background()
	recent, err := tracker.GetRecentFiles(ctx, RecentFilesOptions{
		MaxFiles: 10,
		MaxAge:   24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("GetRecentFiles() error = %v", err)
	}

	if len(recent) < 3 {
		t.Errorf("GetRecentFiles() got %d files, want at least 3", len(recent))
	}

	// Check that files are sorted by most recent first
	for i := 0; i < len(recent)-1; i++ {
		if recent[i].LastModified.Before(recent[i+1].LastModified) {
			t.Errorf("GetRecentFiles() not sorted: %s (%v) before %s (%v)",
				recent[i].FilePath, recent[i].LastModified,
				recent[i+1].FilePath, recent[i+1].LastModified)
		}
	}

	// Most recent should be file3
	if recent[0].FilePath != "file3.go" {
		t.Errorf("GetRecentFiles() most recent = %s, want file3.go", recent[0].FilePath)
	}
}

func TestRecentFilesTracker_FilterByExtension(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	mustWrite(t, repo, "main.go", "package main")
	mustWrite(t, repo, "utils.go", "package utils")
	mustWrite(t, repo, "README.md", "# Doc")
	mustWrite(t, repo, "config.json", "{}")
	mustCommitAll(t, repo, "Add files")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	tracker := NewRecentFilesTracker(repo, repoPath)

	ctx := context.Background()
	recent, err := tracker.GetRecentFiles(ctx, RecentFilesOptions{
		MaxFiles:  10,
		Extension: []string{".go"},
	})
	if err != nil {
		t.Fatalf("GetRecentFiles() error = %v", err)
	}

	// Should only have .go files
	for _, file := range recent {
		if len(file.FilePath) < 3 || file.FilePath[len(file.FilePath)-3:] != ".go" {
			t.Errorf("GetRecentFiles() with .go filter returned %s", file.FilePath)
		}
	}

	if len(recent) < 2 {
		t.Errorf("GetRecentFiles() got %d .go files, want at least 2", len(recent))
	}
}

func TestRecentFilesTracker_GetRecentlyModifiedFiles(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	mustWrite(t, repo, "file1.go", "package main")
	mustCommitAll(t, repo, "Add file1")

	mustWrite(t, repo, "file2.go", "package main")
	mustCommitAll(t, repo, "Add file2")

	mustWrite(t, repo, "file3.go", "package main")
	mustCommitAll(t, repo, "Add file3")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	tracker := NewRecentFilesTracker(repo, repoPath)

	ctx := context.Background()
	files, err := tracker.GetRecentlyModifiedFiles(ctx, 2)
	if err != nil {
		t.Fatalf("GetRecentlyModifiedFiles() error = %v", err)
	}

	if len(files) == 0 {
		t.Error("GetRecentlyModifiedFiles() returned no files")
	}

	// Should contain files from last 2 commits
	if len(files) < 2 {
		t.Errorf("GetRecentlyModifiedFiles() got %d files, want at least 2", len(files))
	}
}

func TestRecentFilesTracker_GetFileHistory(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	// Create file and modify it multiple times
	mustWrite(t, repo, "main.go", "package main\n\nfunc main() {}")
	mustCommitAll(t, repo, "Initial version")

	time.Sleep(100 * time.Millisecond)

	mustWrite(t, repo, "main.go", "package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}")
	mustCommitAll(t, repo, "Add println")

	time.Sleep(100 * time.Millisecond)

	mustWrite(t, repo, "main.go", "package main\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}")
	mustCommitAll(t, repo, "Update message")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	tracker := NewRecentFilesTracker(repo, repoPath)

	ctx := context.Background()
	history, err := tracker.GetFileHistory(ctx, "main.go", 10)
	if err != nil {
		t.Fatalf("GetFileHistory() error = %v", err)
	}

	if len(history) < 3 {
		t.Errorf("GetFileHistory() got %d commits, want at least 3", len(history))
	}

	// All entries should be for main.go
	for _, entry := range history {
		if entry.FilePath != "main.go" {
			t.Errorf("GetFileHistory() entry for %s, want main.go", entry.FilePath)
		}
	}

	// Should have commit messages
	if history[0].CommitMsg == "" {
		t.Error("GetFileHistory() entry has empty commit message")
	}
}

func TestRecentFilesTracker_MaxFilesLimit(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	// Create many files
	for i := 0; i < 20; i++ {
		mustWrite(t, repo, "file"+string(rune('A'+i))+".go", "package main")
	}
	mustCommitAll(t, repo, "Add many files")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	tracker := NewRecentFilesTracker(repo, repoPath)

	ctx := context.Background()
	recent, err := tracker.GetRecentFiles(ctx, RecentFilesOptions{
		MaxFiles: 5,
	})
	if err != nil {
		t.Fatalf("GetRecentFiles() error = %v", err)
	}

	if len(recent) > 5 {
		t.Errorf("GetRecentFiles() got %d files, want max 5", len(recent))
	}
}

func TestRecentFilesTracker_ChangeTypes(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	// Added file
	mustWrite(t, repo, "new.go", "package main")
	mustCommitAll(t, repo, "Add new file")

	// Modified file
	mustWrite(t, repo, "new.go", "package main\n\n// Modified")
	mustCommitAll(t, repo, "Modify file")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	tracker := NewRecentFilesTracker(repo, repoPath)

	ctx := context.Background()
	recent, err := tracker.GetRecentFiles(ctx, RecentFilesOptions{
		MaxFiles: 10,
	})
	if err != nil {
		t.Fatalf("GetRecentFiles() error = %v", err)
	}

	if len(recent) == 0 {
		t.Fatal("GetRecentFiles() returned no files")
	}

	// Check that change type is set
	if recent[0].ChangeType == "" {
		t.Error("GetRecentFiles() ChangeType is empty")
	}
}
