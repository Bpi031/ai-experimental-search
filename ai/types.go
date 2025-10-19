package ai

import "time"

// DocumentChunk is a unit of text stored in the vector DB.
// It carries enough metadata to map results back to files and commits.
type DocumentChunk struct {
    ID         string            `json:"id"`
    RepoPath   string            `json:"repo_path"`
    CommitHash string            `json:"commit_hash"`
    FilePath   string            `json:"file_path"`
    StartLine  int               `json:"start_line"`
    EndLine    int               `json:"end_line"`
    BlobHash   string            `json:"blob_hash"`
    Language   string            `json:"language"`
    Content    string            `json:"content"`
    Meta       map[string]string `json:"meta"`
    CreatedAt  time.Time         `json:"created_at"`
}

type SearchResult struct {
    Chunk DocumentChunk
    Score float64
}
