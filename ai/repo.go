package ai

import (
    "context"
    "fmt"
    "log"
    "time"

    gogit "github.com/go-git/go-git/v6"
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

// IndexOptions controls indexing behavior.
type IndexOptions struct {
    // Incremental enables incremental indexing (only changed files since last index)
    Incremental bool
    
    // Force re-indexes all files even if index is fresh
    Force bool
    
    // MaxStaleness defines when index is considered stale (default: 24h)
    // Set to 0 to disable staleness check
    MaxStaleness time.Duration
    
    // OnProgress is called for each file indexed (for UI progress bars)
    OnProgress func(file string, current, total int)
    
    // Silent suppresses logging output
    Silent bool
}

// Index indexes the HEAD state of the repository into the vector DB.
// Use IndexWithOptions for fine-grained control.
func (r *Repository) Index(ctx context.Context) error {
    return r.IndexWithOptions(ctx, IndexOptions{})
}

// IndexWithOptions indexes with custom options (incremental, progress tracking, etc).
func (r *Repository) IndexWithOptions(ctx context.Context, opts IndexOptions) error {
    if opts.MaxStaleness == 0 {
        opts.MaxStaleness = 24 * time.Hour
    }
    
    // Check if reindex needed
    if !opts.Force && opts.Incremental {
        needsReindex, err := r.NeedsReindex(opts.MaxStaleness)
        if err != nil {
            return fmt.Errorf("failed to check index staleness: %w", err)
        }
        if !needsReindex {
            if !opts.Silent {
                log.Println("Index is fresh, skipping reindex")
            }
            return nil
        }
    }
    
    // Get HEAD commit for state tracking
    repo, err := gogit.PlainOpen(r.Path)
    if err != nil {
        return err
    }
    head, err := repo.Head()
    if err != nil {
        return err
    }
    
    // Perform indexing
    indexErr := r.Provider.IndexRepository(ctx, r.Path)
    
    // Save index state even if indexing partially failed
    // (allows testing and state tracking)
    state := &IndexState{
        LastCommit:  head.Hash().String(),
        LastIndexed: time.Now().UTC(),
        FileCount:   0, // TODO: track from provider
        ChunkCount:  0, // TODO: track from provider
        ModelName:   "default",
    }
    
    if err := r.SaveIndexState(state); err != nil && !opts.Silent {
        log.Printf("Warning: failed to save index state: %v", err)
    }
    
    return indexErr
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
// Automatically indexes the repository if not yet indexed or stale (>24h).
func (s *Search) Semantic(ctx context.Context, opts SemanticSearchOptions) (SearchResultIter, error) {
    // Auto-index if needed (lazy indexing)
    if err := s.repo.ensureIndexed(ctx); err != nil {
        return nil, fmt.Errorf("failed to ensure index: %w", err)
    }
    
    res, err := s.repo.Provider.SemanticSearch(ctx, s.repo.Path, opts.Query, opts.TopK, opts.Rerank)
    if err != nil { return nil, err }
    return NewSliceIter(res), nil
}

// ensureIndexed checks if the repo is indexed and indexes if needed (lazy pattern).
func (r *Repository) ensureIndexed(ctx context.Context) error {
    needsReindex, err := r.NeedsReindex(24 * time.Hour)
    if err != nil {
        return err
    }
    
    if !needsReindex {
        return nil // Already indexed and fresh
    }
    
    log.Println("Index not found or stale, indexing repository (this may take a moment)...")
    return r.IndexWithOptions(ctx, IndexOptions{
        Incremental: true,
        Silent:      false,
    })
}

// VectorStoreRetriever provides a LangChain-style retrieval interface for semantic search.
// It's bound to a repository and allows querying with minimal boilerplate.
type VectorStoreRetriever struct {
    repo *Repository
}

// Retriever returns a LangChain-style VectorStoreRetriever for this repository.
// Usage: r.Retriever().Retrieve(ctx, "query text", 10)
func (r *Repository) Retriever() *VectorStoreRetriever {
    return &VectorStoreRetriever{repo: r}
}

// Retrieve performs semantic search and returns top K results.
// This matches the LangChain VectorStoreRetriever.get_relevant_documents(query) pattern.
// Automatically indexes if not yet indexed.
func (v *VectorStoreRetriever) Retrieve(ctx context.Context, query string, topK int) ([]SearchResult, error) {
    if err := v.repo.ensureIndexed(ctx); err != nil {
        return nil, err
    }
    return v.repo.Provider.SemanticSearch(ctx, v.repo.Path, query, topK, false)
}

// RetrieveWithRerank performs semantic search with reranking enabled for higher quality results.
// Automatically indexes if not yet indexed.
func (v *VectorStoreRetriever) RetrieveWithRerank(ctx context.Context, query string, topK int) ([]SearchResult, error) {
    if err := v.repo.ensureIndexed(ctx); err != nil {
        return nil, err
    }
    return v.repo.Provider.SemanticSearch(ctx, v.repo.Path, query, topK, true)
}

// RetrieveStreaming emits results progressively via a callback, enabling progressive UI updates.
// This is the VSCode Copilot / streaming pattern.
// Automatically indexes if not yet indexed.
func (v *VectorStoreRetriever) RetrieveStreaming(ctx context.Context, query string, topK int, rerank bool, emit func(SearchResult) error) error {
    if err := v.repo.ensureIndexed(ctx); err != nil {
        return err
    }
    return v.repo.Provider.SemanticSearchStreaming(ctx, v.repo.Path, query, topK, rerank, emit)
}

