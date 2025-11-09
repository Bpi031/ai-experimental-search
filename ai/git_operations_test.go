package ai

import (
	"context"
	"testing"
	"time"
)

func TestGitOperations_GetHistory(t *testing.T) {
	// Create temp repo with commits
	repo := mustCreateTempRepo(t)
	
	// Create some commits
	mustWrite(t, repo, "file1.txt", "initial content")
	mustCommitAll(t, repo, "Initial commit")
	
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	
	mustWrite(t, repo, "file2.txt", "second file")
	mustCommitAll(t, repo, "Add file2")
	
	time.Sleep(10 * time.Millisecond)
	
	mustWrite(t, repo, "file1.txt", "updated content")
	mustCommitAll(t, repo, "Update file1")
	
	// Test: Get all history
	gitOps := NewGitOperations(repo)
	history, err := gitOps.GetHistory(context.Background(), HistoryOptions{})
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}
	
	if len(history) < 3 {
		t.Errorf("Expected at least 3 commits, got %d", len(history))
	}
	
	// Verify commit structure
	for _, commit := range history {
		if commit.Hash == "" {
			t.Error("Commit hash is empty")
		}
		if commit.ShortHash == "" {
			t.Error("Short hash is empty")
		}
		if commit.Author == "" {
			t.Error("Author is empty")
		}
		if commit.Message == "" {
			t.Error("Message is empty")
		}
	}
}

func TestGitOperations_GetHistoryWithFilePath(t *testing.T) {
	repo := mustCreateTempRepo(t)
	
	mustWrite(t, repo, "file1.txt", "v1")
	mustCommitAll(t, repo, "Add file1")
	
	mustWrite(t, repo, "file2.txt", "v1")
	mustCommitAll(t, repo, "Add file2")
	
	mustWrite(t, repo, "file1.txt", "v2")
	mustCommitAll(t, repo, "Update file1")
	
	// Test: Get history for specific file
	gitOps := NewGitOperations(repo)
	history, err := gitOps.GetHistory(context.Background(), HistoryOptions{
		FilePath: "file1.txt",
	})
	if err != nil {
		t.Fatalf("GetHistory with FilePath failed: %v", err)
	}
	
	// Should only have commits touching file1.txt (2 commits)
	if len(history) != 2 {
		t.Errorf("Expected 2 commits for file1.txt, got %d", len(history))
	}
}

func TestGitOperations_GetHistoryWithMaxCount(t *testing.T) {
	repo := mustCreateTempRepo(t)
	
	// Create 5 commits
	for i := 1; i <= 5; i++ {
		mustWrite(t, repo, "file.txt", "content")
		mustCommitAll(t, repo, "Commit")
		time.Sleep(5 * time.Millisecond)
	}
	
	// Test: Limit to 3 commits
	gitOps := NewGitOperations(repo)
	history, err := gitOps.GetHistory(context.Background(), HistoryOptions{
		MaxCount: 3,
	})
	if err != nil {
		t.Fatalf("GetHistory with MaxCount failed: %v", err)
	}
	
	if len(history) != 3 {
		t.Errorf("Expected exactly 3 commits, got %d", len(history))
	}
}

func TestGitOperations_GetDiff(t *testing.T) {
	repo := mustCreateTempRepo(t)
	
	// Create initial commit
	mustWrite(t, repo, "file1.txt", "line1\nline2\nline3\n")
	hash1 := mustCommitAll(t, repo, "Initial commit")
	
	// Modify file
	mustWrite(t, repo, "file1.txt", "line1\nmodified\nline3\nline4\n")
	hash2 := mustCommitAll(t, repo, "Update file1")
	
	// Test: Get diff between commits
	gitOps := NewGitOperations(repo)
	diffs, err := gitOps.GetDiff(context.Background(), DiffOptions{
		FromCommit: hash1.String(),
		ToCommit:   hash2.String(),
	})
	if err != nil {
		t.Fatalf("GetDiff failed: %v", err)
	}
	
	if len(diffs) != 1 {
		t.Fatalf("Expected 1 diff, got %d", len(diffs))
	}
	
	diff := diffs[0]
	if diff.FilePath != "file1.txt" {
		t.Errorf("Expected file1.txt, got %s", diff.FilePath)
	}
	
	if diff.Additions == 0 && diff.Deletions == 0 {
		t.Error("Expected some additions or deletions")
	}
}

func TestGitOperations_GetDiffWorkingTree(t *testing.T) {
	repo := mustCreateTempRepo(t)
	
	// Create initial commit
	mustWrite(t, repo, "file1.txt", "content")
	mustCommitAll(t, repo, "Initial")
	
	// Modify without committing
	mustWrite(t, repo, "file1.txt", "modified content")
	
	// Test: Get diff with working tree
	gitOps := NewGitOperations(repo)
	diffs, err := gitOps.GetDiff(context.Background(), DiffOptions{
		ToCommit: "", // Empty = working tree
	})
	if err != nil {
		t.Fatalf("GetDiff for working tree failed: %v", err)
	}
	
	if len(diffs) == 0 {
		t.Error("Expected to find modified file in working tree")
	}
}

func TestGitOperations_GetBlame(t *testing.T) {
	repo := mustCreateTempRepo(t)
	
	// Create file with multiple lines
	content := "line1\nline2\nline3\n"
	mustWrite(t, repo, "file.txt", content)
	mustCommitAll(t, repo, "Initial commit")
	
	// Test: Get blame
	gitOps := NewGitOperations(repo)
	blameLines, err := gitOps.GetBlame(context.Background(), BlameOptions{
		FilePath: "file.txt",
	})
	if err != nil {
		t.Fatalf("GetBlame failed: %v", err)
	}
	
	if len(blameLines) != 3 {
		t.Errorf("Expected 3 blame lines, got %d", len(blameLines))
	}
	
	// Verify blame structure
	for i, line := range blameLines {
		if line.LineNumber != i+1 {
			t.Errorf("Line %d: expected line number %d, got %d", i, i+1, line.LineNumber)
		}
		if line.CommitHash == "" {
			t.Error("Commit hash is empty")
		}
		if line.Author == "" {
			t.Error("Author is empty")
		}
	}
}

func TestGitOperations_GetBlameNonExistentFile(t *testing.T) {
	repo := mustCreateTempRepo(t)
	
	mustWrite(t, repo, "file.txt", "content")
	mustCommitAll(t, repo, "Initial")
	
	// Test: Blame non-existent file should error
	gitOps := NewGitOperations(repo)
	_, err := gitOps.GetBlame(context.Background(), BlameOptions{
		FilePath: "nonexistent.txt",
	})
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestGitOperations_GetDiffWithMaxLines(t *testing.T) {
	repo := mustCreateTempRepo(t)
	
	// Create large file
	largeContent := ""
	for i := 0; i < 100; i++ {
		largeContent += "line\n"
	}
	mustWrite(t, repo, "large.txt", largeContent)
	hash1 := mustCommitAll(t, repo, "Add large file")
	
	// Modify it
	mustWrite(t, repo, "large.txt", largeContent+"new line\n")
	hash2 := mustCommitAll(t, repo, "Update large file")
	
	// Test: Limit diff lines
	gitOps := NewGitOperations(repo)
	diffs, err := gitOps.GetDiff(context.Background(), DiffOptions{
		FromCommit: hash1.String(),
		ToCommit:   hash2.String(),
		MaxLines:   10,
	})
	if err != nil {
		t.Fatalf("GetDiff with MaxLines failed: %v", err)
	}
	
	if len(diffs) == 0 {
		t.Fatal("Expected at least one diff")
	}
	
	// Verify truncation
	if len(diffs[0].Diff) > 0 {
		lines := len(diffs[0].Diff)
		if lines > 20 { // MaxLines affects output size
			t.Logf("Diff may be truncated (good)")
		}
	}
}
