package ai

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/go-git/go-git/v6"
	_ "github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
)

// RecentFile represents a recently modified file with metadata.
type RecentFile struct {
	FilePath     string    `json:"file_path"`
	LastModified time.Time `json:"last_modified"`
	Author       string    `json:"author"`
	Email        string    `json:"email"`
	CommitHash   string    `json:"commit_hash"`
	CommitMsg    string    `json:"commit_message"`
	ChangeType   string    `json:"change_type"` // "modified", "added", "deleted"
	LinesChanged int       `json:"lines_changed"`
}

// RecentFilesOptions configures recent files retrieval.
type RecentFilesOptions struct {
	MaxFiles  int           // Maximum files to return (default: 20)
	MaxAge    time.Duration // Only include files modified within this duration (default: 7 days)
	Author    string        // Filter by author
	Extension []string      // Filter by file extensions
	Since     time.Time     // Only commits after this time
}

// RecentFilesTracker tracks recently modified files using Git history.
type RecentFilesTracker struct {
	repo *git.Repository
	path string
}

// NewRecentFilesTracker creates a new recent files tracker.
func NewRecentFilesTracker(repo *git.Repository, repoPath string) *RecentFilesTracker {
	return &RecentFilesTracker{
		repo: repo,
		path: repoPath,
	}
}

// GetRecentFiles returns recently modified files based on Git history.
func (rft *RecentFilesTracker) GetRecentFiles(ctx context.Context, opts RecentFilesOptions) ([]RecentFile, error) {
	// Set defaults
	if opts.MaxFiles == 0 {
		opts.MaxFiles = 20
	}
	if opts.MaxAge == 0 {
		opts.MaxAge = 7 * 24 * time.Hour // 7 days
	}
	if opts.Since.IsZero() {
		opts.Since = time.Now().Add(-opts.MaxAge)
	}

	// Get HEAD reference
	ref, err := rft.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Get commit history
	commitIter, err := rft.repo.Log(&git.LogOptions{
		From:  ref.Hash(),
		Since: &opts.Since,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get log: %w", err)
	}
	defer commitIter.Close()

	// Track files and their most recent modification
	fileMap := make(map[string]*RecentFile)

	// Iterate through commits
	err = commitIter.ForEach(func(c *object.Commit) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Filter by author if specified
		if opts.Author != "" && c.Author.Email != opts.Author && c.Author.Name != opts.Author {
			return nil
		}

		// Get parent commit for diff
		var parentTree *object.Tree
		if c.NumParents() > 0 {
			parent, err := c.Parent(0)
			if err == nil {
				parentTree, _ = parent.Tree()
			}
		}

		currentTree, err := c.Tree()
		if err != nil {
			return nil
		}

		// Get changes
		changes, err := rft.getChanges(parentTree, currentTree)
		if err != nil {
			return nil
		}

		// Process each changed file
		for _, change := range changes {
			filePath := change.To.Name
			if filePath == "" {
				filePath = change.From.Name
			}

			// Filter by extension if specified
			if len(opts.Extension) > 0 && !hasExtension(filePath, opts.Extension) {
				continue
			}

			// Only track if not already seen (most recent)
			if _, exists := fileMap[filePath]; !exists {
				changeType := "modified"
				if change.From.Name == "" {
					changeType = "added"
				} else if change.To.Name == "" {
					changeType = "deleted"
				}

				fileMap[filePath] = &RecentFile{
					FilePath:     filePath,
					LastModified: c.Author.When,
					Author:       c.Author.Name,
					Email:        c.Author.Email,
					CommitHash:   c.Hash.String(),
					CommitMsg:    getShortMessage(c.Message),
					ChangeType:   changeType,
					LinesChanged: 0, // Will be updated if needed
				}
			}

			// Stop if we have enough files
			if len(fileMap) >= opts.MaxFiles*2 { // Get extra for sorting
				return fmt.Errorf("stop iteration")
			}
		}

		return nil
	})

	// Convert map to slice
	recentFiles := make([]RecentFile, 0, len(fileMap))
	for _, file := range fileMap {
		recentFiles = append(recentFiles, *file)
	}

	// Sort by last modified (most recent first)
	sort.Slice(recentFiles, func(i, j int) bool {
		return recentFiles[i].LastModified.After(recentFiles[j].LastModified)
	})

	// Limit to MaxFiles
	if len(recentFiles) > opts.MaxFiles {
		recentFiles = recentFiles[:opts.MaxFiles]
	}

	return recentFiles, nil
}

// GetRecentlyModifiedFiles returns files modified in the last N commits.
func (rft *RecentFilesTracker) GetRecentlyModifiedFiles(ctx context.Context, maxCommits int) ([]string, error) {
	if maxCommits == 0 {
		maxCommits = 10
	}

	ref, err := rft.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	commitIter, err := rft.repo.Log(&git.LogOptions{
		From: ref.Hash(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get log: %w", err)
	}
	defer commitIter.Close()

	filesSet := make(map[string]bool)
	count := 0

	err = commitIter.ForEach(func(c *object.Commit) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if count >= maxCommits {
			return fmt.Errorf("stop iteration")
		}
		count++

		var parentTree *object.Tree
		if c.NumParents() > 0 {
			parent, err := c.Parent(0)
			if err == nil {
				parentTree, _ = parent.Tree()
			}
		}

		currentTree, err := c.Tree()
		if err != nil {
			return nil
		}

		changes, err := rft.getChanges(parentTree, currentTree)
		if err != nil {
			return nil
		}

		for _, change := range changes {
			if change.To.Name != "" {
				filesSet[change.To.Name] = true
			}
			if change.From.Name != "" {
				filesSet[change.From.Name] = true
			}
		}

		return nil
	})

	files := make([]string, 0, len(filesSet))
	for file := range filesSet {
		files = append(files, file)
	}

	return files, nil
}

// GetFileHistory returns commit history for a specific file.
func (rft *RecentFilesTracker) GetFileHistory(ctx context.Context, filePath string, maxCommits int) ([]RecentFile, error) {
	if maxCommits == 0 {
		maxCommits = 50
	}

	ref, err := rft.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	commitIter, err := rft.repo.Log(&git.LogOptions{
		From:     ref.Hash(),
		FileName: &filePath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get log: %w", err)
	}
	defer commitIter.Close()

	history := []RecentFile{}

	err = commitIter.ForEach(func(c *object.Commit) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if len(history) >= maxCommits {
			return fmt.Errorf("stop iteration")
		}

		history = append(history, RecentFile{
			FilePath:     filePath,
			LastModified: c.Author.When,
			Author:       c.Author.Name,
			Email:        c.Author.Email,
			CommitHash:   c.Hash.String(),
			CommitMsg:    getShortMessage(c.Message),
			ChangeType:   "modified",
		})

		return nil
	})

	return history, nil
}

// getChanges returns changes between two trees.
func (rft *RecentFilesTracker) getChanges(from, to *object.Tree) (object.Changes, error) {
	if from == nil {
		// Initial commit - all files are new
		changes := object.Changes{}
		err := to.Files().ForEach(func(f *object.File) error {
			changes = append(changes, &object.Change{
				From: object.ChangeEntry{},
				To: object.ChangeEntry{
					Name: f.Name,
					Tree: to,
					TreeEntry: object.TreeEntry{
						Name: f.Name,
						Mode: f.Mode,
						Hash: f.Hash,
					},
				},
			})
			return nil
		})
		return changes, err
	}

	return from.Diff(to)
}

// getShortMessage returns the first line of commit message.
func getShortMessage(message string) string {
	lines := splitLines(message)
	if len(lines) > 0 {
		return lines[0]
	}
	return message
}

// splitLines splits string by newlines.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// hasExtension checks if file has one of the specified extensions.
func hasExtension(filePath string, extensions []string) bool {
	for _, ext := range extensions {
		if len(filePath) >= len(ext) {
			fileExt := filePath[len(filePath)-len(ext):]
			if fileExt == ext || fileExt == "."+ext {
				return true
			}
		}
	}
	return false
}

// Repository methods for recent files

// GetRecentFiles returns recently modified files.
func (r *Repository) GetRecentFiles(ctx context.Context, opts RecentFilesOptions) ([]RecentFile, error) {
	repo, err := git.PlainOpen(r.Path)
	if err != nil {
		return nil, err
	}
	tracker := NewRecentFilesTracker(repo, r.Path)
	return tracker.GetRecentFiles(ctx, opts)
}

// GetRecentlyModifiedFiles returns files from the last N commits.
func (r *Repository) GetRecentlyModifiedFiles(ctx context.Context, maxCommits int) ([]string, error) {
	repo, err := git.PlainOpen(r.Path)
	if err != nil {
		return nil, err
	}
	tracker := NewRecentFilesTracker(repo, r.Path)
	return tracker.GetRecentlyModifiedFiles(ctx, maxCommits)
}

// GetFileHistory returns commit history for a specific file.
func (r *Repository) GetFileHistory(ctx context.Context, filePath string, maxCommits int) ([]RecentFile, error) {
	repo, err := git.PlainOpen(r.Path)
	if err != nil {
		return nil, err
	}
	tracker := NewRecentFilesTracker(repo, r.Path)
	return tracker.GetFileHistory(ctx, filePath, maxCommits)
}
