package ai

import (
    "bufio"
    "context"
    "crypto/sha1"
    "encoding/hex"
    "fmt"
    "io"
    "path/filepath"
    "strings"
    "time"

    gogit "github.com/go-git/go-git/v6"
    "github.com/go-git/go-git/v6/plumbing/object"
)

// HybridProvider wires TEI + Chroma for embeddings and retrieval; LLM adapters are added later.
type HybridProvider struct {
    cfg          Config
    tei          *TEIClient
    chroma       *ChromaClient
    collectionID string
}

func NewHybridProvider(cfg Config) *HybridProvider {
    hp := &HybridProvider{
        cfg:    cfg,
        tei:    NewTEIClient(cfg.TEIEndpoint),
        chroma: NewChromaClient(cfg.ChromaURL),
    }
    return hp
}

func (h *HybridProvider) ensureCollection(ctx context.Context, name string) error {
    if h.collectionID != "" { return nil }
    id, err := h.chroma.EnsureCollection(ctx, name)
    if err != nil { return err }
    h.collectionID = id
    return nil
}

// IndexRepository performs a simple full-scan index of HEAD tree.
// It chunks files into ~1000-character blocks with line boundaries.
func (h *HybridProvider) IndexRepository(ctx context.Context, repoPath string) error {
    if err := h.ensureCollection(ctx, safeCollectionName(repoPath)); err != nil { return err }
    repo, err := gogit.PlainOpen(repoPath)
    if err != nil { return err }
    headRef, err := repo.Head()
    if err != nil { return err }
    commit, err := repo.CommitObject(headRef.Hash())
    if err != nil { return err }
    tree, err := commit.Tree()
    if err != nil { return err }

    // Walk files
    var upserts []UpsertItem
    err = tree.Files().ForEach(func(f *object.File) error {
        // Skip vendored and binary-ish files quickly
        if skipPath(f.Name) { return nil }
        r, err := f.Reader()
        if err != nil { return nil }
        defer r.Close()
        // Naive text read with size cap
        const maxFileBytes = 2 * 1024 * 1024
        lr := io.LimitReader(r, maxFileBytes)
        up, err := h.buildChunksForFile(ctx, repoPath, commit.Hash.String(), f, lr)
        if err == nil && len(up) > 0 {
            upserts = append(upserts, up...)
            // Batch flush to keep memory bounded
            if len(upserts) >= 128 {
                if err := h.chroma.Upsert(ctx, h.collectionID, upserts); err != nil { return err }
                upserts = upserts[:0]
            }
        }
        return nil
    })
    if err != nil { return err }
    if len(upserts) > 0 {
        if err := h.chroma.Upsert(ctx, h.collectionID, upserts); err != nil { return err }
    }
    return nil
}

func (h *HybridProvider) buildChunksForFile(ctx context.Context, repoPath, commitHash string, f *object.File, r io.Reader) ([]UpsertItem, error) {
    // Read lines and group into chunks ~1000 chars
    scanner := bufio.NewScanner(r)
    // Increase buffer for long lines
    buf := make([]byte, 0, 256*1024)
    scanner.Buffer(buf, 1024*1024)
    var (
        chunks   []string
        starts   []int
        cur      strings.Builder
        startLn  = 1
        curLn    = 1
    )
    for scanner.Scan() {
        line := scanner.Text()
        if cur.Len() == 0 { startLn = curLn }
        cur.WriteString(line)
        cur.WriteString("\n")
        if cur.Len() >= 1000 {
            chunks = append(chunks, cur.String())
            starts = append(starts, startLn)
            cur.Reset()
        }
        curLn++
    }
    if cur.Len() > 0 {
        chunks = append(chunks, cur.String())
        starts = append(starts, startLn)
    }
    if err := scanner.Err(); err != nil { return nil, err }
    if len(chunks) == 0 { return nil, nil }

    // Embed chunks with TEI
    embs, err := h.tei.Embed(ctx, chunks)
    if err != nil { return nil, err }

    // Prepare upserts
    items := make([]UpsertItem, 0, len(chunks))
    for i, ch := range chunks {
        id := chunkID(commitHash, f.Name, starts[i], starts[i]+strings.Count(ch, "\n"))
        items = append(items, UpsertItem{
            ID:        id,
            Embedding: embs[i],
            Document:  ch,
            Metadata: map[string]any{
                "repo_path":   repoPath,
                "commit_hash": commitHash,
                "file_path":   f.Name,
                "start_line":  starts[i],
                "end_line":    starts[i] + strings.Count(ch, "\n"),
                "blob_hash":   f.Hash.String(),
                "language":    languageFromPath(f.Name),
                "created_at":  time.Now().UTC().Format(time.RFC3339Nano),
            },
        })
    }
    return items, nil
}

// SemanticSearch embeds the query and retrieves topK results from Chroma, optionally reranking with TEI.
func (h *HybridProvider) SemanticSearch(ctx context.Context, repoPath string, query string, topK int, rerank bool) ([]SearchResult, error) {
    results := make([]SearchResult, 0, topK)
    err := h.SemanticSearchStreaming(ctx, repoPath, query, topK, rerank, func(sr SearchResult) error {
        results = append(results, sr)
        return nil
    })
    return results, err
}

// SemanticSearchStreaming emits results progressively as they're retrieved and reranked.
// This enables GitHub Copilot-style progressive rendering and early cancellation.
func (h *HybridProvider) SemanticSearchStreaming(ctx context.Context, repoPath string, query string, topK int, rerank bool, emit func(SearchResult) error) error {
    if err := h.ensureCollection(ctx, safeCollectionName(repoPath)); err != nil { return err }
    
    // Check cancellation before expensive embed
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    qemb, err := h.tei.Embed(ctx, []string{query})
    if err != nil { return err }
    
    // Check cancellation before query
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    qr, err := h.chroma.Query(ctx, h.collectionID, qemb, topK)
    if err != nil { return err }
    if len(qr.Documents) == 0 { return nil }

    // Flatten first query results set
    docs := qr.Documents[0]
    metas := qr.Metadatas[0]
    dists := qr.Distances[0]
    
    // Emit results progressively (VSCode Copilot pattern)
    results := make([]SearchResult, len(docs))
    for i := range docs {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
        
        chunk := DocumentChunk{
            ID:        safeString(qr.IDs[0], i),
            RepoPath:  strMeta(metas[i], "repo_path"),
            CommitHash: strMeta(metas[i], "commit_hash"),
            FilePath:  strMeta(metas[i], "file_path"),
            StartLine: intMeta(metas[i], "start_line"),
            EndLine:   intMeta(metas[i], "end_line"),
            BlobHash:  strMeta(metas[i], "blob_hash"),
            Language:  strMeta(metas[i], "language"),
            Content:   docs[i],
            Meta:      convertMeta(metas[i]),
            CreatedAt: time.Now().UTC(),
        }
        score := 1.0 - dists[i]
        results[i] = SearchResult{Chunk: chunk, Score: score}
        
        // Emit immediately if no reranking (progressive results)
        if !rerank {
            if err := emit(results[i]); err != nil {
                return err
            }
        }
    }

    // If reranking, do it in batch then emit sorted results
    if rerank && len(results) > 1 {
        items := make([]RerankItem, len(results))
        for i := range results { items[i] = RerankItem{Text: results[i].Chunk.Content} }
        items, err = h.tei.Rerank(ctx, query, items)
        if err == nil {
            for i := range results { results[i].Score = items[i].Score }
            sortByScoreDesc(results)
        }
        // Emit reranked results
        for i := range results {
            select {
            case <-ctx.Done():
                return ctx.Err()
            default:
            }
            if err := emit(results[i]); err != nil {
                return err
            }
        }
    }
    
    return nil
}

func safeCollectionName(repoPath string) string {
    // Use base name + hash of abs path to avoid collisions
    base := filepath.Base(repoPath)
    sum := sha1.Sum([]byte(repoPath))
    return fmt.Sprintf("%s-%s", base, hex.EncodeToString(sum[:6]))
}

func chunkID(commitHash, path string, start, end int) string {
    h := sha1.Sum([]byte(fmt.Sprintf("%s|%s|%d|%d", commitHash, path, start, end)))
    return hex.EncodeToString(h[:])
}

func skipPath(p string) bool {
    p = strings.ToLower(p)
    if strings.HasPrefix(p, "vendor/") || strings.Contains(p, "/vendor/") { return true }
    if strings.HasSuffix(p, ".png") || strings.HasSuffix(p, ".jpg") || strings.HasSuffix(p, ".gif") { return true }
    if strings.HasSuffix(p, ".zip") || strings.HasSuffix(p, ".pdf") || strings.HasSuffix(p, ".jar") { return true }
    return false
}

func languageFromPath(p string) string {
    ext := strings.ToLower(filepath.Ext(p))
    switch ext {
    case ".go": return "go"
    case ".js": return "javascript"
    case ".ts": return "typescript"
    case ".py": return "python"
    case ".java": return "java"
    case ".rb": return "ruby"
    case ".rs": return "rust"
    case ".cpp", ".cc", ".cxx": return "cpp"
    case ".c": return "c"
    case ".cs": return "csharp"
    case ".md": return "markdown"
    default: return strings.TrimPrefix(ext, ".")
    }
}

func safeString(ids []string, i int) string {
    if i >= 0 && i < len(ids) { return ids[i] }
    return ""
}

func strMeta(m map[string]any, k string) string {
    if v, ok := m[k]; ok {
        if s, ok := v.(string); ok { return s }
        return fmt.Sprintf("%v", v)
    }
    return ""
}

func intMeta(m map[string]any, k string) int {
    if v, ok := m[k]; ok {
        switch t := v.(type) {
        case float64: return int(t)
        case int: return t
        case string:
            var n int
            fmt.Sscanf(t, "%d", &n)
            return n
        }
    }
    return 0
}

func convertMeta(m map[string]any) map[string]string {
    out := make(map[string]string, len(m))
    for k, v := range m { out[k] = fmt.Sprintf("%v", v) }
    return out
}

func sortByScoreDesc(res []SearchResult) {
    for i := 0; i < len(res); i++ {
        for j := i+1; j < len(res); j++ {
            if res[j].Score > res[i].Score { res[i], res[j] = res[j], res[i] }
        }
    }
}
