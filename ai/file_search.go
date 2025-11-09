package ai

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/object"
)

// FileSearcher provides fast filename-only search capabilities.
// This is separate from content search and optimized for finding files by name,
// similar to GitHub Copilot's # file references.
type FileSearcher struct {
	repo *git.Repository
	path string
}

// NewFileSearcher creates a new file searcher for the repository.
func NewFileSearcher(repo *git.Repository, repoPath string) *FileSearcher {
	return &FileSearcher{
		repo: repo,
		path: repoPath,
	}
}

// FileMatch represents a matched file with relevance scoring.
type FileMatch struct {
	FilePath  string  // Relative path from repo root
	FileName  string  // Just the filename
	Directory string  // Directory path
	Score     float64 // Relevance score (higher = better match)
	MatchType string  // "exact", "prefix", "contains", "glob"
}

// FindFilesOptions configures file search behavior.
type FindFilesOptions struct {
	Query       string   // Search query (supports glob patterns like "*.go", "test*")
	MaxResults  int      // Maximum results to return (default: 50)
	Extensions  []string // Filter by extensions (e.g., [".go", ".js"])
	ExcludeDirs []string // Directories to exclude (e.g., ["node_modules", ".git"])
	CaseSensitive bool   // Case-sensitive matching
	UseGlob     bool     // Enable glob pattern matching
}

// FindFiles searches for files by name using fast pattern matching.
// This is the primary method for filename-only search, optimized for speed.
func (fs *FileSearcher) FindFiles(ctx context.Context, opts FindFilesOptions) ([]FileMatch, error) {
	if opts.MaxResults == 0 {
		opts.MaxResults = 50
	}

	// Default exclusions
	if len(opts.ExcludeDirs) == 0 {
		opts.ExcludeDirs = []string{".git", "node_modules", "vendor", ".idea", ".vscode"}
	}

	// Get HEAD tree
	ref, err := fs.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	commit, err := fs.repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}

	matches := []FileMatch{}
	seen := make(map[string]bool)

	// Walk tree and match files
	err = tree.Files().ForEach(func(f *object.File) error {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Skip if already at max results
		if len(matches) >= opts.MaxResults {
			return nil
		}

		filePath := f.Name
		fileName := filepath.Base(filePath)
		dirPath := filepath.Dir(filePath)

		// Skip duplicates
		if seen[filePath] {
			return nil
		}

		// Check excluded directories
		if fs.isExcluded(dirPath, opts.ExcludeDirs) {
			return nil
		}

		// Filter by extension if specified
		if len(opts.Extensions) > 0 && !fs.hasExtension(fileName, opts.Extensions) {
			return nil
		}

		// Match query
		match, matchType, score := fs.matchFile(fileName, filePath, opts.Query, opts.CaseSensitive, opts.UseGlob)
		if match {
			matches = append(matches, FileMatch{
				FilePath:  filePath,
				FileName:  fileName,
				Directory: dirPath,
				Score:     score,
				MatchType: matchType,
			})
			seen[filePath] = true
		}

		return nil
	})

	if err != nil && err != context.Canceled {
		return nil, fmt.Errorf("failed to walk tree: %w", err)
	}

	// Sort by score (highest first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	// Limit results
	if len(matches) > opts.MaxResults {
		matches = matches[:opts.MaxResults]
	}

	return matches, nil
}

// matchFile determines if a file matches the query and returns score.
func (fs *FileSearcher) matchFile(fileName, filePath, query string, caseSensitive, useGlob bool) (bool, string, float64) {
	if query == "" {
		return true, "all", 0.5
	}

	queryStr := query
	fileStr := fileName
	pathStr := filePath

	if !caseSensitive {
		queryStr = strings.ToLower(query)
		fileStr = strings.ToLower(fileName)
		pathStr = strings.ToLower(filePath)
	}

	// 1. Exact match (highest score)
	if fileStr == queryStr {
		return true, "exact", 1.0
	}

	// 2. Glob pattern match
	if useGlob {
		matched, err := filepath.Match(queryStr, fileStr)
		if err == nil && matched {
			return true, "glob", 0.85
		}
	}

	// 3. Prefix match (high score)
	if strings.HasPrefix(fileStr, queryStr) {
		return true, "prefix", 0.8
	}

	// 4. Suffix match (before extension)
	baseName := strings.TrimSuffix(fileStr, filepath.Ext(fileStr))
	if strings.HasSuffix(baseName, queryStr) {
		return true, "suffix", 0.75
	}

	// 5. Contains in filename (medium-high score)
	if strings.Contains(fileStr, queryStr) {
		return true, "contains", 0.7
	}

	// 6. Contains in full path (medium score)
	if strings.Contains(pathStr, queryStr) {
		return true, "path_contains", 0.6
	}

	// 7. Fuzzy match (word boundaries)
	if fs.fuzzyMatch(fileStr, queryStr) {
		return true, "fuzzy", 0.5
	}

	return false, "", 0.0
}

// fuzzyMatch checks if all characters in query appear in order in fileName.
func (fs *FileSearcher) fuzzyMatch(fileName, query string) bool {
	if query == "" {
		return true
	}

	queryIdx := 0
	for i := 0; i < len(fileName) && queryIdx < len(query); i++ {
		if fileName[i] == query[queryIdx] {
			queryIdx++
		}
	}

	return queryIdx == len(query)
}

// isExcluded checks if a directory path should be excluded.
func (fs *FileSearcher) isExcluded(dirPath string, excludeDirs []string) bool {
	for _, excluded := range excludeDirs {
		if strings.Contains(dirPath, excluded) {
			return true
		}
	}
	return false
}

// hasExtension checks if fileName has one of the specified extensions.
func (fs *FileSearcher) hasExtension(fileName string, extensions []string) bool {
	ext := filepath.Ext(fileName)
	for _, e := range extensions {
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		if strings.EqualFold(ext, e) {
			return true
		}
	}
	return false
}

// ListAllFiles returns all files in the repository (cached for performance).
func (fs *FileSearcher) ListAllFiles(ctx context.Context) ([]string, error) {
	ref, err := fs.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	commit, err := fs.repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}

	files := []string{}
	err = tree.Files().ForEach(func(f *object.File) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		files = append(files, f.Name)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	return files, nil
}

// GetFilesByExtension returns all files with specified extensions.
func (fs *FileSearcher) GetFilesByExtension(ctx context.Context, extensions []string) ([]string, error) {
	allFiles, err := fs.ListAllFiles(ctx)
	if err != nil {
		return nil, err
	}

	filtered := []string{}
	for _, file := range allFiles {
		if fs.hasExtension(file, extensions) {
			filtered = append(filtered, file)
		}
	}

	return filtered, nil
}

// Repository methods for file search

// FindFiles searches for files by name (high-level API).
func (r *Repository) FindFiles(ctx context.Context, opts FindFilesOptions) ([]FileMatch, error) {
	repo, err := git.PlainOpen(r.Path)
	if err != nil {
		return nil, err
	}
	fs := NewFileSearcher(repo, r.Path)
	return fs.FindFiles(ctx, opts)
}

// ListAllFiles returns all files in the repository.
func (r *Repository) ListAllFiles(ctx context.Context) ([]string, error) {
	repo, err := git.PlainOpen(r.Path)
	if err != nil {
		return nil, err
	}
	fs := NewFileSearcher(repo, r.Path)
	return fs.ListAllFiles(ctx)
}

// GetFilesByExtension returns files filtered by extension.
func (r *Repository) GetFilesByExtension(ctx context.Context, extensions []string) ([]string, error) {
	repo, err := git.PlainOpen(r.Path)
	if err != nil {
		return nil, err
	}
	fs := NewFileSearcher(repo, r.Path)
	return fs.GetFilesByExtension(ctx, extensions)
}

// Simple wrapper functions for quick file search

// FindFilesByName is a simple wrapper for quick filename search.
func FindFilesByName(ctx context.Context, repoPath, query string, maxResults int) ([]FileMatch, error) {
	repo, err := Open(repoPath)
	if err != nil {
		return nil, err
	}

	return repo.FindFiles(ctx, FindFilesOptions{
		Query:      query,
		MaxResults: maxResults,
	})
}

// FindFilesByPattern searches using glob patterns (e.g., "*.go", "test*").
func FindFilesByPattern(ctx context.Context, repoPath, pattern string, maxResults int) ([]FileMatch, error) {
	repo, err := Open(repoPath)
	if err != nil {
		return nil, err
	}

	return repo.FindFiles(ctx, FindFilesOptions{
		Query:      pattern,
		MaxResults: maxResults,
		UseGlob:    true,
	})
}
