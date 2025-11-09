package ai

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v6"
)

// GrepMatch represents a single match from grep search.
type GrepMatch struct {
	FilePath    string   `json:"file_path"`
	LineNumber  int      `json:"line_number"`
	Column      int      `json:"column"`
	Line        string   `json:"line"`
	Match       string   `json:"match"`
	Before      []string `json:"before,omitempty"` // Context lines before
	After       []string `json:"after,omitempty"`  // Context lines after
}

// GrepOptions configures grep search behavior.
type GrepOptions struct {
	Pattern         string   // Search pattern (text or regex)
	UseRegex        bool     // Treat pattern as regex
	CaseInsensitive bool     // Case-insensitive search
	WholeWord       bool     // Match whole words only
	MaxMatches      int      // Max matches to return (default: 1000)
	ContextLines    int      // Number of context lines before/after (default: 0)
	
	// File filtering
	Include         []string // Glob patterns to include (e.g., ["*.go", "*.js"])
	Exclude         []string // Glob patterns to exclude (e.g., ["*_test.go"])
	FilePath        string   // Search only in specific file
	
	// Advanced
	Invert          bool     // Invert match (lines NOT matching)
	MaxLineLength   int      // Skip lines longer than this (prevent huge matches)
}

// GrepSearcher provides fast text search across files.
type GrepSearcher struct {
	repo *git.Repository
	path string
}

// NewGrepSearcher creates a new grep searcher.
func NewGrepSearcher(repo *git.Repository, repoPath string) *GrepSearcher {
	return &GrepSearcher{
		repo: repo,
		path: repoPath,
	}
}

// GrepSearch performs fast text/regex search across repository files.
func (gs *GrepSearcher) GrepSearch(ctx context.Context, opts GrepOptions) ([]GrepMatch, error) {
	// Set defaults
	if opts.MaxMatches == 0 {
		opts.MaxMatches = 1000
	}
	if opts.MaxLineLength == 0 {
		opts.MaxLineLength = 10000 // 10KB per line max
	}

	// Compile regex if needed
	var re *regexp.Regexp
	var err error
	if opts.UseRegex {
		flags := ""
		if opts.CaseInsensitive {
			flags = "(?i)"
		}
		re, err = regexp.Compile(flags + opts.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex: %w", err)
		}
	}

	matches := []GrepMatch{}

	// Get files to search
	files, err := gs.getFilesToSearch(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Search each file
	for _, file := range files {
		select {
		case <-ctx.Done():
			return matches, ctx.Err()
		default:
		}

		if len(matches) >= opts.MaxMatches {
			break
		}

		fileMatches, err := gs.searchFile(file, opts, re)
		if err != nil {
			continue // Skip files with errors
		}

		matches = append(matches, fileMatches...)
		
		// Trim if over limit
		if len(matches) > opts.MaxMatches {
			matches = matches[:opts.MaxMatches]
			break
		}
	}

	return matches, nil
}

// searchFile searches within a single file.
func (gs *GrepSearcher) searchFile(filePath string, opts GrepOptions, re *regexp.Regexp) ([]GrepMatch, error) {
	fullPath := filepath.Join(gs.path, filePath)
	
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	matches := []GrepMatch{}
	scanner := bufio.NewScanner(file)
	
	// Increase buffer size for large lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, opts.MaxLineLength)

	lineNum := 0
	var contextBuffer []string
	if opts.ContextLines > 0 {
		contextBuffer = make([]string, 0, opts.ContextLines)
	}

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip very long lines
		if len(line) > opts.MaxLineLength {
			continue
		}

		// Check if line matches
		matched, matchStr, col := gs.matchLine(line, opts, re)

		if matched != opts.Invert {
			// Create match
			match := GrepMatch{
				FilePath:   filePath,
				LineNumber: lineNum,
				Column:     col,
				Line:       line,
				Match:      matchStr,
			}

			// Add context lines before
			if opts.ContextLines > 0 {
				match.Before = make([]string, len(contextBuffer))
				copy(match.Before, contextBuffer)
			}

			matches = append(matches, match)

			// Clear context buffer after match
			if opts.ContextLines > 0 {
				contextBuffer = contextBuffer[:0]
			}
		} else {
			// Update context buffer
			if opts.ContextLines > 0 {
				contextBuffer = append(contextBuffer, line)
				if len(contextBuffer) > opts.ContextLines {
					contextBuffer = contextBuffer[1:]
				}
			}
		}
	}

	// Add context lines after for each match
	if opts.ContextLines > 0 {
		gs.addAfterContext(fullPath, matches, opts.ContextLines)
	}

	return matches, nil
}

// matchLine checks if a line matches the pattern.
func (gs *GrepSearcher) matchLine(line string, opts GrepOptions, re *regexp.Regexp) (bool, string, int) {
	if opts.UseRegex {
		if loc := re.FindStringIndex(line); loc != nil {
			return true, line[loc[0]:loc[1]], loc[0]
		}
		return false, "", 0
	}

	// Plain text search
	searchLine := line
	searchPattern := opts.Pattern

	if opts.CaseInsensitive {
		searchLine = strings.ToLower(line)
		searchPattern = strings.ToLower(opts.Pattern)
	}

	if opts.WholeWord {
		// Check word boundaries
		index := strings.Index(searchLine, searchPattern)
		if index != -1 {
			// Check boundaries
			before := index == 0 || !isWordChar(rune(searchLine[index-1]))
			after := index+len(searchPattern) >= len(searchLine) || 
			         !isWordChar(rune(searchLine[index+len(searchPattern)]))
			
			if before && after {
				return true, line[index:index+len(searchPattern)], index
			}
		}
		return false, "", 0
	}

	if index := strings.Index(searchLine, searchPattern); index != -1 {
		return true, line[index:index+len(opts.Pattern)], index
	}

	return false, "", 0
}

// isWordChar checks if a rune is a word character.
func isWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || 
	       (r >= '0' && r <= '9') || r == '_'
}

// addAfterContext adds context lines after matches.
func (gs *GrepSearcher) addAfterContext(filePath string, matches []GrepMatch, contextLines int) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	matchIdx := 0

	for scanner.Scan() && matchIdx < len(matches) {
		lineNum++
		
		// Check if we need to collect after context
		if lineNum > matches[matchIdx].LineNumber && 
		   lineNum <= matches[matchIdx].LineNumber+contextLines {
			matches[matchIdx].After = append(matches[matchIdx].After, scanner.Text())
		}

		// Move to next match
		if lineNum == matches[matchIdx].LineNumber+contextLines {
			matchIdx++
		}
	}

	return nil
}

// getFilesToSearch returns list of files to search based on options.
func (gs *GrepSearcher) getFilesToSearch(ctx context.Context, opts GrepOptions) ([]string, error) {
	// If specific file requested
	if opts.FilePath != "" {
		return []string{opts.FilePath}, nil
	}

	// Get all files
	fs := NewFileSearcher(gs.repo, gs.path)
	allFiles, err := fs.ListAllFiles(ctx)
	if err != nil {
		return nil, err
	}

	// Filter by include/exclude patterns
	filtered := []string{}
	for _, file := range allFiles {
		// Check exclude patterns
		excluded := false
		for _, pattern := range opts.Exclude {
			matched, _ := filepath.Match(pattern, filepath.Base(file))
			if matched {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		// Check include patterns (if specified)
		if len(opts.Include) > 0 {
			included := false
			for _, pattern := range opts.Include {
				matched, _ := filepath.Match(pattern, filepath.Base(file))
				if matched {
					included = true
					break
				}
			}
			if !included {
				continue
			}
		}

		filtered = append(filtered, file)
	}

	return filtered, nil
}

// GrepCount returns count of matches without returning all results (faster).
func (gs *GrepSearcher) GrepCount(ctx context.Context, opts GrepOptions) (int, error) {
	opts.ContextLines = 0 // Don't need context for counting
	matches, err := gs.GrepSearch(ctx, opts)
	return len(matches), err
}

// GrepFiles returns only filenames containing matches (no line details).
func (gs *GrepSearcher) GrepFiles(ctx context.Context, opts GrepOptions) ([]string, error) {
	matches, err := gs.GrepSearch(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Deduplicate files
	fileSet := make(map[string]bool)
	for _, match := range matches {
		fileSet[match.FilePath] = true
	}

	files := make([]string, 0, len(fileSet))
	for file := range fileSet {
		files = append(files, file)
	}

	return files, nil
}

// Repository methods for grep search

// GrepSearch performs text/regex search across repository.
func (r *Repository) GrepSearch(ctx context.Context, opts GrepOptions) ([]GrepMatch, error) {
	repo, err := git.PlainOpen(r.Path)
	if err != nil {
		return nil, err
	}
	searcher := NewGrepSearcher(repo, r.Path)
	return searcher.GrepSearch(ctx, opts)
}

// GrepCount returns count of matches.
func (r *Repository) GrepCount(ctx context.Context, opts GrepOptions) (int, error) {
	repo, err := git.PlainOpen(r.Path)
	if err != nil {
		return 0, err
	}
	searcher := NewGrepSearcher(repo, r.Path)
	return searcher.GrepCount(ctx, opts)
}

// GrepFiles returns files containing matches.
func (r *Repository) GrepFiles(ctx context.Context, opts GrepOptions) ([]string, error) {
	repo, err := git.PlainOpen(r.Path)
	if err != nil {
		return nil, err
	}
	searcher := NewGrepSearcher(repo, r.Path)
	return searcher.GrepFiles(ctx, opts)
}

// Simple wrapper functions

// GrepRepo performs grep search on a repository.
func GrepRepo(ctx context.Context, repoPath, pattern string, useRegex bool, maxMatches int) ([]GrepMatch, error) {
	repo, err := Open(repoPath)
	if err != nil {
		return nil, err
	}

	return repo.GrepSearch(ctx, GrepOptions{
		Pattern:    pattern,
		UseRegex:   useRegex,
		MaxMatches: maxMatches,
	})
}
