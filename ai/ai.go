package ai

import (
    "context"
    "sync"
)

// DefaultProvider returns a hybrid provider using HybridDefaults.
var (
    defaultProvider     *HybridProvider
    defaultProviderOnce sync.Once
)

func DefaultProvider() *HybridProvider {
    defaultProviderOnce.Do(func() {
        cfg := HybridDefaults()
        defaultProvider = NewHybridProvider(cfg)
    })
    return defaultProvider
}

func IndexRepo(ctx context.Context, repoPath string) error {
    return DefaultProvider().IndexRepository(ctx, repoPath)
}

func SemanticSearchRepo(ctx context.Context, repoPath, query string, topK int, rerank bool) ([]SearchResult, error) {
    return DefaultProvider().SemanticSearch(ctx, repoPath, query, topK, rerank)
}

// KeywordSearchRepo performs fast keyword search over the repository HEAD.
// It returns results matching the query with optional regex and case-sensitivity.
// This is the top-level wrapper for quick access, similar to SemanticSearchRepo.
func KeywordSearchRepo(ctx context.Context, repoPath, query string, topK int) ([]SearchResult, error) {
    opts := KeywordSearchOptions{
        Query:         query,
        CaseSensitive: false,
        UseRegex:      false,
        TopK:          topK,
    }
    return keywordSearchRepo(ctx, repoPath, opts)
}

// KeywordSearchRepoWithOptions provides full control over keyword search behavior.
func KeywordSearchRepoWithOptions(ctx context.Context, repoPath string, opts KeywordSearchOptions) ([]SearchResult, error) {
    return keywordSearchRepo(ctx, repoPath, opts)
}

// SemanticSearchRepoStreaming performs semantic search with progressive result emission (VSCode Copilot pattern).
// Results are emitted via the callback as they're retrieved and processed, enabling incremental UI updates.
// The search is cancellable via context and doesn't load all results into memory before returning.
func SemanticSearchRepoStreaming(ctx context.Context, repoPath, query string, topK int, rerank bool, emit func(SearchResult) error) error {
    return DefaultProvider().SemanticSearchStreaming(ctx, repoPath, query, topK, rerank, emit)
}

