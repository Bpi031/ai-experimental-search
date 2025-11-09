package ai

import (
	"context"
	"strings"
	"testing"
)

func TestGrepSearcher_BasicSearch(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	mustWrite(t, repo, "main.go", "package main\n\nfunc authenticate() {\n\ttoken := \"secret\"\n}")
	mustWrite(t, repo, "utils.go", "package utils\n\nfunc getToken() string {\n\treturn \"token123\"\n}")
	mustWrite(t, repo, "README.md", "# Authentication\n\nUse tokens for auth.")
	mustCommitAll(t, repo, "Add files")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	searcher := NewGrepSearcher(repo, repoPath)

	ctx := context.Background()

	matches, err := searcher.GrepSearch(ctx, GrepOptions{
		Pattern:    "token",
		MaxMatches: 100,
	})
	if err != nil {
		t.Fatalf("GrepSearch() error = %v", err)
	}

	if len(matches) < 3 {
		t.Errorf("GrepSearch() got %d matches, want at least 3", len(matches))
	}

	// Check match details
	for _, match := range matches {
		if match.FilePath == "" {
			t.Error("GrepSearch() match has empty FilePath")
		}
		if match.LineNumber == 0 {
			t.Error("GrepSearch() match has zero LineNumber")
		}
		if match.Line == "" {
			t.Error("GrepSearch() match has empty Line")
		}
	}
}

func TestGrepSearcher_RegexSearch(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	mustWrite(t, repo, "config.go", "const API_KEY = \"abc123\"\nconst SECRET_KEY = \"xyz789\"")
	mustWrite(t, repo, "main.go", "var apiKey = \"key456\"")
	mustCommitAll(t, repo, "Add files")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	searcher := NewGrepSearcher(repo, repoPath)

	ctx := context.Background()

	// Search for KEY pattern (uppercase)
	matches, err := searcher.GrepSearch(ctx, GrepOptions{
		Pattern:    "[A-Z_]+KEY",
		UseRegex:   true,
		MaxMatches: 100,
	})
	if err != nil {
		t.Fatalf("GrepSearch() regex error = %v", err)
	}

	if len(matches) < 2 {
		t.Errorf("GrepSearch() regex got %d matches, want at least 2", len(matches))
	}
}

func TestGrepSearcher_CaseInsensitive(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	mustWrite(t, repo, "test.go", "package test\n\nvar TODO = \"Fix this\"\nvar todo = \"also fix\"\nvar ToDo = \"and this\"")
	mustCommitAll(t, repo, "Add TODO items")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	searcher := NewGrepSearcher(repo, repoPath)

	ctx := context.Background()

	matches, err := searcher.GrepSearch(ctx, GrepOptions{
		Pattern:         "todo",
		CaseInsensitive: true,
		MaxMatches:      100,
	})
	if err != nil {
		t.Fatalf("GrepSearch() case insensitive error = %v", err)
	}

	// Should match all 3 variants
	if len(matches) != 3 {
		t.Errorf("GrepSearch() case insensitive got %d matches, want 3", len(matches))
	}
}

func TestGrepSearcher_ContextLines(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	content := `package main

func setup() {
	// Initialize
	config := loadConfig()
	target := "important"
	validate(config)
}

func teardown() {
	cleanup()
}`
	mustWrite(t, repo, "main.go", content)
	mustCommitAll(t, repo, "Add context test")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	searcher := NewGrepSearcher(repo, repoPath)

	ctx := context.Background()

	matches, err := searcher.GrepSearch(ctx, GrepOptions{
		Pattern:      "target",
		ContextLines: 2,
		MaxMatches:   100,
	})
	if err != nil {
		t.Fatalf("GrepSearch() context error = %v", err)
	}

	// Should have matches with context
	if len(matches) == 0 {
		t.Fatal("GrepSearch() context got no matches")
	}

	// Verify context lines are included
	foundContext := false
	for _, match := range matches {
		if len(match.Before) > 0 || len(match.After) > 0 {
			foundContext = true
			break
		}
	}
	if !foundContext {
		t.Error("GrepSearch() context lines not found in matches")
	}
}

func TestGrepSearcher_FileFiltering(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	mustWrite(t, repo, "main.go", "package main\n\nvar pattern = \"match\"")
	mustWrite(t, repo, "test.py", "pattern = 'match'")
	mustWrite(t, repo, "data.json", "{\"pattern\": \"match\"}")
	mustCommitAll(t, repo, "Add files")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	searcher := NewGrepSearcher(repo, repoPath)

	ctx := context.Background()

	matches, err := searcher.GrepSearch(ctx, GrepOptions{
		Pattern:    "match",
		Include:    []string{"*.go"},
		MaxMatches: 100,
	})
	if err != nil {
		t.Fatalf("GrepSearch() file filter error = %v", err)
	}

	// Should only match .go files
	for _, match := range matches {
		if !strings.HasSuffix(match.FilePath, ".go") {
			t.Errorf("GrepSearch() file filter got match in %s, want only .go files", match.FilePath)
		}
	}
}

func TestGrepSearcher_ExcludePatterns(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	mustWrite(t, repo, "main.go", "package main\n\nvar test = \"value\"")
	mustWrite(t, repo, "main_test.go", "package main\n\nfunc TestValue() { test := \"value\" }")
	mustWrite(t, repo, "helper.go", "package main\n\nvar test = \"value\"")
	mustCommitAll(t, repo, "Add files")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	searcher := NewGrepSearcher(repo, repoPath)

	ctx := context.Background()

	matches, err := searcher.GrepSearch(ctx, GrepOptions{
		Pattern:    "test",
		Exclude:    []string{"*_test.go"},
		MaxMatches: 100,
	})
	if err != nil {
		t.Fatalf("GrepSearch() exclude error = %v", err)
	}

	// Should not include main_test.go
	for _, match := range matches {
		if match.FilePath == "main_test.go" {
			t.Error("GrepSearch() exclude returned match from excluded file")
		}
	}
}

func TestGrepSearcher_GrepCount(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	mustWrite(t, repo, "main.go", "package main\n\nvar x = 1\nvar y = 2\nvar z = 3")
	mustWrite(t, repo, "utils.go", "package utils\n\nvar a = 1\nvar b = 2")
	mustCommitAll(t, repo, "Add files")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	searcher := NewGrepSearcher(repo, repoPath)

	ctx := context.Background()

	count, err := searcher.GrepCount(ctx, GrepOptions{
		Pattern:    "var",
		MaxMatches: 100,
	})
	if err != nil {
		t.Fatalf("GrepCount() error = %v", err)
	}

	if count < 2 {
		t.Errorf("GrepCount() got %d matches, want at least 2", count)
	}
}

func TestGrepSearcher_GrepFiles(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	mustWrite(t, repo, "file1.go", "contains error")
	mustWrite(t, repo, "file2.go", "no issues here")
	mustWrite(t, repo, "file3.go", "error handling")
	mustCommitAll(t, repo, "Add files")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	searcher := NewGrepSearcher(repo, repoPath)

	ctx := context.Background()

	files, err := searcher.GrepFiles(ctx, GrepOptions{
		Pattern: "error",
	})
	if err != nil {
		t.Fatalf("GrepFiles() error = %v", err)
	}

	if len(files) < 2 {
		t.Errorf("GrepFiles() got %d files, want at least 2", len(files))
	}

	// Should not contain file2.go
	for _, file := range files {
		if file == "file2.go" {
			t.Error("GrepFiles() included file2.go which doesn't contain 'error'")
		}
	}
}

func TestGrepSearcher_WholeWord(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	mustWrite(t, repo, "test.go", "package test\n\nvar test = \"test\"\nvar testing = \"testing\"\nvar latest = \"latest\"")
	mustCommitAll(t, repo, "Add word test")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	searcher := NewGrepSearcher(repo, repoPath)

	ctx := context.Background()

	matches, err := searcher.GrepSearch(ctx, GrepOptions{
		Pattern:   "test",
		WholeWord: true,
	})
	if err != nil {
		t.Fatalf("GrepSearch() error = %v", err)
	}

	// Should only match "test" not "testing" or "latest"
	if len(matches) != 2 {
		t.Errorf("GrepSearch() WholeWord got %d matches, want 2", len(matches))
	}
}

func TestGrepSearcher_InvertMatch(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	content := `line one
line with error
line three
line with error again
line five`
	mustWrite(t, repo, "test.txt", content)
	mustCommitAll(t, repo, "Add test file")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	searcher := NewGrepSearcher(repo, repoPath)

	ctx := context.Background()

	matches, err := searcher.GrepSearch(ctx, GrepOptions{
		Pattern: "error",
		Invert:  true,
	})
	if err != nil {
		t.Fatalf("GrepSearch() error = %v", err)
	}

	// Should return lines without "error" (3 lines)
	if len(matches) != 3 {
		t.Errorf("GrepSearch() InvertMatch got %d matches, want 3", len(matches))
	}

	// Verify no match contains "error"
	for _, match := range matches {
		if strings.Contains(match.Line, "error") {
			t.Errorf("InvertMatch returned line containing 'error': %s", match.Line)
		}
	}
}
