package gitai
// Package main demonstrates the 5 indexing strategies for ai-experimental-search.
//
// Scenarios:
// 1. First use: Explicit repo.Index() or auto-index on first search
// 2. After git pull/commit: Background watcher checks every 5-10 min
// 3. IDE startup: Lazy check - if stale (>1 day) → reindex
// 4. CI/CD: Explicit index after merge to main
// 5. CLI tool: git ai status shows staleness, user runs git ai reindex
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-git/go-git/v6/ai"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	fs := flag.NewFlagSet(command, flag.ExitOnError)

	switch command {
	case "init":
		initCmd(fs)
	case "status":
		statusCmd(fs)
	case "reindex":
		reindexCmd(fs)
	case "search":
		searchCmd(fs)
	case "watch":
		watchCmd(fs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage: git-ai <command> [options]

Commands:
  init              Initialize semantic search (index repository)
  status            Show index status (last indexed, staleness)
  reindex           Re-index repository (use --incremental for changed files only)
  search <query>    Semantic search over repository
  watch             Start background watcher (for long-running processes)

Examples:
  # First time setup (Scenario 1: First use)
  git-ai init
  
  # Check if index is stale (Scenario 5: CLI tool)
  git-ai status
  
  # Re-index after git pull (Scenario 3: After commit)
  git pull
  git-ai reindex --incremental
  
  # Search (auto-indexes if needed)
  git-ai search "OAuth token refresh"
  
  # Start background watcher (Scenario 2: Background watcher)
  git-ai watch --interval=5m
`)
}

// initCmd implements Scenario 1: First use - explicit indexing
func initCmd(fs *flag.FlagSet) {
	force := fs.Bool("force", false, "Force reindex even if already indexed")
	fs.Parse(os.Args[2:])

	repoPath := "."
	if fs.NArg() > 0 {
		repoPath = fs.Arg(0)
	}

	repo, err := ai.Open(repoPath)
	if err != nil {
		log.Fatal(err)
	}

	if repo.IsIndexed() && !*force {
		fmt.Println("✓ Repository is already indexed")
		fmt.Println("  Use --force to reindex")
		return
	}

	fmt.Println("Indexing repository (this may take a minute)...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	start := time.Now()
	if err := repo.Index(ctx); err != nil {
		log.Fatalf("Indexing failed: %v", err)
	}

	fmt.Printf("✓ Semantic search initialized in %v\n", time.Since(start).Round(time.Second))
	printStatus(repo)
}

// statusCmd implements Scenario 5: CLI tool - show index status
func statusCmd(fs *flag.FlagSet) {
	fs.Parse(os.Args[2:])

	repoPath := "."
	if fs.NArg() > 0 {
		repoPath = fs.Arg(0)
	}

	repo, err := ai.Open(repoPath)
	if err != nil {
		log.Fatal(err)
	}

	printStatus(repo)
}

func printStatus(repo *ai.Repository) {
	if !repo.IsIndexed() {
		fmt.Println("✗ Repository is not indexed")
		fmt.Println("  Run: git-ai init")
		return
	}

	state, err := repo.GetIndexState()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n📊 Index Status:")
	fmt.Printf("  ✓ Indexed: %d files\n", state.FileCount)
	fmt.Printf("  ✓ Last updated: %v ago (commit %s)\n",
		time.Since(state.LastIndexed).Round(time.Second),
		state.LastCommit[:7])
	fmt.Printf("  ✓ Embedding model: %s\n", state.ModelName)

	needsReindex, _ := repo.NeedsReindex(24 * time.Hour)
	if needsReindex {
		fmt.Println("\n⚠️  Index is stale (>24h or HEAD changed)")
		fmt.Println("   Run: git-ai reindex --incremental")
	}
}

// reindexCmd implements Scenario 3: After git pull/commit - incremental reindex
func reindexCmd(fs *flag.FlagSet) {
	incremental := fs.Bool("incremental", false, "Only reindex changed files")
	force := fs.Bool("force", false, "Force reindex even if fresh")
	fs.Parse(os.Args[2:])

	repoPath := "."
	if fs.NArg() > 0 {
		repoPath = fs.Arg(0)
	}

	repo, err := ai.Open(repoPath)
	if err != nil {
		log.Fatal(err)
	}

	opts := ai.IndexOptions{
		Incremental: *incremental,
		Force:       *force,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	fmt.Println("Re-indexing repository...")
	start := time.Now()

	if err := repo.IndexWithOptions(ctx, opts); err != nil {
		log.Fatalf("Reindex failed: %v", err)
	}

	fmt.Printf("✓ Reindex complete in %v\n", time.Since(start).Round(time.Second))
}

// searchCmd implements lazy auto-indexing (Scenario 3: IDE startup)
func searchCmd(fs *flag.FlagSet) {
	topK := fs.Int("top", 10, "Number of results to return")
	rerank := fs.Bool("rerank", false, "Enable reranking for higher quality")
	fs.Parse(os.Args[2:])

	if fs.NArg() < 1 {
		fmt.Println("Usage: git-ai search <query>")
		os.Exit(1)
	}

	query := fs.Arg(0)
	repoPath := "."

	repo, err := ai.Open(repoPath)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Auto-indexes if not indexed or stale (lazy pattern)
	fmt.Printf("Searching for: %s\n\n", query)
	iter, err := repo.Search().Semantic(ctx, ai.SemanticSearchOptions{
		Query:  query,
		TopK:   *topK,
		Rerank: *rerank,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer iter.Close()

	count := 0
	for {
		result, err := iter.Next()
		if err != nil {
			break
		}
		count++
		fmt.Printf("%d. %s:%d (score: %.3f)\n",
			count,
			result.Chunk.FilePath,
			result.Chunk.StartLine,
			result.Score)
		fmt.Printf("   %s\n\n", result.Chunk.Content)
	}

	if count == 0 {
		fmt.Println("No results found")
	}
}

// watchCmd implements Scenario 2: Background watcher
func watchCmd(fs *flag.FlagSet) {
	interval := fs.Duration("interval", 5*time.Minute, "Check interval (e.g., 5m, 10m)")
	fs.Parse(os.Args[2:])

	repoPath := "."
	if fs.NArg() > 0 {
		repoPath = fs.Arg(0)
	}

	repo, err := ai.Open(repoPath)
	if err != nil {
		log.Fatal(err)
	}

	if !repo.IsIndexed() {
		fmt.Println("Repository not indexed. Indexing now...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		if err := repo.Index(ctx); err != nil {
			cancel()
			log.Fatalf("Initial indexing failed: %v", err)
		}
		cancel()
	}

	fmt.Printf("Starting background watcher (checking every %v)\n", *interval)
	fmt.Println("Press Ctrl+C to stop")

	watcher := ai.NewRepoWatcher(repo, *interval)
	ctx := context.Background()
	watcher.Start(ctx)
}
