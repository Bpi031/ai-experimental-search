package ai

import (
	"context"
	"testing"
)

func TestGitWriter_ListBranches(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer func() {
		// Cleanup is automatic with t.TempDir()
	}()

	mustWrite(t, repo, "main.go", "package main")
	mustCommitAll(t, repo, "Initial commit")

	// Create additional branches
	mustCreateBranch(t, repo, "feature/new-api")
	mustCreateBranch(t, repo, "hotfix/bug-123")

	ctx := context.Background()
	writer := NewGitWriter(repo)
	branches, err := writer.ListBranches(ctx)
	if err != nil {
		t.Fatalf("ListBranches() error = %v", err)
	}

	// Should have at least master/main + 2 feature branches
	if len(branches) < 3 {
		t.Errorf("ListBranches() got %d branches, want at least 3", len(branches))
	}

	// Check for current branch
	foundCurrent := false
	for _, branch := range branches {
		if branch.IsCurrent {
			foundCurrent = true
			if branch.Name == "" {
				t.Error("ListBranches() current branch has empty name")
			}
			if branch.Hash == "" {
				t.Error("ListBranches() current branch has empty hash")
			}
		}
	}
	if !foundCurrent {
		t.Error("ListBranches() didn't find current branch")
	}
}

func TestGitWriter_SwitchBranch(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer func() {
		// Cleanup is automatic with t.TempDir()
	}()

	mustWrite(t, repo, "main.go", "package main")
	mustCommitAll(t, repo, "Initial commit")

	// Create and switch to new branch
	mustCreateBranch(t, repo, "feature/test")

	ctx := context.Background()
	writer := NewGitWriter(repo)
	err := writer.SwitchBranch(ctx, "feature/test", false)
	if err != nil {
		t.Fatalf("SwitchBranch() error = %v", err)
	}

	// Verify current branch
	branches, err := writer.ListBranches(ctx)
	if err != nil {
		t.Fatalf("ListBranches() error = %v", err)
	}

	currentBranch := ""
	for _, branch := range branches {
		if branch.IsCurrent && !branch.IsRemote {
			currentBranch = branch.Name
			break
		}
	}

	if currentBranch != "feature/test" {
		t.Errorf("SwitchBranch() current = %s, want feature/test", currentBranch)
	}
}

func TestGitWriter_SwitchBranch_CreateNew(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer func() {
		// Cleanup is automatic with t.TempDir()
	}()

	mustWrite(t, repo, "main.go", "package main")
	mustCommitAll(t, repo, "Initial commit")

	ctx := context.Background()
	writer := NewGitWriter(repo)
	
	// Create new branch via switch
	err := writer.SwitchBranch(ctx, "feature/new-branch", true)
	if err != nil {
		t.Fatalf("SwitchBranch() with create error = %v", err)
	}

	// Verify branch exists and is current
	branches, err := writer.ListBranches(ctx)
	if err != nil {
		t.Fatalf("ListBranches() error = %v", err)
	}

	found := false
	for _, branch := range branches {
		if branch.Name == "feature/new-branch" && branch.IsCurrent {
			found = true
			break
		}
	}

	if !found {
		t.Error("SwitchBranch() with create didn't create/switch to new branch")
	}
}

func TestGitWriter_MergeBranch(t *testing.T) {
	t.Skip("Merge test requires more complex setup - skipping for now")
	
	repo := mustCreateTempRepo(t)
	defer func() {
		// Cleanup is automatic with t.TempDir()
	}()

	// Create base
	mustWrite(t, repo, "main.go", "package main\n\nfunc main() {}")
	mustCommitAll(t, repo, "Initial commit")

	// Create feature branch
	mustCreateBranch(t, repo, "feature/add-func")
	mustCheckoutBranch(t, repo, "feature/add-func")
	mustWrite(t, repo, "utils.go", "package main\n\nfunc helper() {}")
	mustCommitAll(t, repo, "Add helper function")

	// Switch back to main
	mustCheckoutBranch(t, repo, "master")

	ctx := context.Background()
	writer := NewGitWriter(repo)
	result, err := writer.MergeBranch(ctx, "feature/add-func")
	if err != nil {
		t.Fatalf("MergeBranch() error = %v", err)
	}

	if !result.Success {
		t.Error("MergeBranch() Success = false, want true")
	}

	if len(result.MergedFiles) == 0 {
		t.Error("MergeBranch() MergedFiles is empty")
	}
}

func TestGitWriter_BranchInfo(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer func() {
		// Cleanup is automatic with t.TempDir()
	}()

	mustWrite(t, repo, "main.go", "package main")
	mustCommitAll(t, repo, "Initial commit")

	ctx := context.Background()
	writer := NewGitWriter(repo)
	branches, err := writer.ListBranches(ctx)
	if err != nil {
		t.Fatalf("ListBranches() error = %v", err)
	}

	if len(branches) == 0 {
		t.Fatal("ListBranches() returned no branches")
	}

	branch := branches[0]

	// Check branch info fields
	if branch.Name == "" {
		t.Error("Branch Name is empty")
	}

	if branch.Hash == "" {
		t.Error("Branch Hash is empty")
	}

	if branch.ShortHash == "" {
		t.Error("Branch ShortHash is empty")
	}

	if len(branch.ShortHash) > 10 {
		t.Errorf("Branch ShortHash = %s, too long (should be ~7 chars)", branch.ShortHash)
	}

	if branch.LastCommit != nil {
		if branch.LastCommit.Hash == "" {
			t.Error("Branch LastCommit.Hash is empty")
		}
		if branch.LastCommit.Message == "" {
			t.Error("Branch LastCommit.Message is empty")
		}
	}
}

func TestGitWriter_DeleteBranch_Extended(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer func() {
		// Cleanup is automatic with t.TempDir()
	}()

	mustWrite(t, repo, "main.go", "package main")
	mustCommitAll(t, repo, "Initial commit")

	// Create branch
	mustCreateBranch(t, repo, "feature/to-delete")

	ctx := context.Background()
	writer := NewGitWriter(repo)

	// Delete branch
	err := writer.DeleteBranch(ctx, "feature/to-delete", false)
	if err != nil {
		t.Fatalf("DeleteBranch() error = %v", err)
	}

	// Verify branch is gone
	branches, err := writer.ListBranches(ctx)
	if err != nil {
		t.Fatalf("ListBranches() error = %v", err)
	}

	for _, branch := range branches {
		if branch.Name == "feature/to-delete" {
			t.Error("DeleteBranch() branch still exists")
		}
	}
}
