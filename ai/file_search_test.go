package ai

import (
	"context"
	"testing"
)

func TestFileSearcher_FindFiles(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	// Create test files
	mustWrite(t, repo, "main.go", "package main")
	mustWrite(t, repo, "test_helper.go", "package test")
	mustWrite(t, repo, "internal/utils.go", "package utils")
	mustWrite(t, repo, "cmd/server/main.go", "package main")
	mustWrite(t, repo, "README.md", "# Test")
	mustCommitAll(t, repo, "Add test files")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	searcher := NewFileSearcher(repo, repoPath)

	ctx := context.Background()

	tests := []struct {
		name          string
		opts          FindFilesOptions
		wantMinCount  int
		wantContains  string
	}{
		{
			name: "find all go files",
			opts: FindFilesOptions{
				Query:      "go",
				MaxResults: 10,
			},
			wantMinCount: 3,
			wantContains: "main.go",
		},
		{
			name: "find by exact name",
			opts: FindFilesOptions{
				Query:      "main.go",
				MaxResults: 10,
			},
			wantMinCount: 1,
			wantContains: "main.go",
		},
		{
			name: "find with glob pattern",
			opts: FindFilesOptions{
				Query:      "*.go",
				UseGlob:    true,
				MaxResults: 10,
			},
			wantMinCount: 3,
			wantContains: "main.go",
		},
		{
			name: "find by prefix",
			opts: FindFilesOptions{
				Query:      "test",
				MaxResults: 10,
			},
			wantMinCount: 1,
			wantContains: "test_helper.go",
		},
		{
			name: "filter by extension",
			opts: FindFilesOptions{
				Query:      "",
				Extensions: []string{".md"},
				MaxResults: 10,
			},
			wantMinCount: 1,
			wantContains: "README.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, err := searcher.FindFiles(ctx, tt.opts)
			if err != nil {
				t.Fatalf("FindFiles() error = %v", err)
			}

			if len(matches) < tt.wantMinCount {
				t.Errorf("FindFiles() got %d matches, want at least %d", len(matches), tt.wantMinCount)
			}

			// Check if expected file is in results
			found := false
			for _, match := range matches {
				if match.FileName == tt.wantContains || match.FilePath == tt.wantContains {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("FindFiles() results don't contain %s", tt.wantContains)
			}
		})
	}
}

func TestFileSearcher_ListAllFiles(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	mustWrite(t, repo, "file1.go", "package main")
	mustWrite(t, repo, "file2.go", "package main")
	mustWrite(t, repo, "file3.txt", "text")
	mustCommitAll(t, repo, "Add files")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	searcher := NewFileSearcher(repo, repoPath)

	ctx := context.Background()
	files, err := searcher.ListAllFiles(ctx)
	if err != nil {
		t.Fatalf("ListAllFiles() error = %v", err)
	}

	if len(files) < 3 {
		t.Errorf("ListAllFiles() got %d files, want at least 3", len(files))
	}
}

func TestFileSearcher_GetFilesByExtension(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	mustWrite(t, repo, "file1.go", "package main")
	mustWrite(t, repo, "file2.go", "package main")
	mustWrite(t, repo, "file3.txt", "text")
	mustWrite(t, repo, "file4.md", "# Doc")
	mustCommitAll(t, repo, "Add files")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	searcher := NewFileSearcher(repo, repoPath)

	ctx := context.Background()

	tests := []struct {
		name       string
		extensions []string
		wantCount  int
	}{
		{
			name:       "find go files",
			extensions: []string{".go"},
			wantCount:  2,
		},
		{
			name:       "find markdown files",
			extensions: []string{".md"},
			wantCount:  1,
		},
		{
			name:       "find multiple extensions",
			extensions: []string{".go", ".md"},
			wantCount:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := searcher.GetFilesByExtension(ctx, tt.extensions)
			if err != nil {
				t.Fatalf("GetFilesByExtension() error = %v", err)
			}

			if len(files) != tt.wantCount {
				t.Errorf("GetFilesByExtension() got %d files, want %d", len(files), tt.wantCount)
			}
		})
	}
}

func TestFileSearcher_CaseInsensitive(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	mustWrite(t, repo, "TestFile.go", "package main")
	mustWrite(t, repo, "UPPERCASE.txt", "text")
	mustCommitAll(t, repo, "Add files")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	searcher := NewFileSearcher(repo, repoPath)

	ctx := context.Background()

	// Case insensitive search (default)
	matches, err := searcher.FindFiles(ctx, FindFilesOptions{
		Query:         "testfile",
		CaseSensitive: false,
		MaxResults:    10,
	})
	if err != nil {
		t.Fatalf("FindFiles() error = %v", err)
	}

	if len(matches) == 0 {
		t.Error("FindFiles() case insensitive search found no matches")
	}
}

func TestFileSearcher_MatchScoring(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	mustWrite(t, repo, "test.go", "package main")
	mustWrite(t, repo, "test_utils.go", "package utils")
	mustWrite(t, repo, "internal/test.go", "package internal")
	mustCommitAll(t, repo, "Add files")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	searcher := NewFileSearcher(repo, repoPath)

	ctx := context.Background()

	matches, err := searcher.FindFiles(ctx, FindFilesOptions{
		Query:      "test.go",
		MaxResults: 10,
	})
	if err != nil {
		t.Fatalf("FindFiles() error = %v", err)
	}

	if len(matches) < 2 {
		t.Fatalf("FindFiles() got %d matches, want at least 2", len(matches))
	}

	// Exact match should have highest score
	if matches[0].FileName != "test.go" {
		t.Errorf("FindFiles() first match = %s, want test.go (exact match should rank highest)", matches[0].FileName)
	}

	if matches[0].Score < matches[1].Score {
		t.Errorf("FindFiles() exact match score %f should be higher than %f", matches[0].Score, matches[1].Score)
	}
}
