package ai

import "context"

// DefaultProvider returns a hybrid provider using HybridDefaults.
func DefaultProvider() *HybridProvider {
    cfg := HybridDefaults()
    return NewHybridProvider(cfg)
}

func IndexRepo(ctx context.Context, repoPath string) error {
    return DefaultProvider().IndexRepository(ctx, repoPath)
}

func SemanticSearchRepo(ctx context.Context, repoPath, query string, topK int, rerank bool) ([]SearchResult, error) {
    return DefaultProvider().SemanticSearch(ctx, repoPath, query, topK, rerank)
}
