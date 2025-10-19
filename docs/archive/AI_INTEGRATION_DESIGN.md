# AI Integration Design — go-git

## Overview

This document describes the design for adding AI-powered search and code update capabilities to go-git. The goal is to enable AI agents to:

1. **Semantic search** — search repository content using natural language queries
2. **Code search** — find code patterns, functions, and symbols using AI understanding
3. **AI-driven updates** — apply code changes suggested by AI with proper git commit workflow

## Architecture

### High-level design

```
┌─────────────────────────────────────────────────────────────┐
│                      Repository                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  Existing API: Clone, Commit, Log, Object, etc.        │ │
│  └────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  NEW: AI Integration Layer                             │ │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐  │ │
│  │  │ AISemanticSearch │ AICodeSearch │ │ AIUpdateCode │  │ │
│  │  └──────────────┘ └──────────────┘ └──────────────┘  │ │
│  └────────────────────────────────────────────────────────┘ │
│           │                  │                  │           │
└───────────┼──────────────────┼──────────────────┼───────────┘
            │                  │                  │
            ▼                  ▼                  ▼
    ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
    │ AI Provider  │  │ AI Provider  │  │ Worktree +   │
    │ (OpenAI,     │  │ (embeddings, │  │ Commit Flow  │
    │  Anthropic,  │  │  AST parse)  │  │              │
    │  local LLM)  │  │              │  │              │
    └──────────────┘  └──────────────┘  └──────────────┘
```

### Key design principles

1. **Pluggable AI backends** — use an interface so users can bring their own AI provider (OpenAI, Anthropic, local models, etc.)
2. **Non-invasive** — add new methods to `Repository` without breaking existing API
3. **Git-native** — AI updates should create proper commits with messages, author, and history
4. **Lazy evaluation** — only read/index content when needed (avoid scanning entire repos upfront)
5. **Error handling** — AI operations can fail (rate limits, model errors); provide clear errors and fallbacks

## API Design

### 1. AIProvider interface (pluggable backend)

```go
// ai_provider.go
package git

import (
	"context"
	"io"
)

// AIProvider is the interface for pluggable AI backends that power
// semantic search, code search, and code generation.
type AIProvider interface {
	// SemanticSearch searches repository content using natural language.
	// Returns a ranked list of file paths and relevant snippets.
	SemanticSearch(ctx context.Context, query string, files []FileContent) ([]SearchResult, error)
	
	// CodeSearch finds code patterns using AI understanding of syntax and semantics.
	// Returns matches with location and context.
	CodeSearch(ctx context.Context, query string, files []FileContent) ([]CodeMatch, error)
	
	// GenerateCodeUpdate suggests code changes based on a natural language request.
	// Returns a set of file edits to apply.
	GenerateCodeUpdate(ctx context.Context, request string, files []FileContent) ([]FileEdit, error)
}

// FileContent represents a file's content for AI processing.
type FileContent struct {
	Path    string
	Content string
	Hash    plumbing.Hash // for tracking changes
}

// SearchResult represents a semantic search match.
type SearchResult struct {
	FilePath string
	Snippet  string
	Score    float64 // relevance score (0-1)
	LineStart int
	LineEnd   int
}

// CodeMatch represents a code search match.
type CodeMatch struct {
	FilePath   string
	FunctionName string // if applicable
	LineStart  int
	LineEnd    int
	CodeSnippet string
	Description string // AI-generated explanation
}

// FileEdit represents a code change to apply.
type FileEdit struct {
	FilePath string
	OldContent string // empty if new file
	NewContent string
	Description string // AI explanation of the change
}

// AIUpdateOptions configures AI-driven code updates.
type AIUpdateOptions struct {
	// CommitMessage is the commit message for the changes.
	CommitMessage string
	// Author is the commit author (defaults to repo config).
	Author *object.Signature
	// AllowNewFiles permits creating new files.
	AllowNewFiles bool
	// AllowDeleteFiles permits deleting files.
	AllowDeleteFiles bool
	// DryRun returns the changes without applying them.
	DryRun bool
}
```

### 2. Repository AI methods

Add these methods to `Repository`:

```go
// ai_search.go
package git

import (
	"context"
	"fmt"
	"io"
	
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
)

// SetAIProvider configures the AI backend for this repository.
func (r *Repository) SetAIProvider(provider AIProvider) {
	r.aiProvider = provider
}

// AISemanticSearch searches the repository using natural language.
// It reads the current HEAD tree and searches all files.
func (r *Repository) AISemanticSearch(ctx context.Context, query string) ([]SearchResult, error) {
	if r.aiProvider == nil {
		return nil, fmt.Errorf("AI provider not configured; call SetAIProvider first")
	}
	
	files, err := r.readAllFiles(ctx)
	if err != nil {
		return nil, err
	}
	
	return r.aiProvider.SemanticSearch(ctx, query, files)
}

// AICodeSearch finds code patterns using AI understanding.
func (r *Repository) AICodeSearch(ctx context.Context, query string) ([]CodeMatch, error) {
	if r.aiProvider == nil {
		return nil, fmt.Errorf("AI provider not configured; call SetAIProvider first")
	}
	
	files, err := r.readAllFiles(ctx)
	if err != nil {
		return nil, err
	}
	
	return r.aiProvider.CodeSearch(ctx, query, files)
}

// AIUpdateCode applies AI-suggested code changes and commits them.
func (r *Repository) AIUpdateCode(ctx context.Context, request string, opts *AIUpdateOptions) (plumbing.Hash, error) {
	if r.aiProvider == nil {
		return plumbing.ZeroHash, fmt.Errorf("AI provider not configured")
	}
	
	if opts == nil {
		opts = &AIUpdateOptions{}
	}
	
	// Read current files
	files, err := r.readAllFiles(ctx)
	if err != nil {
		return plumbing.ZeroHash, err
	}
	
	// Get AI suggestions
	edits, err := r.aiProvider.GenerateCodeUpdate(ctx, request, files)
	if err != nil {
		return plumbing.ZeroHash, err
	}
	
	if opts.DryRun {
		// Return edits without applying
		// (caller can inspect edits before applying)
		return plumbing.ZeroHash, nil
	}
	
	// Apply edits to worktree
	wt, err := r.Worktree()
	if err != nil {
		return plumbing.ZeroHash, err
	}
	
	for _, edit := range edits {
		if err := applyEdit(wt, edit, opts); err != nil {
			return plumbing.ZeroHash, fmt.Errorf("applying edit to %s: %w", edit.FilePath, err)
		}
	}
	
	// Stage changes
	for _, edit := range edits {
		if _, err := wt.Add(edit.FilePath); err != nil {
			return plumbing.ZeroHash, fmt.Errorf("staging %s: %w", edit.FilePath, err)
		}
	}
	
	// Commit
	commitMsg := opts.CommitMessage
	if commitMsg == "" {
		commitMsg = fmt.Sprintf("AI update: %s", request)
	}
	
	commitOpts := &CommitOptions{}
	if opts.Author != nil {
		commitOpts.Author = opts.Author
	}
	
	hash, err := wt.Commit(commitMsg, commitOpts)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("committing changes: %w", err)
	}
	
	return hash, nil
}

// readAllFiles reads all files from HEAD tree.
func (r *Repository) readAllFiles(ctx context.Context) ([]FileContent, error) {
	head, err := r.Head()
	if err != nil {
		return nil, err
	}
	
	commit, err := r.CommitObject(head.Hash())
	if err != nil {
		return nil, err
	}
	
	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}
	
	var files []FileContent
	err = tree.Files().ForEach(func(f *object.File) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		content, err := f.Contents()
		if err != nil {
			return err
		}
		
		files = append(files, FileContent{
			Path:    f.Name,
			Content: content,
			Hash:    f.Hash,
		})
		return nil
	})
	
	return files, err
}

// applyEdit writes a file edit to the worktree filesystem.
func applyEdit(wt *Worktree, edit FileEdit, opts *AIUpdateOptions) error {
	if edit.OldContent == "" && !opts.AllowNewFiles {
		return fmt.Errorf("new file creation not allowed: %s", edit.FilePath)
	}
	
	if edit.NewContent == "" && !opts.AllowDeleteFiles {
		return fmt.Errorf("file deletion not allowed: %s", edit.FilePath)
	}
	
	if edit.NewContent == "" {
		// Delete file
		return wt.Filesystem.Remove(edit.FilePath)
	}
	
	// Write new/updated file
	f, err := wt.Filesystem.Create(edit.FilePath)
	if err != nil {
		return err
	}
	defer f.Close()
	
	_, err = io.WriteString(f, edit.NewContent)
	return err
}
```

### 3. Example AI Provider implementation (OpenAI)

```go
// ai_provider_openai.go
package git

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OpenAIProvider implements AIProvider using OpenAI API.
type OpenAIProvider struct {
	APIKey     string
	Model      string // e.g., "gpt-4", "gpt-3.5-turbo"
	HTTPClient *http.Client
}

// NewOpenAIProvider creates an OpenAI-backed AI provider.
func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		APIKey:     apiKey,
		Model:      "gpt-4",
		HTTPClient: &http.Client{},
	}
}

func (p *OpenAIProvider) SemanticSearch(ctx context.Context, query string, files []FileContent) ([]SearchResult, error) {
	// Build context for AI
	var filesContext strings.Builder
	for _, f := range files {
		filesContext.WriteString(fmt.Sprintf("File: %s\n%s\n\n", f.Path, f.Content))
	}
	
	prompt := fmt.Sprintf(`Search the following repository files for content matching this query: "%s"

Repository files:
%s

Return a JSON array of matches with this structure:
[{"file_path": "path/to/file", "snippet": "relevant code snippet", "score": 0.95, "line_start": 10, "line_end": 20}]

Return up to 10 most relevant matches, ranked by relevance.`, query, filesContext.String())
	
	response, err := p.callOpenAI(ctx, prompt)
	if err != nil {
		return nil, err
	}
	
	var results []SearchResult
	if err := json.Unmarshal([]byte(response), &results); err != nil {
		return nil, fmt.Errorf("parsing AI response: %w", err)
	}
	
	return results, nil
}

func (p *OpenAIProvider) CodeSearch(ctx context.Context, query string, files []FileContent) ([]CodeMatch, error) {
	// Similar to SemanticSearch but focused on code patterns
	var filesContext strings.Builder
	for _, f := range files {
		filesContext.WriteString(fmt.Sprintf("File: %s\n%s\n\n", f.Path, f.Content))
	}
	
	prompt := fmt.Sprintf(`Find code matching this pattern: "%s"

Repository files:
%s

Return a JSON array of code matches with this structure:
[{"file_path": "path/to/file", "function_name": "funcName", "line_start": 10, "line_end": 20, "code_snippet": "...", "description": "explanation"}]

Focus on functions, classes, and meaningful code blocks.`, query, filesContext.String())
	
	response, err := p.callOpenAI(ctx, prompt)
	if err != nil {
		return nil, err
	}
	
	var matches []CodeMatch
	if err := json.Unmarshal([]byte(response), &matches); err != nil {
		return nil, fmt.Errorf("parsing AI response: %w", err)
	}
	
	return matches, nil
}

func (p *OpenAIProvider) GenerateCodeUpdate(ctx context.Context, request string, files []FileContent) ([]FileEdit, error) {
	var filesContext strings.Builder
	for _, f := range files {
		filesContext.WriteString(fmt.Sprintf("File: %s\n%s\n\n", f.Path, f.Content))
	}
	
	prompt := fmt.Sprintf(`Generate code changes for this request: "%s"

Current repository files:
%s

Return a JSON array of file edits with this structure:
[{"file_path": "path/to/file", "old_content": "...", "new_content": "...", "description": "explanation of change"}]

Include the full new content for each file. For new files, old_content should be empty.`, request, filesContext.String())
	
	response, err := p.callOpenAI(ctx, prompt)
	if err != nil {
		return nil, err
	}
	
	var edits []FileEdit
	if err := json.Unmarshal([]byte(response), &edits); err != nil {
		return nil, fmt.Errorf("parsing AI response: %w", err)
	}
	
	return edits, nil
}

func (p *OpenAIProvider) callOpenAI(ctx context.Context, prompt string) (string, error) {
	// Simplified OpenAI API call
	// In production, use a proper OpenAI SDK
	reqBody := map[string]interface{}{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
	}
	
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}
	
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI API error: %s", body)
	}
	
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}
	
	return result.Choices[0].Message.Content, nil
}
```

## Example usage

### Example 1: Semantic search

```go
package main

import (
	"context"
	"fmt"
	"log"
	
	"github.com/go-git/go-git/v6"
)

func main() {
	// Open repository
	repo, err := git.PlainOpen("/path/to/repo")
	if err != nil {
		log.Fatal(err)
	}
	
	// Configure AI provider
	aiProvider := git.NewOpenAIProvider("your-openai-api-key")
	repo.SetAIProvider(aiProvider)
	
	// Search for code
	ctx := context.Background()
	results, err := repo.AISemanticSearch(ctx, "authentication and JWT token validation")
	if err != nil {
		log.Fatal(err)
	}
	
	// Display results
	for _, result := range results {
		fmt.Printf("Found in %s (score: %.2f):\n%s\n\n", 
			result.FilePath, result.Score, result.Snippet)
	}
}
```

### Example 2: Code search

```go
func main() {
	repo, _ := git.PlainOpen("/path/to/repo")
	aiProvider := git.NewOpenAIProvider("your-api-key")
	repo.SetAIProvider(aiProvider)
	
	ctx := context.Background()
	matches, err := repo.AICodeSearch(ctx, "functions that handle HTTP requests")
	if err != nil {
		log.Fatal(err)
	}
	
	for _, match := range matches {
		fmt.Printf("Function: %s in %s (lines %d-%d)\n", 
			match.FunctionName, match.FilePath, match.LineStart, match.LineEnd)
		fmt.Printf("Description: %s\n", match.Description)
		fmt.Printf("Code:\n%s\n\n", match.CodeSnippet)
	}
}
```

### Example 3: AI-driven code update

```go
func main() {
	repo, _ := git.PlainOpen("/path/to/repo")
	aiProvider := git.NewOpenAIProvider("your-api-key")
	repo.SetAIProvider(aiProvider)
	
	ctx := context.Background()
	
	// Request code changes
	request := "Add error handling to all database queries and log errors"
	
	opts := &git.AIUpdateOptions{
		CommitMessage: "Add error handling to database queries",
		Author: &object.Signature{
			Name:  "AI Assistant",
			Email: "ai@example.com",
			When:  time.Now(),
		},
		AllowNewFiles: false,
		DryRun: false,
	}
	
	hash, err := repo.AIUpdateCode(ctx, request, opts)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Changes committed: %s\n", hash)
}
```

## Implementation plan

### Phase 1: Core infrastructure (ai_provider.go, ai_search.go)
- [ ] Create `ai_provider.go` with AIProvider interface and types
- [ ] Add `aiProvider` field to Repository struct
- [ ] Implement `SetAIProvider()` method
- [ ] Implement `readAllFiles()` helper

### Phase 2: Search functions
- [ ] Implement `AISemanticSearch()`
- [ ] Implement `AICodeSearch()`
- [ ] Add tests with mock AI provider

### Phase 3: Update function
- [ ] Implement `AIUpdateCode()`
- [ ] Implement `applyEdit()` helper
- [ ] Add staging and commit flow
- [ ] Add tests for dry-run and actual updates

### Phase 4: OpenAI provider
- [ ] Create `ai_provider_openai.go`
- [ ] Implement OpenAIProvider struct
- [ ] Add proper API error handling and rate limiting
- [ ] Add tests with mocked HTTP responses

### Phase 5: Documentation and examples
- [ ] Add examples to `_examples/ai-search/`
- [ ] Add examples to `_examples/ai-update/`
- [ ] Update READMEDEV.md with AI integration section
- [ ] Add API docs

## File structure

```
go-git/
├── ai_provider.go          # AIProvider interface and types
├── ai_search.go            # AI search/update methods on Repository
├── ai_provider_openai.go   # OpenAI implementation
├── ai_provider_test.go     # Tests with mock provider
├── _examples/
│   ├── ai-search/
│   │   └── main.go
│   └── ai-update/
│       └── main.go
└── READMEDEV.md            # Updated with AI section
```

## Security considerations

1. **API keys** — never commit API keys; use environment variables
2. **Rate limiting** — implement backoff and retry for AI API calls
3. **Input validation** — sanitize file content before sending to AI
4. **Output validation** — verify AI-generated code before applying
5. **Permissions** — respect AllowNewFiles/AllowDeleteFiles flags
6. **Audit trail** — all AI changes go through proper git commit flow

## Alternative AI providers

Users can implement their own providers:

```go
type LocalLLMProvider struct {
	ModelPath string
}

func (p *LocalLLMProvider) SemanticSearch(ctx context.Context, query string, files []FileContent) ([]SearchResult, error) {
	// Call local model (e.g., llama.cpp, ollama)
	// ...
}

// Similar for CodeSearch and GenerateCodeUpdate
```

## Vector Database Options for Semantic Search

### The question: Do we need a vector database?

**Short answer: No, it's not necessary to embed a vector database into go-git itself.** There are several architectural approaches, each with different tradeoffs:

### Approach 1: External Vector Database (Recommended for production)

**Architecture:**
```
go-git Repository
    ↓ (reads files)
AIProvider (client)
    ↓ (sends embeddings request)
External Vector DB Service
    - Pinecone
    - Weaviate
    - Qdrant
    - Milvus
    - Chroma
```

**Pros:**
- ✅ Scalable to large repositories (millions of files)
- ✅ Fast similarity search (optimized indexes)
- ✅ Persistent embeddings cache (don't re-embed on every search)
- ✅ Go-git stays lightweight
- ✅ Users can choose their own vector DB

**Cons:**
- ❌ Requires external service/setup
- ❌ Network latency
- ❌ Additional infrastructure cost

**Example implementation:**

```go
// ai_provider_vectordb.go
package git

import (
	"context"
	"fmt"
	
	"github.com/pinecone-io/go-pinecone/pinecone" // example
)

type VectorDBProvider struct {
	VectorDB      *pinecone.Client
	EmbeddingAPI  string // OpenAI embeddings API
	IndexName     string
	apiKey        string
}

func NewVectorDBProvider(vectorDBClient *pinecone.Client, embeddingAPIKey string) *VectorDBProvider {
	return &VectorDBProvider{
		VectorDB:     vectorDBClient,
		EmbeddingAPI: "https://api.openai.com/v1/embeddings",
		apiKey:       embeddingAPIKey,
	}
}

func (p *VectorDBProvider) SemanticSearch(ctx context.Context, query string, files []FileContent) ([]SearchResult, error) {
	// 1. Generate embedding for query
	queryEmbedding, err := p.generateEmbedding(ctx, query)
	if err != nil {
		return nil, err
	}
	
	// 2. Search vector DB for similar embeddings
	results, err := p.VectorDB.Query(ctx, &pinecone.QueryRequest{
		Vector:          queryEmbedding,
		TopK:            10,
		IncludeMetadata: true,
	})
	if err != nil {
		return nil, err
	}
	
	// 3. Map vector DB results to SearchResult
	var searchResults []SearchResult
	for _, match := range results.Matches {
		searchResults = append(searchResults, SearchResult{
			FilePath:  match.Metadata["file_path"].(string),
			Snippet:   match.Metadata["snippet"].(string),
			Score:     float64(match.Score),
			LineStart: int(match.Metadata["line_start"].(float64)),
			LineEnd:   int(match.Metadata["line_end"].(float64)),
		})
	}
	
	return searchResults, nil
}

func (p *VectorDBProvider) generateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Call OpenAI embeddings API (or any embedding model)
	// Returns a vector like [0.123, -0.456, 0.789, ...]
	// Implementation omitted for brevity
	return nil, nil
}

// IndexRepository pre-computes embeddings for all files and stores in vector DB
func (p *VectorDBProvider) IndexRepository(ctx context.Context, files []FileContent) error {
	for _, file := range files {
		// Split file into chunks (e.g., 500 tokens each)
		chunks := splitIntoChunks(file.Content, 500)
		
		for i, chunk := range chunks {
			embedding, err := p.generateEmbedding(ctx, chunk)
			if err != nil {
				return err
			}
			
			// Upsert to vector DB
			_, err = p.VectorDB.Upsert(ctx, &pinecone.UpsertRequest{
				Vectors: []*pinecone.Vector{
					{
						ID:     fmt.Sprintf("%s_chunk_%d", file.Path, i),
						Values: embedding,
						Metadata: map[string]interface{}{
							"file_path":  file.Path,
							"snippet":    chunk,
							"line_start": i * 10, // approximate
							"line_end":   (i + 1) * 10,
						},
					},
				},
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}
```

### Approach 2: Embedding-only (No vector DB)

**Architecture:**
```
go-git Repository
    ↓ (reads files)
AIProvider
    ↓ (generates embeddings for query + all files)
In-memory similarity calculation
    ↓ (cosine similarity)
Return top-K results
```

**Pros:**
- ✅ No external dependencies
- ✅ Simple to implement
- ✅ Works for small-to-medium repos (<1000 files)
- ✅ No infrastructure needed

**Cons:**
- ❌ Slow for large repositories (must embed all files on every search)
- ❌ Expensive (API calls for embeddings on every search)
- ❌ No embedding cache

**Example implementation:**

```go
// ai_provider_embedding.go
package git

import (
	"context"
	"math"
	"sort"
)

type EmbeddingOnlyProvider struct {
	EmbeddingAPI string
	apiKey       string
}

func (p *EmbeddingOnlyProvider) SemanticSearch(ctx context.Context, query string, files []FileContent) ([]SearchResult, error) {
	// 1. Generate embedding for query
	queryEmbed, err := p.generateEmbedding(ctx, query)
	if err != nil {
		return nil, err
	}
	
	// 2. Generate embeddings for all files (expensive!)
	type fileWithEmbedding struct {
		file      FileContent
		embedding []float32
	}
	var fileEmbeddings []fileWithEmbedding
	
	for _, file := range files {
		embed, err := p.generateEmbedding(ctx, file.Content[:min(8000, len(file.Content))]) // truncate to fit token limit
		if err != nil {
			return nil, err
		}
		fileEmbeddings = append(fileEmbeddings, fileWithEmbedding{file, embed})
	}
	
	// 3. Calculate cosine similarity
	type scoredResult struct {
		file  FileContent
		score float64
	}
	var scored []scoredResult
	
	for _, fe := range fileEmbeddings {
		similarity := cosineSimilarity(queryEmbed, fe.embedding)
		scored = append(scored, scoredResult{fe.file, similarity})
	}
	
	// 4. Sort by score and return top-K
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})
	
	var results []SearchResult
	for i := 0; i < min(10, len(scored)); i++ {
		results = append(results, SearchResult{
			FilePath: scored[i].file.Path,
			Snippet:  scored[i].file.Content[:min(200, len(scored[i].file.Content))],
			Score:    scored[i].score,
		})
	}
	
	return results, nil
}

func cosineSimilarity(a, b []float32) float64 {
	var dotProduct, magA, magB float64
	for i := range a {
		dotProduct += float64(a[i] * b[i])
		magA += float64(a[i] * a[i])
		magB += float64(b[i] * b[i])
	}
	return dotProduct / (math.Sqrt(magA) * math.Sqrt(magB))
}
```

### Approach 3: Hybrid (LLM-based without embeddings)

**Architecture:**
```
go-git Repository
    ↓ (reads files)
AIProvider
    ↓ (sends all files + query to LLM in single prompt)
LLM analyzes and returns matches
```

**Pros:**
- ✅ No vector DB needed
- ✅ No embeddings needed
- ✅ LLM can understand context and code semantics better
- ✅ Works well for small repos

**Cons:**
- ❌ Limited by LLM context window (e.g., 128K tokens for GPT-4)
- ❌ Expensive (entire repo sent on every search)
- ❌ Slow (LLM inference time)
- ❌ Not suitable for large repos

**This is what the current design document shows (already implemented above).**

### Approach 4: Existing frameworks that provide semantic search

Instead of implementing from scratch, use existing Go libraries:

#### Option A: **LangChain Go** ✅ (Mature and actively maintained)
- **Status**: Production-ready with 7.8k stars, 185 contributors, actively maintained
- **Version**: v0.1.13 (stable releases)
- **Provides everything you need**:
  - ✅ **Vector stores** with 10+ implementations (Pinecone, Qdrant, Chroma, Weaviate, Milvus, PostgreSQL/pgvector, Redis, MongoDB, OpenSearch, Azure AI Search)
  - ✅ **Embeddings** with multiple providers (OpenAI, Hugging Face, Jina, VoyageAI, Bedrock, local models via Cybertron)
  - ✅ **Document loaders** (including RecursiveDirectoryLoader for code repositories)
  - ✅ **Text splitters** (for chunking code/documents)
  - ✅ **Retriever interface** for semantic search
  - ✅ **Chains and agents** for complex LLM workflows
- **GitHub**: https://github.com/tmc/langchaingo
- **Docs**: https://tmc.github.io/langchaingo/docs/
- **API**: https://pkg.go.dev/github.com/tmc/langchaingo

#### Option B: **Semantic search libraries:**

1. **Qdrant Go client** (vector database with Go SDK)
   ```go
   import "github.com/qdrant/go-client/qdrant"
   ```

2. **Weaviate Go client** (vector database)
   ```go
   import "github.com/weaviate/weaviate-go-client/v4/weaviate"
   ```

3. **txtai** (via HTTP API from Go)
   - Python-based semantic search engine
   - Can run as a service, call from Go

4. **Milvus Go SDK** (distributed vector database)
   ```go
   import "github.com/milvus-io/milvus-sdk-go/v2/client"
   ```

#### Option C: **Code-specific search engines:**

1. **GitHub Code Search** (via GitHub API)
   - Already handles semantic understanding of code
   - Rate limited, requires GitHub token

2. **Sourcegraph** (via GraphQL API)
   - Powerful code search with AI features
   - Can self-host or use cloud version

### Recommendation: Pluggable architecture

**Best approach for go-git:**

1. **Keep go-git lightweight** — don't embed a vector database
2. **Use pluggable AIProvider interface** (as designed above)
3. **Provide multiple provider implementations:**
   - `OpenAIProvider` — simple LLM-based (for small repos)
   - `VectorDBProvider` — with external Pinecone/Qdrant/Weaviate (for large repos)
   - `EmbeddingCacheProvider` — generates embeddings once, caches in SQLite/file
4. **Let users choose** based on their needs

### Updated implementation example with vector DB option

```go
// Usage example: Small repo (no vector DB)
repo, _ := git.PlainOpen("/path/to/small-repo")
aiProvider := git.NewOpenAIProvider("api-key")
repo.SetAIProvider(aiProvider)
results, _ := repo.AISemanticSearch(ctx, "authentication logic")

// Usage example: Large repo with vector DB
repo, _ := git.PlainOpen("/path/to/large-repo")
vectorDB := pinecone.NewClient("api-key")
aiProvider := git.NewVectorDBProvider(vectorDB, "embedding-api-key")

// Index repository once (can be done offline)
files, _ := repo.readAllFiles(ctx)
aiProvider.IndexRepository(ctx, files)

// Fast searches using vector DB
repo.SetAIProvider(aiProvider)
results, _ := repo.AISemanticSearch(ctx, "authentication logic")
```

### Summary table

| Approach | Setup Complexity | Search Speed | Cost | Best For |
|----------|-----------------|--------------|------|----------|
| LLM-only (current design) | Low | Slow | High | Small repos (<100 files) |
| Embedding-only | Low | Medium | Medium | Medium repos (<1000 files) |
| External Vector DB | High | Fast | Low (after indexing) | Large repos (1000+ files) |
| Hybrid (cached embeddings) | Medium | Fast | Low | Any size, offline capable |
| **LangChain Go** | **Low-Medium** | **Fast** | **Low** | **Any size, recommended** |

## Using LangChain Go with go-git (Recommended Approach)

### Why LangChain Go is perfect for this use case

**LangChain Go provides everything we need out of the box:**

1. ✅ **Vector store abstraction** — one interface, 10+ backends
2. ✅ **Automatic embeddings** — handles chunking, batching, and embedding generation
3. ✅ **Semantic search API** — `SimilaritySearch()` with score threshold
4. ✅ **Document loaders** — `RecursiveDirectoryLoader` for code repositories
5. ✅ **Retriever pattern** — clean API for search workflows
6. ✅ **Production-ready** — 7.8k stars, used by 1.5k projects

### Complete implementation using LangChain Go

Here's how to integrate LangChain Go with go-git:

```go
// ai_provider_langchain.go
package git

import (
	"context"
	"fmt"
	"path/filepath"
	
	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/chroma"    // or pinecone, qdrant, etc.
)

// LangChainProvider implements AIProvider using LangChain Go.
type LangChainProvider struct {
	vectorStore vectorstores.VectorStore
	embedder    embeddings.Embedder
	llm         *openai.LLM
}

// NewLangChainProvider creates a provider using LangChain Go with Chroma vector store.
func NewLangChainProvider(apiKey string, chromaURL string) (*LangChainProvider, error) {
	// Create LLM client
	llm, err := openai.New(openai.WithToken(apiKey))
	if err != nil {
		return nil, err
	}
	
	// Create embedder
	embedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		return nil, err
	}
	
	// Create vector store (Chroma in this example)
	store, err := chroma.New(
		chroma.WithChromaURL(chromaURL),
		chroma.WithEmbedder(embedder),
	)
	if err != nil {
		return nil, err
	}
	
	return &LangChainProvider{
		vectorStore: store,
		embedder:    embedder,
		llm:         llm,
	}, nil
}

// IndexRepository indexes all files from a repository into the vector store.
func (p *LangChainProvider) IndexRepository(ctx context.Context, repoPath string) error {
	// Load all files from repository
	loader := documentloaders.NewRecursiveDirectory(repoPath)
	docs, err := loader.Load(ctx)
	if err != nil {
		return fmt.Errorf("loading documents: %w", err)
	}
	
	// Split documents into chunks (500 characters with 50 overlap)
	splitter := textsplitter.NewRecursiveCharacter()
	splitter.ChunkSize = 500
	splitter.ChunkOverlap = 50
	
	splitDocs, err := textsplitter.SplitDocuments(splitter, docs)
	if err != nil {
		return fmt.Errorf("splitting documents: %w", err)
	}
	
	// Add metadata (file paths) to each chunk
	for i := range splitDocs {
		if splitDocs[i].Metadata == nil {
			splitDocs[i].Metadata = make(map[string]any)
		}
		// Extract file path from document source
		splitDocs[i].Metadata["file_path"] = splitDocs[i].Metadata["source"]
	}
	
	// Add to vector store (automatically generates embeddings)
	_, err = p.vectorStore.AddDocuments(ctx, splitDocs)
	if err != nil {
		return fmt.Errorf("adding documents to vector store: %w", err)
	}
	
	return nil
}

func (p *LangChainProvider) SemanticSearch(ctx context.Context, query string, files []FileContent) ([]SearchResult, error) {
	// Note: files parameter is ignored since we're using the indexed vector store
	
	// Perform similarity search
	docs, err := p.vectorStore.SimilaritySearch(ctx, query, 10,
		vectorstores.WithScoreThreshold(0.7), // Only return results with >70% similarity
	)
	if err != nil {
		return nil, err
	}
	
	// Convert to SearchResult
	var results []SearchResult
	for _, doc := range docs {
		filePath := ""
		if path, ok := doc.Metadata["file_path"].(string); ok {
			filePath = path
		}
		
		lineStart := 0
		if line, ok := doc.Metadata["line_start"].(int); ok {
			lineStart = line
		}
		
		results = append(results, SearchResult{
			FilePath:  filePath,
			Snippet:   doc.PageContent,
			Score:     doc.Score,
			LineStart: lineStart,
			LineEnd:   lineStart + countLines(doc.PageContent),
		})
	}
	
	return results, nil
}

func (p *LangChainProvider) CodeSearch(ctx context.Context, query string, files []FileContent) ([]CodeMatch, error) {
	// Use semantic search with code-specific query augmentation
	enhancedQuery := fmt.Sprintf("Code that implements: %s", query)
	
	docs, err := p.vectorStore.SimilaritySearch(ctx, enhancedQuery, 10)
	if err != nil {
		return nil, err
	}
	
	// Convert to CodeMatch with AI-generated descriptions
	var matches []CodeMatch
	for _, doc := range docs {
		filePath := ""
		if path, ok := doc.Metadata["file_path"].(string); ok {
			filePath = path
		}
		
		// Extract function name if available
		functionName := extractFunctionName(doc.PageContent)
		
		matches = append(matches, CodeMatch{
			FilePath:     filePath,
			FunctionName: functionName,
			CodeSnippet:  doc.PageContent,
			Score:        doc.Score,
			Description:  fmt.Sprintf("Matches query: %s", query),
		})
	}
	
	return matches, nil
}

func (p *LangChainProvider) GenerateCodeUpdate(ctx context.Context, request string, files []FileContent) ([]FileEdit, error) {
	// Use LLM to generate code updates
	// First, find relevant code using semantic search
	relevantDocs, err := p.vectorStore.SimilaritySearch(ctx, request, 5)
	if err != nil {
		return nil, err
	}
	
	// Build context from relevant code
	var contextBuilder string
	for _, doc := range relevantDocs {
		filePath := doc.Metadata["file_path"].(string)
		contextBuilder += fmt.Sprintf("File: %s\n%s\n\n", filePath, doc.PageContent)
	}
	
	// Generate code changes using LLM
	prompt := fmt.Sprintf(`Given this request: "%s"

Relevant code:
%s

Generate code changes in JSON format:
[{"file_path": "...", "old_content": "...", "new_content": "...", "description": "..."}]`, 
		request, contextBuilder)
	
	completion, err := p.llm.Call(ctx, prompt)
	if err != nil {
		return nil, err
	}
	
	// Parse LLM response into FileEdit objects
	// (Implementation would parse JSON from completion)
	var edits []FileEdit
	// ... parsing logic ...
	
	return edits, nil
}

func countLines(text string) int {
	return len(strings.Split(text, "\n"))
}

func extractFunctionName(code string) string {
	// Simple regex to extract function name
	// More sophisticated parsing could use go/parser
	re := regexp.MustCompile(`func\s+(\w+)`)
	matches := re.FindStringSubmatch(code)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
```

### Usage example with LangChain Go

```go
package main

import (
	"context"
	"fmt"
	"log"
	
	"github.com/go-git/go-git/v6"
)

func main() {
	ctx := context.Background()
	
	// Open repository
	repo, err := git.PlainOpen("/path/to/repo")
	if err != nil {
		log.Fatal(err)
	}
	
	// Create LangChain provider with Chroma vector store
	provider, err := git.NewLangChainProvider(
		"your-openai-api-key",
		"http://localhost:8000", // Chroma server URL
	)
	if err != nil {
		log.Fatal(err)
	}
	
	// Index the repository ONCE (can be done offline)
	fmt.Println("Indexing repository...")
	err = provider.IndexRepository(ctx, "/path/to/repo")
	if err != nil {
		log.Fatal(err)
	}
	
	// Set provider
	repo.SetAIProvider(provider)
	
	// Now you can do fast semantic searches!
	results, err := repo.AISemanticSearch(ctx, "authentication logic with JWT")
	if err != nil {
		log.Fatal(err)
	}
	
	for _, result := range results {
		fmt.Printf("Found in %s (score: %.2f):\n%s\n\n", 
			result.FilePath, result.Score, result.Snippet)
	}
}
```

### Vector store options with LangChain Go

LangChain Go supports 10+ vector stores out of the box:

| Vector Store | Best For | Setup |
|--------------|----------|-------|
| **Chroma** | Local development, prototyping | Run Docker: `docker run -p 8000:8000 chromadb/chroma` |
| **Qdrant** | Production, on-premise | Cloud or self-hosted |
| **Pinecone** | Managed cloud, scale | Sign up at pinecone.io |
| **Weaviate** | Rich schema, hybrid search | Cloud or self-hosted |
| **PostgreSQL/pgvector** | Existing Postgres DB | Install pgvector extension |
| **Redis** | High-speed, existing Redis | Redis Stack or RedisJSON |
| **Milvus** | Massive scale (billions of vectors) | Cloud or Kubernetes |
| **MongoDB** | Existing MongoDB | MongoDB Atlas or self-hosted |
| **OpenSearch** | AWS ecosystem | AWS OpenSearch Service |
| **Azure AI Search** | Azure ecosystem | Azure Cognitive Search |

### Switching vector stores is trivial

```go
// Use Chroma (local)
store, _ := chroma.New(
	chroma.WithChromaURL("http://localhost:8000"),
	chroma.WithEmbedder(embedder),
)

// OR use Qdrant (production)
store, _ := qdrant.New(
	qdrant.WithURL("http://qdrant:6333"),
	qdrant.WithEmbedder(embedder),
)

// OR use Pinecone (managed cloud)
store, _ := pinecone.New(
	pinecone.WithAPIKey("your-api-key"),
	pinecone.WithEnvironment("us-east-1-aws"),
	pinecone.WithIndexName("go-git-code"),
	pinecone.WithEmbedder(embedder),
)
```

### Benefits of using LangChain Go

1. **No need to implement vector storage yourself** — 10+ battle-tested implementations
2. **Automatic embedding generation** — handles batching, chunking, and API calls
3. **Clean abstractions** — VectorStore and Embedder interfaces are well-designed
4. **Production-ready** — used by 1.5k projects, actively maintained
5. **Easy to switch backends** — change one line to switch from Chroma to Pinecone
6. **Document loading built-in** — `RecursiveDirectoryLoader` for code repositories
7. **Text splitting** — smart chunking algorithms for code
8. **Score thresholds** — built-in filtering for relevance

### My recommendation for go-git

**✅ Use LangChain Go as the foundation** — it provides everything we need:

1. **Default provider: LangChainProvider** (recommended)
   - Uses LangChain Go's abstractions
   - User chooses their vector store (Chroma for dev, Pinecone/Qdrant for prod)
   - Handles embeddings, chunking, and indexing automatically
   - One-time repository indexing, fast searches thereafter

2. **Fallback provider: SimpleLLMProvider** (for small repos)
   - Direct LLM calls without vector DB
   - Zero setup, just API key
   - Works for repos with <50 files

**Recommended architecture:**

```
go-git/
├── ai_provider.go                      # AIProvider interface (keep as-is)
├── ai_provider_langchain.go            # LangChain Go provider (RECOMMENDED)
├── ai_provider_simple.go               # Simple LLM fallback
├── ai_search.go                        # Repository AI methods (keep as-is)
└── _examples/
    ├── ai-search-langchain/            # LangChain + Chroma (main example)
    ├── ai-search-langchain-pinecone/   # Production example with Pinecone
    └── ai-search-simple/               # Simple fallback for tiny repos
```

### Why LangChain Go is the right choice

| Criteria | LangChain Go | Custom Implementation |
|----------|--------------|----------------------|
| **Vector store support** | ✅ 10+ out of the box | ❌ Need to implement each one |
| **Embeddings** | ✅ Auto-handled with batching | ❌ Need to implement batching logic |
| **Document loading** | ✅ Built-in loaders | ❌ Need to write file readers |
| **Text splitting** | ✅ Smart chunking | ❌ Need to implement chunking |
| **Production-ready** | ✅ 7.8k stars, battle-tested | ❌ Untested |
| **Maintenance** | ✅ Community-maintained | ❌ You maintain |
| **Flexibility** | ✅ Easy to switch backends | ❌ Locked into your impl |

### Answer to your question

> "Does LangChain Go provide vector DB for auto storage and embedding the code in go-git, and provide API for LLM semantic search?"

**YES! LangChain Go provides:**

✅ **Vector DB storage** — 10+ implementations (Chroma, Qdrant, Pinecone, Weaviate, etc.)
✅ **Automatic embeddings** — via `embeddings.Embedder` interface with OpenAI, Hugging Face, etc.
✅ **Auto code storage** — `RecursiveDirectoryLoader` loads and indexes entire repositories
✅ **Semantic search API** — `vectorstores.VectorStore.SimilaritySearch()` with score thresholds
✅ **LLM integration** — Built-in LLM clients (OpenAI, Anthropic, local models)
✅ **Retriever pattern** — High-level API for semantic search workflows

**You don't need to implement any of this yourself** — LangChain Go has it all!

## Implementation recommendation

I recommend implementing the **LangChainProvider** first:

**Pros:**
- ✅ Uses industry-standard library (like Python's LangChain)
- ✅ 90% less code to maintain
- ✅ Users can choose their vector DB
- ✅ Production-ready from day one
- ✅ Easy to test with local Chroma
- ✅ Scales to production with Pinecone/Qdrant

**Implementation effort:**
- ~200 lines of code (vs ~1000+ for custom)
- 2-3 hours of work
- Minimal maintenance burden

## Next steps

Would you like me to:
1. ✅ **Implement LangChainProvider** (recommended) — complete working implementation with LangChain Go
2. **Create example with Chroma** — local Docker setup for testing
3. **Create example with Pinecone** — production-ready example
4. **Add SimpleLLMProvider** — fallback for tiny repos without vector DB

Let me know and I'll implement it!
