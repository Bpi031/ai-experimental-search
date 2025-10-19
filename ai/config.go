package ai

import (
    "os"
)

// Config controls how embeddings, vector DB, and LLM are wired.
// Defaults:
// - Embeddings: local TEI at http://localhost:8081
// - Vector DB: local Chroma at http://localhost:8000
// - LLM: cloud API (OpenAI) if OPENAI_API_KEY is set; otherwise unset
// You can override via environment variables below.
type Config struct {
    // Embeddings
    EmbeddingsSource string // "local-tei" | "openai" | "custom"
    TEIEndpoint      string // default: http://localhost:8081

    // Vector DB
    VectorDB   string // "chroma" | "qdrant" | "memory"
    ChromaURL  string // default: http://localhost:8000
    QdrantURL  string // optional

    // LLM provider (default: cloud)
    LLMProvider   string // "openai" | "gemini" | "claude" | "ollama"
    OpenAIAPIKey  string
    OpenAIModel   string // default: gpt-4o-mini or gpt-4-turbo-preview
    GoogleAPIKey  string
    GeminiModel   string // default: gemini-1.5-pro
    AnthropicKey  string
    ClaudeModel   string // default: claude-3-sonnet
    OllamaURL     string // default: http://localhost:11434
    OllamaModel   string // default: qwen2.5-coder:7b
}

// Default returns a Config with sensible defaults that match our docs:
// - Local embeddings (TEI), local Chroma
// - Cloud LLM (OpenAI) if key present; otherwise provider remains empty
func Default() Config {
    return Config{
        EmbeddingsSource: getenv("AI_EMBEDDINGS_SOURCE", "local-tei"),
        TEIEndpoint:      getenv("AI_TEI_ENDPOINT", "http://localhost:8081"),

        VectorDB:  getenv("AI_VECTOR_DB", "chroma"),
        ChromaURL: getenv("AI_CHROMA_URL", "http://localhost:8000"),
        QdrantURL: getenv("AI_QDRANT_URL", ""),

        LLMProvider:  getenv("AI_LLM_PROVIDER", defaultLLMProvider()),
        OpenAIAPIKey: os.Getenv("OPENAI_API_KEY"),
        OpenAIModel:  getenv("OPENAI_MODEL", "gpt-4o-mini"),
        GoogleAPIKey: os.Getenv("GOOGLE_API_KEY"),
        GeminiModel:  getenv("GEMINI_MODEL", "gemini-1.5-pro"),
        AnthropicKey: os.Getenv("ANTHROPIC_API_KEY"),
        ClaudeModel:  getenv("CLAUDE_MODEL", "claude-3-sonnet"),
        OllamaURL:    getenv("OLLAMA_URL", "http://localhost:11434"),
        OllamaModel:  getenv("OLLAMA_MODEL", "qwen2.5-coder:7b"),
    }
}

// HybridDefaults returns a configuration with the intended default behavior:
// local embeddings + local vector DB + cloud LLM when available.
func HybridDefaults() Config { return Default() }

func getenv(k, def string) string {
    if v := os.Getenv(k); v != "" {
        return v
    }
    return def
}

func defaultLLMProvider() string {
    // Prefer OpenAI if present, else Gemini, else Claude, else Ollama as fallback
    if os.Getenv("OPENAI_API_KEY") != "" {
        return "openai"
    }
    if os.Getenv("GOOGLE_API_KEY") != "" {
        return "gemini"
    }
    if os.Getenv("ANTHROPIC_API_KEY") != "" {
        return "claude"
    }
    // No cloud keys set; fall back to local if available
    return getenv("AI_LLM_PROVIDER_FALLBACK", "ollama")
}
