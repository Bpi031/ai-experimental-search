package ai

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/object"
)

// WorkspaceStats provides comprehensive statistics about the repository.
type WorkspaceStats struct {
	// File counts
	TotalFiles      int               `json:"total_files"`
	FilesByLanguage map[string]int    `json:"files_by_language"`
	FilesByExt      map[string]int    `json:"files_by_extension"`
	
	// Lines of code
	TotalLines       int64             `json:"total_lines"`
	LinesByLanguage  map[string]int64  `json:"lines_by_language"`
	
	// Size metrics
	TotalSize        int64             `json:"total_size_bytes"`
	SizeByLanguage   map[string]int64  `json:"size_by_language"`
	
	// Directory structure
	MaxDepth         int               `json:"max_depth"`
	DirectoryCount   int               `json:"directory_count"`
	TopDirectories   []DirectoryInfo   `json:"top_directories"`
	
	// Language breakdown
	PrimaryLanguage  string            `json:"primary_language"`
	LanguagePercent  map[string]float64 `json:"language_percent"`
	
	// Metadata
	CommitHash       string            `json:"commit_hash"`
	LastModified     string            `json:"last_modified"`
}

// DirectoryInfo contains statistics for a directory.
type DirectoryInfo struct {
	Path      string  `json:"path"`
	FileCount int     `json:"file_count"`
	TotalSize int64   `json:"total_size"`
	Depth     int     `json:"depth"`
}

// LanguageInfo contains detailed language statistics.
type LanguageInfo struct {
	Name       string  `json:"name"`
	Files      int     `json:"files"`
	Lines      int64   `json:"lines"`
	Bytes      int64   `json:"bytes"`
	Percentage float64 `json:"percentage"`
}

// WorkspaceAnalyzer analyzes repository workspace statistics.
type WorkspaceAnalyzer struct {
	repo *git.Repository
	path string
}

// NewWorkspaceAnalyzer creates a new workspace analyzer.
func NewWorkspaceAnalyzer(repo *git.Repository, repoPath string) *WorkspaceAnalyzer {
	return &WorkspaceAnalyzer{
		repo: repo,
		path: repoPath,
	}
}

// GetWorkspaceStats computes comprehensive workspace statistics.
func (wa *WorkspaceAnalyzer) GetWorkspaceStats(ctx context.Context) (*WorkspaceStats, error) {
	stats := &WorkspaceStats{
		FilesByLanguage: make(map[string]int),
		FilesByExt:      make(map[string]int),
		LinesByLanguage: make(map[string]int64),
		SizeByLanguage:  make(map[string]int64),
		LanguagePercent: make(map[string]float64),
		TopDirectories:  []DirectoryInfo{},
	}

	// Get HEAD commit
	ref, err := wa.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	stats.CommitHash = ref.Hash().String()

	commit, err := wa.repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	stats.LastModified = commit.Committer.When.Format("2006-01-02 15:04:05")

	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}

	// Track directories
	dirStats := make(map[string]*DirectoryInfo)

	// Walk tree and collect stats
	err = tree.Files().ForEach(func(f *object.File) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		filePath := f.Name
		ext := filepath.Ext(filePath)
		dir := filepath.Dir(filePath)
		
		// Skip hidden files and common non-code files
		if strings.HasPrefix(filepath.Base(filePath), ".") {
			return nil
		}

	// Get file size
	size := f.Size		// Update file counts
		stats.TotalFiles++
		stats.TotalSize += size

		// Track by extension
		if ext != "" {
			stats.FilesByExt[ext]++
		}

		// Detect language
		language := detectLanguage(filePath)
		if language != "" {
			stats.FilesByLanguage[language]++
			stats.SizeByLanguage[language] += size

			// Count lines for code files
			lines, err := wa.countLines(f)
			if err == nil {
				stats.TotalLines += lines
				stats.LinesByLanguage[language] += lines
			}
		}

		// Track directory stats
		depth := strings.Count(dir, string(filepath.Separator)) + 1
		if depth > stats.MaxDepth {
			stats.MaxDepth = depth
		}

		if _, exists := dirStats[dir]; !exists {
			dirStats[dir] = &DirectoryInfo{
				Path:      dir,
				FileCount: 0,
				TotalSize: 0,
				Depth:     depth,
			}
			stats.DirectoryCount++
		}
		dirStats[dir].FileCount++
		dirStats[dir].TotalSize += size

		return nil
	})

	if err != nil && err != context.Canceled {
		return nil, fmt.Errorf("failed to analyze tree: %w", err)
	}

	// Calculate language percentages
	if stats.TotalSize > 0 {
		for lang, size := range stats.SizeByLanguage {
			stats.LanguagePercent[lang] = float64(size) / float64(stats.TotalSize) * 100.0
		}
	}

	// Find primary language (by size)
	maxSize := int64(0)
	for lang, size := range stats.SizeByLanguage {
		if size > maxSize {
			maxSize = size
			stats.PrimaryLanguage = lang
		}
	}

	// Get top directories by file count
	topDirs := make([]DirectoryInfo, 0, len(dirStats))
	for _, info := range dirStats {
		topDirs = append(topDirs, *info)
	}
	
	// Sort by file count (descending)
	for i := 0; i < len(topDirs); i++ {
		for j := i + 1; j < len(topDirs); j++ {
			if topDirs[j].FileCount > topDirs[i].FileCount {
				topDirs[i], topDirs[j] = topDirs[j], topDirs[i]
			}
		}
	}

	// Take top 10
	if len(topDirs) > 10 {
		stats.TopDirectories = topDirs[:10]
	} else {
		stats.TopDirectories = topDirs
	}

	return stats, nil
}

// GetLanguageBreakdown returns detailed language statistics.
func (wa *WorkspaceAnalyzer) GetLanguageBreakdown(ctx context.Context) ([]LanguageInfo, error) {
	stats, err := wa.GetWorkspaceStats(ctx)
	if err != nil {
		return nil, err
	}

	languages := []LanguageInfo{}
	for lang := range stats.FilesByLanguage {
		info := LanguageInfo{
			Name:       lang,
			Files:      stats.FilesByLanguage[lang],
			Lines:      stats.LinesByLanguage[lang],
			Bytes:      stats.SizeByLanguage[lang],
			Percentage: stats.LanguagePercent[lang],
		}
		languages = append(languages, info)
	}

	// Sort by percentage (descending)
	for i := 0; i < len(languages); i++ {
		for j := i + 1; j < len(languages); j++ {
			if languages[j].Percentage > languages[i].Percentage {
				languages[i], languages[j] = languages[j], languages[i]
			}
		}
	}

	return languages, nil
}

// countLines counts the number of lines in a file.
func (wa *WorkspaceAnalyzer) countLines(f *object.File) (int64, error) {
	contents, err := f.Contents()
	if err != nil {
		return 0, err
	}

	lines := int64(strings.Count(contents, "\n"))
	if len(contents) > 0 && !strings.HasSuffix(contents, "\n") {
		lines++ // Count last line if no trailing newline
	}

	return lines, nil
}

// detectLanguage detects programming language from file extension.
func detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	
	// Language mappings
	languageMap := map[string]string{
		".go":     "Go",
		".py":     "Python",
		".js":     "JavaScript",
		".ts":     "TypeScript",
		".jsx":    "JavaScript",
		".tsx":    "TypeScript",
		".java":   "Java",
		".c":      "C",
		".cpp":    "C++",
		".cc":     "C++",
		".cxx":    "C++",
		".h":      "C/C++ Header",
		".hpp":    "C++ Header",
		".cs":     "C#",
		".rb":     "Ruby",
		".php":    "PHP",
		".swift":  "Swift",
		".kt":     "Kotlin",
		".rs":     "Rust",
		".scala":  "Scala",
		".sh":     "Shell",
		".bash":   "Shell",
		".zsh":    "Shell",
		".pl":     "Perl",
		".r":      "R",
		".m":      "Objective-C",
		".mm":     "Objective-C++",
		".sql":    "SQL",
		".html":   "HTML",
		".htm":    "HTML",
		".css":    "CSS",
		".scss":   "SCSS",
		".sass":   "Sass",
		".less":   "Less",
		".xml":    "XML",
		".json":   "JSON",
		".yaml":   "YAML",
		".yml":    "YAML",
		".toml":   "TOML",
		".md":     "Markdown",
		".txt":    "Text",
		".vue":    "Vue",
		".svelte": "Svelte",
		".dart":   "Dart",
		".lua":    "Lua",
		".vim":    "Vim Script",
		".el":     "Emacs Lisp",
		".clj":    "Clojure",
		".ex":     "Elixir",
		".exs":    "Elixir",
		".erl":    "Erlang",
		".hs":     "Haskell",
		".ml":     "OCaml",
		".proto":  "Protocol Buffers",
		".graphql": "GraphQL",
		".gql":    "GraphQL",
	}

	if lang, ok := languageMap[ext]; ok {
		return lang
	}

	return ""
}

// Repository methods for workspace stats

// GetWorkspaceStats returns comprehensive workspace statistics.
func (r *Repository) GetWorkspaceStats(ctx context.Context) (*WorkspaceStats, error) {
	repo, err := git.PlainOpen(r.Path)
	if err != nil {
		return nil, err
	}
	wa := NewWorkspaceAnalyzer(repo, r.Path)
	return wa.GetWorkspaceStats(ctx)
}

// GetLanguageBreakdown returns detailed language statistics.
func (r *Repository) GetLanguageBreakdown(ctx context.Context) ([]LanguageInfo, error) {
	repo, err := git.PlainOpen(r.Path)
	if err != nil {
		return nil, err
	}
	wa := NewWorkspaceAnalyzer(repo, r.Path)
	return wa.GetLanguageBreakdown(ctx)
}
