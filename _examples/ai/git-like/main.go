package main

import (
	"context"
	"fmt"
	"log"
	"time"

	git "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/ai"
	"github.com/go-git/go-git/v6/plumbing/object"
)

func main() {
	ctx := context.Background()

	// Open repository (standard go-git)
	repo, err := git.PlainOpen(".")
	if err != nil {
		log.Fatal(err)
	}

	// === Standard Git Operations ===
	
	// Get worktree
	w, err := repo.Worktree()
	if err != nil {
		log.Fatal(err)
	}

	// Check status
	status, err := w.Status()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Git status:", status)

	// === AI Operations (Git-like API) ===

	// Index repository for AI search (like git add)
	fmt.Println("\nIndexing repository for AI search...")
	if err := repo.AIIndex(ctx); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Index complete!")

	// Keyword search (like git grep)
	fmt.Println("\n=== Keyword Search (like git grep) ===")
	kwIter, err := repo.AIKeywordSearch(ctx, ai.KeywordSearchOptions{
		Query: "IndexRepository",
		TopK:  5,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer kwIter.Close()

	err = kwIter.ForEach(func(result *ai.SearchResult) error {
		fmt.Printf("  %s:%d - %s\n",
			result.Chunk.FilePath,
			result.Chunk.StartLine,
			result.Chunk.Content[:min(80, len(result.Chunk.Content))])
		return nil
	})
	if err != nil {
		log.Printf("keyword search: %v", err)
	}

	// Semantic search (like git log --grep with understanding)
	fmt.Println("\n=== Semantic Search (AI-powered) ===")
	semIter, err := repo.AISemanticSearch(ctx, ai.SemanticSearchOptions{
		Query:  "index repository for search",
		TopK:   3,
		Rerank: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer semIter.Close()

	err = semIter.ForEach(func(result *ai.SearchResult) error {
		fmt.Printf("  %.3f - %s:%d-%d\n",
			result.Score,
			result.Chunk.FilePath,
			result.Chunk.StartLine,
			result.Chunk.EndLine)
		return nil
	})
	if err != nil {
		log.Printf("semantic search: %v", err)
	}

	// === Mix Git and AI Operations ===

	// Get commit history (standard git)
	ref, err := repo.Head()
	if err != nil {
		log.Fatal(err)
	}

	commitIter, err := repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n=== Recent Commits (standard git) ===")
	count := 0
	err = commitIter.ForEach(func(c *object.Commit) error {
		if count >= 3 {
			return fmt.Errorf("done")
		}
		fmt.Printf("  %s - %s\n", c.Hash.String()[:7], c.Message[:min(50, len(c.Message))])
		count++
		return nil
	})
	if err != nil && err.Error() != "done" {
		log.Fatal(err)
	}

	// Fluent API (like repo.Log() or repo.CommitObjects())
	fmt.Println("\n=== Fluent AI Search API ===")
	search := repo.AISearch()
	if search != nil {
		kwResults, _ := search.Keyword(ctx, ai.KeywordSearchOptions{
			Query: "Repository",
			TopK:  2,
		})
		if kwResults != nil {
			defer kwResults.Close()
			kwResults.ForEach(func(r *ai.SearchResult) error {
				fmt.Printf("  Found: %s\n", r.Chunk.FilePath)
				return nil
			})
		}
	}

	fmt.Println("\n=== All operations completed ===")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
