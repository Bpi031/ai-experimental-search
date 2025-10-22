package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type ChromaClient struct {
	BaseURL string
	http    *http.Client
}

func NewChromaClient(baseURL string) *ChromaClient {
	return &ChromaClient{
		BaseURL: baseURL,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

// Chroma uses a REST API (v0.4+) with endpoints like /api/v1/collections
// We'll implement just enough for: ensure collection, upsert, query.

type chromaCollection struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (c *ChromaClient) EnsureCollection(ctx context.Context, name string) (string, error) {
	// Try to get existing, else create
	escName := url.QueryEscape(name)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/v1/collections?name="+escName, nil)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	// Read body fully to support flexible decoding
	if resp.StatusCode == 200 {
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		// Try single-object shape
		var one chromaCollection
		if err := json.Unmarshal(data, &one); err == nil && one.ID != "" {
			return one.ID, nil
		}
		// Try wrapper shape {"collections":[...]}
		var wrap struct{ Collections []chromaCollection `json:"collections"` }
		if err := json.Unmarshal(data, &wrap); err == nil && len(wrap.Collections) > 0 {
			return wrap.Collections[0].ID, nil
		}
		// fallthrough to create
	} else {
		// Ensure body closed for non-200
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	// Create
	body, _ := json.Marshal(map[string]any{"name": name})
	req, _ = http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/v1/collections", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err = c.http.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		var out chromaCollection
		if err := json.NewDecoder(resp.Body).Decode(&out); err == nil && out.ID != "" {
			resp.Body.Close()
			return out.ID, nil
		}
		// Drain and proceed to GET fallback
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	} else {
		// Drain body and try GET fallback (collection may already exist)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	// Fallback: GET again; if collection exists now, return it
	req, _ = http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/v1/collections?name="+escName, nil)
	resp, err = c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		data, _ := io.ReadAll(resp.Body)
		var one chromaCollection
		if err := json.Unmarshal(data, &one); err == nil && one.ID != "" {
			return one.ID, nil
		}
		var wrap struct{ Collections []chromaCollection `json:"collections"` }
		if err := json.Unmarshal(data, &wrap); err == nil && len(wrap.Collections) > 0 {
			return wrap.Collections[0].ID, nil
		}
	}
	return "", fmt.Errorf("chroma create collection failed: collection not available after create attempt")
}

type UpsertItem struct {
	ID        string
	Embedding []float32
	Document  string
	Metadata  map[string]any
}

func (c *ChromaClient) Upsert(ctx context.Context, collectionID string, items []UpsertItem) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]string, len(items))
	embeddings := make([][]float32, len(items))
	documents := make([]string, len(items))
	metadatas := make([]map[string]any, len(items))
	for i, it := range items {
		ids[i] = it.ID
		embeddings[i] = it.Embedding
		documents[i] = it.Document
		metadatas[i] = it.Metadata
	}
	body, _ := json.Marshal(map[string]any{
		"ids":        ids,
		"embeddings": embeddings,
		"documents":  documents,
		"metadatas":  metadatas,
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/v1/collections/"+collectionID+"/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return fmt.Errorf("chroma upsert failed: %s", resp.Status)
	}
	return nil
}

type QueryResult struct {
	IDs       [][]string         `json:"ids"`
	Distances [][]float64        `json:"distances"`
	Documents [][]string         `json:"documents"`
	Metadatas [][]map[string]any `json:"metadatas"`
}

func (c *ChromaClient) Query(ctx context.Context, collectionID string, queryEmbeddings [][]float32, topK int) (QueryResult, error) {
	body, _ := json.Marshal(map[string]any{
		"query_embeddings": queryEmbeddings,
		"n_results":        topK,
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/v1/collections/"+collectionID+"/query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return QueryResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return QueryResult{}, fmt.Errorf("chroma query failed: %s", resp.Status)
	}
	var out QueryResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return QueryResult{}, err
	}
	return out, nil
}
