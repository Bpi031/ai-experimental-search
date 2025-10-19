package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type TEIClient struct {
	Endpoint string
	http     *http.Client
}

func NewTEIClient(endpoint string) *TEIClient {
	return &TEIClient{
		Endpoint: endpoint,
		http:     &http.Client{Timeout: 60 * time.Second},
	}
}

type teiEmbedRequest struct {
	Inputs []string `json:"inputs"`
}

type teiEmbedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

func (c *TEIClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	body, _ := json.Marshal(teiEmbedRequest{Inputs: texts})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint+"/embed", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("tei embed failed: %s", resp.Status)
	}
	// Read entire body to support multiple possible response shapes
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// Trim leading spaces to inspect first char
	s := strings.TrimLeftFunc(string(data), func(r rune) bool { return r == ' ' || r == '\n' || r == '\t' || r == '\r' })
	if len(s) == 0 {
		return nil, errors.New("tei: empty response")
	}
	if s[0] == '[' { // direct array: [[...]]
		var arr [][]float32
		if err := json.Unmarshal([]byte(s), &arr); err != nil {
			return nil, err
		}
		if len(arr) != len(texts) {
			return nil, errors.New("tei: embeddings count mismatch")
		}
		return arr, nil
	}
	// object shape: {"embeddings": [[...]]}
	var out teiEmbedResponse
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, err
	}
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
	Query string   `json:"query"`
	Texts []string `json:"texts"`
}

type teiRerankResponse struct {
	Scores []float64 `json:"scores"`
}

func (c *TEIClient) Rerank(ctx context.Context, query string, items []RerankItem) ([]RerankItem, error) {
	if len(items) == 0 {
		return items, nil
	}
	passages := make([]string, len(items))
	for i := range items {
		passages[i] = items[i].Text
	}
	body, _ := json.Marshal(teiRerankRequest{Query: query, Texts: passages})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint+"/rerank", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		// Some TEI models do not support rerank; treat as no-op
		return items, nil
	}
	var out teiRerankResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return items, nil
	}
	if len(out.Scores) != len(items) {
		return nil, errors.New("tei: rerank scores mismatch")
	}
	for i := range items {
		items[i].Score = out.Scores[i]
	}
	return items, nil
}
