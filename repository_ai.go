package git

import (
	"context"

	"github.com/go-git/go-git/v6/ai"
)

// AIProvider returns or initializes the AI provider for this repository.
// Lazy initialization uses hybrid defaults (local embeddings + cloud LLM).
func (r *Repository) AIProvider() *ai.HybridProvider {
	if r.aiProvider == nil {
		r.aiProvider = ai.DefaultProvider()
	}
	return r.aiProvider.(*ai.HybridProvider)
}

// SetAIProvider assigns a custom AI provider (e.g., for testing or custom config).
func (r *Repository) SetAIProvider(provider *ai.HybridProvider) {
	r.aiProvider = provider
}

// AIIndex indexes the repository for semantic search (one-time setup).
// Like git add, this prepares content for later AI operations.
func (r *Repository) AIIndex(ctx context.Context) error {
	wt, err := r.Worktree()
	if err != nil {
		return err
	}
	aiRepo, err := ai.Open(wt.Filesystem.Root())
	if err != nil {
		return err
	}
	aiRepo.Provider = r.AIProvider()
	return aiRepo.Index(ctx)
}

// AIIndexWithOptions indexes with custom options (incremental, progress callbacks).
func (r *Repository) AIIndexWithOptions(ctx context.Context, opts ai.IndexOptions) error {
	wt, err := r.Worktree()
	if err != nil {
		return err
	}
	aiRepo, err := ai.Open(wt.Filesystem.Root())
	if err != nil {
		return err
	}
	aiRepo.Provider = r.AIProvider()
	return aiRepo.IndexWithOptions(ctx, opts)
}

// AIKeywordSearch performs local keyword/pattern search (like git grep).
// Returns results matching the pattern with file paths and line numbers.
func (r *Repository) AIKeywordSearch(ctx context.Context, opts ai.KeywordSearchOptions) (ai.SearchResultIter, error) {
	wt, err := r.Worktree()
	if err != nil {
		return nil, err
	}
	aiRepo, err := ai.Open(wt.Filesystem.Root())
	if err != nil {
		return nil, err
	}
	aiRepo.Provider = r.AIProvider()
	return aiRepo.Search().Keyword(ctx, opts)
}

// AISemanticSearch performs AI-powered semantic code search.
// Like git log --grep, but understands code meaning instead of just text.
func (r *Repository) AISemanticSearch(ctx context.Context, opts ai.SemanticSearchOptions) (ai.SearchResultIter, error) {
	wt, err := r.Worktree()
	if err != nil {
		return nil, err
	}
	aiRepo, err := ai.Open(wt.Filesystem.Root())
	if err != nil {
		return nil, err
	}
	aiRepo.Provider = r.AIProvider()
	return aiRepo.Search().Semantic(ctx, opts)
}

// AISearch returns a fluent search interface (like r.Log() or r.CommitObjects()).
func (r *Repository) AISearch() *ai.Search {
	wt, err := r.Worktree()
	if err != nil {
		return nil
	}
	aiRepo, err := ai.Open(wt.Filesystem.Root())
	if err != nil {
		return nil
	}
	aiRepo.Provider = r.AIProvider()
	return aiRepo.Search()
}
