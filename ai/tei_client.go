package ai

import (
    "bytes"
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "net/http"
    "time"
)

type TEIClient struct {
    Endpoint string
    http     *http.Client
}

func NewTEIClient(endpoint string) *TEIClient {
    return &TEIClient{
        Endpoint: endpoint,
        http: &http.Client{Timeout: 60 * time.Second},
    }
}

type teiEmbedRequest struct {
    Input []string `json:"input"`
}

type teiEmbedResponse struct {
    Embeddings [][]float32 `json:"embeddings"`
}

func (c *TEIClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
    if len(texts) == 0 {
        return nil, nil
    }
    body, _ := json.Marshal(teiEmbedRequest{Input: texts})
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint+"/embed", bytes.NewReader(body))
    if err != nil { return nil, err }
    req.Header.Set("Content-Type", "application/json")
    resp, err := c.http.Do(req)
    if err != nil { return nil, err }
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("tei embed failed: %s", resp.Status)
    }
    var out teiEmbedResponse
    if err := json.NewDecoder(resp.Body).Decode(&out); err != nil { return nil, err }
    if len(out.Embeddings) != len(texts) {
        return nil, errors.New("tei: embeddings count mismatch")
    }
    return out.Embeddings, nil
}

// RerankItem holds text to rerank; Score is output.
type RerankItem struct {
    Text  string
    Score float64
}

type teiRerankRequest struct {
    Query   string   `json:"query"`
    Passages []string `json:"passages"`
}

type teiRerankResponse struct {
    Scores []float64 `json:"scores"`
}

func (c *TEIClient) Rerank(ctx context.Context, query string, items []RerankItem) ([]RerankItem, error) {
    if len(items) == 0 {
        return items, nil
    }
    passages := make([]string, len(items))
    for i := range items { passages[i] = items[i].Text }
    body, _ := json.Marshal(teiRerankRequest{Query: query, Passages: passages})
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint+"/rerank", bytes.NewReader(body))
    if err != nil { return nil, err }
    req.Header.Set("Content-Type", "application/json")
    resp, err := c.http.Do(req)
    if err != nil { return nil, err }
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("tei rerank failed: %s", resp.Status)
    }
    var out teiRerankResponse
    if err := json.NewDecoder(resp.Body).Decode(&out); err != nil { return nil, err }
    if len(out.Scores) != len(items) {
        return nil, errors.New("tei: rerank scores mismatch")
    }
    for i := range items {
        items[i].Score = out.Scores[i]
    }
    return items, nil
}
