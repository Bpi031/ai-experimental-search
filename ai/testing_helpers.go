package ai

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	git "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
)

// mustCreateTempRepo creates a temporary git repository for testing
func mustCreateTempRepo(t *testing.T) *git.Repository {
	t.Helper()
	
	tmpDir := t.TempDir()
	
	repo, err := git.PlainInit(tmpDir, false)
	if err != nil {
		t.Fatalf("Failed to init temp repo: %v", err)
	}
	
	// Set default config
	cfg, err := repo.Config()
	if err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}
	
	cfg.User.Name = "Test User"
	cfg.User.Email = "test@example.com"
	
	if err := repo.SetConfig(cfg); err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}
	
	return repo
}

// cleanupRepo is a no-op since t.TempDir() handles cleanup automatically
func cleanupRepo(t *testing.T, repo *git.Repository) {
	// No-op: t.TempDir() handles cleanup automatically
}

// mustWrite writes content to a file in the repository
func mustWrite(t *testing.T, repo *git.Repository, filename string, content string) {
	t.Helper()
	
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}
	
	filePath := filepath.Join(wt.Filesystem.Root(), filename)
	
	// Create parent directories if needed
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create directories: %v", err)
	}
	
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file %s: %v", filename, err)
	}
}

// mustCommitAll stages all changes and commits them
func mustCommitAll(t *testing.T, repo *git.Repository, message string) plumbing.Hash {
	t.Helper()
	
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}
	
	// Add all changes
	if err := wt.AddGlob("."); err != nil {
		t.Fatalf("Failed to add files: %v", err)
	}
	
	// Commit
	hash, err := wt.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	
	return hash
}

// mustGetCommit gets a commit object by hash
func mustGetCommit(t *testing.T, repo *git.Repository, hash plumbing.Hash) *object.Commit {
	t.Helper()
	
	commit, err := repo.CommitObject(hash)
	if err != nil {
		t.Fatalf("Failed to get commit %s: %v", hash, err)
	}
	
	return commit
}

// mustCreateBranch creates a new branch
func mustCreateBranch(t *testing.T, repo *git.Repository, name string) {
	t.Helper()
	
	head, err := repo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD: %v", err)
	}
	
	ref := plumbing.NewHashReference(plumbing.NewBranchReferenceName(name), head.Hash())
	if err := repo.Storer.SetReference(ref); err != nil {
		t.Fatalf("Failed to create branch %s: %v", name, err)
	}
}

// mustCheckoutBranch checks out a branch
func mustCheckoutBranch(t *testing.T, repo *git.Repository, name string) {
	t.Helper()
	
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}
	
	if err := wt.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(name),
	}); err != nil {
		t.Fatalf("Failed to checkout branch %s: %v", name, err)
	}
}

// mustGetFileContent reads a file from the repository
func mustGetFileContent(t *testing.T, repo *git.Repository, filename string) string {
	t.Helper()
	
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}
	
	filePath := filepath.Join(wt.Filesystem.Root(), filename)
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", filename, err)
	}
	
	return string(content)
}

// mustDeleteFile deletes a file from the repository
func mustDeleteFile(t *testing.T, repo *git.Repository, filename string) {
	t.Helper()
	
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}
	
	filePath := filepath.Join(wt.Filesystem.Root(), filename)
	if err := os.Remove(filePath); err != nil {
		t.Fatalf("Failed to delete file %s: %v", filename, err)
	}
}

// assertFileExists checks that a file exists in the repository
func assertFileExists(t *testing.T, repo *git.Repository, filename string) {
	t.Helper()
	
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}
	
	filePath := filepath.Join(wt.Filesystem.Root(), filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("File %s does not exist", filename)
	}
}

// assertFileNotExists checks that a file does not exist in the repository
func assertFileNotExists(t *testing.T, repo *git.Repository, filename string) {
	t.Helper()
	
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}
	
	filePath := filepath.Join(wt.Filesystem.Root(), filename)
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("File %s should not exist", filename)
	}
}
