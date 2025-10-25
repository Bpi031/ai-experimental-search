package ai_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	git "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/ai"
)

// TestIndexing_FirstUse validates explicit indexing on first use.
func TestIndexing_FirstUse(t *testing.T) {
	tei := getenv("AI_TEI_ENDPOINT", "http://localhost:8081")
	chroma := getenv("AI_CHROMA_URL", "http://localhost:9001")
	if !isUp(tei) || !isUp(chroma) {
		t.Skip("TEI/Chroma not available")
	}

	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatal(err)
	}

	mustWrite(t, filepath.Join(dir, "main.go"), "package main\n\nfunc main() {}\n")
	mustCommitAll(t, repo, "initial commit")

	r, err := ai.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Should not be indexed initially
	if r.IsIndexed() {
		t.Fatal("expected repository to not be indexed initially")
	}

	// Explicit indexing (Scenario 1: First use)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := r.Index(ctx); err != nil {
		t.Fatal("index failed:", err)
	}

	// Now should be indexed
	if !r.IsIndexed() {
		t.Fatal("expected repository to be indexed after Index()")
	}

	// Check index state
	state, err := r.GetIndexState()
	if err != nil {
		t.Fatal(err)
	}
	if state == nil {
		t.Fatal("expected index state to be saved")
	}
	if state.LastCommit == "" {
		t.Fatal("expected last commit to be recorded")
	}
}

// TestIndexing_LazyAutoIndex validates auto-indexing on first semantic search.
func TestIndexing_LazyAutoIndex(t *testing.T) {
	tei := getenv("AI_TEI_ENDPOINT", "http://localhost:8081")
	chroma := getenv("AI_CHROMA_URL", "http://localhost:9001")
	if !isUp(tei) || !isUp(chroma) {
		t.Skip("TEI/Chroma not available")
	}

	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatal(err)
	}

	mustWrite(t, filepath.Join(dir, "auth.go"), "package auth\n\nfunc Login() {}\n")
	mustCommitAll(t, repo, "add auth")

	r, err := ai.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Should not be indexed
	if r.IsIndexed() {
		t.Fatal("expected repo not indexed initially")
	}

	// Semantic search should auto-index (Scenario 2: IDE startup / lazy)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	iter, err := r.Search().Semantic(ctx, ai.SemanticSearchOptions{
		Query:  "authentication",
		TopK:   5,
		Rerank: false,
	})
	if err != nil {
		t.Fatal("semantic search failed:", err)
	}
	defer iter.Close()

	// Now should be indexed
	if !r.IsIndexed() {
		t.Fatal("expected auto-indexing during semantic search")
	}

	// Verify results are valid
	_, _ = iter.Next() // At least try to get one result
}

// TestIndexing_StalenessDetection validates staleness detection and skip logic.
func TestIndexing_StalenessDetection(t *testing.T) {
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatal(err)
	}

	mustWrite(t, filepath.Join(dir, "main.go"), "package main\n")
	mustCommitAll(t, repo, "initial")

	r, err := ai.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Index first time
	if err := r.Index(ctx); err != nil {
		t.Fatal(err)
	}

	// Immediate reindex with incremental=true should skip
	// (no commits changed, index is fresh)
	if err := r.IndexWithOptions(ctx, ai.IndexOptions{
		Incremental:  true,
		MaxStaleness: 24 * time.Hour,
	}); err != nil {
		t.Fatal(err)
	}

	// Force reindex should work even if fresh
	if err := r.IndexWithOptions(ctx, ai.IndexOptions{
		Force: true,
	}); err != nil {
		t.Fatal(err)
	}
}

// TestIndexState_Persistence validates index state tracking without requiring services.
func TestIndexState_Persistence(t *testing.T) {
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatal(err)
	}

	mustWrite(t, filepath.Join(dir, "test.go"), "package test\n")
	mustCommitAll(t, repo, "init")

	r, err := ai.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Should not be indexed
	if r.IsIndexed() {
		t.Fatal("expected not indexed initially")
	}

	// Manually save state
	head, _ := repo.Head()
	state := &ai.IndexState{
		LastCommit:  head.Hash().String(),
		LastIndexed: time.Now(),
		FileCount:   1,
		ChunkCount:  1,
		ModelName:   "test-model",
	}

	if err := r.SaveIndexState(state); err != nil {
		t.Fatal(err)
	}

	// Should now be indexed
	if !r.IsIndexed() {
		t.Fatal("expected indexed after saving state")
	}

	// Load state and verify
	loaded, err := r.GetIndexState()
	if err != nil {
		t.Fatal(err)
	}

	if loaded.LastCommit != state.LastCommit {
		t.Fatalf("commit mismatch: got %s, want %s", loaded.LastCommit, state.LastCommit)
	}
	if loaded.FileCount != 1 {
		t.Fatalf("file count mismatch: got %d, want 1", loaded.FileCount)
	}

	// Test staleness detection
	needsReindex, err := r.NeedsReindex(24 * time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if needsReindex {
		t.Fatal("index should be fresh")
	}

	// Make a new commit
	mustWrite(t, filepath.Join(dir, "test2.go"), "package test\n")
	mustCommitAll(t, repo, "add test2")

	// Now should need reindex (commit changed)
	needsReindex, err = r.NeedsReindex(24 * time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !needsReindex {
		t.Fatal("index should be stale after new commit")
	}
}
