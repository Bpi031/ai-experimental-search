package ai

import (
	"context"
	"testing"
)

func TestFileMetadataProvider_GetFileMetadata(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	content := "package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}\n"
	mustWrite(t, repo, "main.go", content)
	mustCommitAll(t, repo, "Add main.go")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()

	ctx := context.Background()
	provider := NewFileMetadataProvider(repo, repoPath)
	metadata, err := provider.GetFileMetadata(ctx, "main.go")
	if err != nil {
		t.Fatalf("GetFileMetadata() error = %v", err)
	}

	// Check basic info
	if metadata.FilePath != "main.go" {
		t.Errorf("GetFileMetadata() FilePath = %s, want main.go", metadata.FilePath)
	}

	if metadata.FileName != "main.go" {
		t.Errorf("GetFileMetadata() FileName = %s, want main.go", metadata.FileName)
	}

	// Check extension
	if metadata.Extension != ".go" {
		t.Errorf("GetFileMetadata() Extension = %s, want .go", metadata.Extension)
	}

	// Check language detection
	if metadata.Language != "Go" {
		t.Errorf("GetFileMetadata() Language = %s, want Go", metadata.Language)
	}

	// Check size
	if metadata.Size == 0 {
		t.Error("GetFileMetadata() Size = 0, want > 0")
	}

	if metadata.SizeHuman == "" {
		t.Error("GetFileMetadata() SizeHuman is empty")
	}

	// Check line count
	if metadata.LineCount != 5 {
		t.Errorf("GetFileMetadata() LineCount = %d, want 5", metadata.LineCount)
	}

	// Check it's not binary
	if metadata.IsBinary {
		t.Error("GetFileMetadata() IsBinary = true, want false for .go file")
	}

	// Check Git status
	if metadata.GitStatus == "" {
		t.Error("GetFileMetadata() GitStatus is empty")
	}

	// Check blob hash
	if metadata.BlobHash == "" {
		t.Error("GetFileMetadata() BlobHash is empty")
	}
}

func TestFileMetadataProvider_GetBulkMetadata(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	mustWrite(t, repo, "file1.go", "package main")
	mustWrite(t, repo, "file2.go", "package utils")
	mustWrite(t, repo, "file3.txt", "text content")
	mustCommitAll(t, repo, "Add files")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()

	ctx := context.Background()
	provider := NewFileMetadataProvider(repo, repoPath)
	files := []string{"file1.go", "file2.go", "file3.txt"}
	metadata, err := provider.GetBulkMetadata(ctx, files)
	if err != nil {
		t.Fatalf("GetBulkMetadata() error = %v", err)
	}

	if len(metadata) != 3 {
		t.Errorf("GetBulkMetadata() got %d items, want 3", len(metadata))
	}

	// Check that all files are present
	fileMap := make(map[string]bool)
	for _, m := range metadata {
		fileMap[m.FilePath] = true
	}

	for _, file := range files {
		if !fileMap[file] {
			t.Errorf("GetBulkMetadata() missing file %s", file)
		}
	}
}

func TestFileMetadataProvider_GitStatus(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	// Create and commit file
	mustWrite(t, repo, "tracked.go", "package main")
	mustCommitAll(t, repo, "Add tracked file")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	provider := NewFileMetadataProvider(repo, repoPath)

	ctx := context.Background()
	metadata, err := provider.GetFileMetadata(ctx, "tracked.go")
	if err != nil {
		t.Fatalf("GetFileMetadata() error = %v", err)
	}

	// Should be tracked
	if metadata.GitStatus != "tracked" {
		t.Errorf("GetFileMetadata() GitStatus = %s, want tracked", metadata.GitStatus)
	}

	// Modify file
	mustWrite(t, repo, "tracked.go", "package main\n\n// Modified")

	metadata, err = provider.GetFileMetadata(ctx, "tracked.go")
	if err != nil {
		t.Fatalf("GetFileMetadata() error = %v", err)
	}

	// Should be modified
	if metadata.GitStatus != "modified" {
		t.Errorf("GetFileMetadata() GitStatus = %s, want modified", metadata.GitStatus)
	}
}

func TestFileMetadataProvider_LastCommit(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	mustWrite(t, repo, "main.go", "package main")
	mustCommitAll(t, repo, "Initial commit")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	provider := NewFileMetadataProvider(repo, repoPath)

	ctx := context.Background()
	metadata, err := provider.GetFileMetadata(ctx, "main.go")
	if err != nil {
		t.Fatalf("GetFileMetadata() error = %v", err)
	}

	// Should have last commit info
	if metadata.LastCommit == nil {
		t.Error("GetFileMetadata() LastCommit is nil")
	} else {
		if metadata.LastCommit.Hash == "" {
			t.Error("GetFileMetadata() LastCommit.Hash is empty")
		}
		if metadata.LastCommit.Message == "" {
			t.Error("GetFileMetadata() LastCommit.Message is empty")
		}
	}
}

func TestFileMetadataProvider_BinaryDetection(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	// Text file
	mustWrite(t, repo, "text.txt", "Hello, World!")
	mustCommitAll(t, repo, "Add text file")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	provider := NewFileMetadataProvider(repo, repoPath)

	ctx := context.Background()
	metadata, err := provider.GetFileMetadata(ctx, "text.txt")
	if err != nil {
		t.Fatalf("GetFileMetadata() error = %v", err)
	}

	if metadata.IsBinary {
		t.Error("GetFileMetadata() IsBinary = true for text file")
	}

	if metadata.Encoding == "" {
		t.Error("GetFileMetadata() Encoding is empty")
	}
}

func TestFileMetadataProvider_SizeFormatting(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{100, "100 B"},
		{1024, "1.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			formatted := formatBytes(tt.bytes)
			if formatted != tt.want {
				t.Errorf("formatBytes(%d) = %s, want %s", tt.bytes, formatted, tt.want)
			}
		})
	}
}

func TestFileMetadataProvider_NonExistentFile(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	mustWrite(t, repo, "exists.go", "package main")
	mustCommitAll(t, repo, "Add file")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	provider := NewFileMetadataProvider(repo, repoPath)

	ctx := context.Background()
	_, err := provider.GetFileMetadata(ctx, "nonexistent.go")
	if err == nil {
		t.Error("GetFileMetadata() expected error for non-existent file, got nil")
	}
}
