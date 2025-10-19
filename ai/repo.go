package ai

import (
    "context"
)

// Repository is a Git-like AI facade bound to a repo path.
// It exposes methods similar in style to go-git's Repository, but for AI use-cases.
type Repository struct {
    Path     string
    Provider *HybridProvider
}

// Open creates an AI repository facade with default hybrid provider.
func Open(path string) (*Repository, error) {
    return &Repository{
        Path:     path,
        Provider: DefaultProvider(),
    }, nil
}

// WithProvider wires a custom provider.
func (r *Repository) WithProvider(p *HybridProvider) *Repository {
    r.Provider = p
    return r
}

// Index indexes the HEAD state of the repository into the vector DB.
func (r *Repository) Index(ctx context.Context) error {
    return r.Provider.IndexRepository(ctx, r.Path)
}

// Search returns a fluent handle for keyword or semantic search.
func (r *Repository) Search() *Search { return &Search{repo: r} }

// Search groups keyword and semantic search entry points.
type Search struct{ repo *Repository }

// KeywordSearchOptions controls keyword search behavior.
type KeywordSearchOptions struct {
    Query         string
    CaseSensitive bool
    UseRegex      bool
    TopK          int // approximate cap; keyword search does not score perfectly, but we cap results
}

// SemanticSearchOptions controls semantic search behavior.
type SemanticSearchOptions struct {
    Query   string
    TopK    int
    Rerank  bool
}

// Keyword performs local keyword search over the repo (HEAD) and returns an iterator of results.
func (s *Search) Keyword(ctx context.Context, opts KeywordSearchOptions) (SearchResultIter, error) {
    res, err := keywordSearchRepo(ctx, s.repo.Path, opts)
    if err != nil { return nil, err }
    return NewSliceIter(res), nil
}

// Semantic performs embedding search via the provider and returns an iterator of results.
func (s *Search) Semantic(ctx context.Context, opts SemanticSearchOptions) (SearchResultIter, error) {
    res, err := s.repo.Provider.SemanticSearch(ctx, s.repo.Path, opts.Query, opts.TopK, opts.Rerank)
    if err != nil { return nil, err }
    return NewSliceIter(res), nil
}
