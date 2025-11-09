package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
)

// GitOperations provides tools for reading Git repository information
type GitOperations struct {
	repo *git.Repository
}

// NewGitOperations creates a new GitOperations instance
func NewGitOperations(repo *git.Repository) *GitOperations {
	return &GitOperations{repo: repo}
}

// DiffOptions configures how diffs are retrieved
type DiffOptions struct {
	// FromCommit specifies the starting commit (empty string for HEAD)
	FromCommit string
	// ToCommit specifies the ending commit (empty string for working tree)
	ToCommit string
	// FilePath optionally filters diff to a specific file
	FilePath string
	// MaxLines limits the number of diff lines returned
	MaxLines int
}

// DiffResult represents a file diff
type DiffResult struct {
	FilePath  string
	Additions int
	Deletions int
	Diff      string
	IsBinary  bool
}

// GetDiff retrieves diffs between commits or working tree
func (g *GitOperations) GetDiff(ctx context.Context, opts DiffOptions) ([]DiffResult, error) {
	// Default to HEAD if FromCommit not specified
	if opts.FromCommit == "" {
		ref, err := g.repo.Head()
		if err != nil {
			return nil, fmt.Errorf("failed to get HEAD: %w", err)
		}
		opts.FromCommit = ref.Hash().String()
	}

	fromHash := plumbing.NewHash(opts.FromCommit)
	fromCommit, err := g.repo.CommitObject(fromHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get from commit: %w", err)
	}

	fromTree, err := fromCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get from tree: %w", err)
	}

	var toTree *object.Tree
	if opts.ToCommit != "" {
		toHash := plumbing.NewHash(opts.ToCommit)
		toCommit, err := g.repo.CommitObject(toHash)
		if err != nil {
			return nil, fmt.Errorf("failed to get to commit: %w", err)
		}
		toTree, err = toCommit.Tree()
		if err != nil {
			return nil, fmt.Errorf("failed to get to tree: %w", err)
		}
	} else {
		// Compare with working tree
		wt, err := g.repo.Worktree()
		if err != nil {
			return nil, fmt.Errorf("failed to get worktree: %w", err)
		}
		status, err := wt.Status()
		if err != nil {
			return nil, fmt.Errorf("failed to get status: %w", err)
		}
		
		// For working tree comparison, show modified files
		var results []DiffResult
		for path, fileStatus := range status {
			if opts.FilePath != "" && path != opts.FilePath {
				continue
			}
			if fileStatus.Worktree != git.Unmodified {
				statusStr := "Modified"
				switch fileStatus.Worktree {
				case git.Added:
					statusStr = "Added"
				case git.Deleted:
					statusStr = "Deleted"
				case git.Modified:
					statusStr = "Modified"
				case git.Renamed:
					statusStr = "Renamed"
				case git.Copied:
					statusStr = "Copied"
				}
				
				results = append(results, DiffResult{
					FilePath: path,
					Diff:     fmt.Sprintf("Status: %s", statusStr),
				})
			}
		}
		return results, nil
	}

	// Get diff between trees
	changes, err := fromTree.Diff(toTree)
	if err != nil {
		return nil, fmt.Errorf("failed to diff trees: %w", err)
	}

	var results []DiffResult
	for _, change := range changes {
		if opts.FilePath != "" && change.To.Name != opts.FilePath && change.From.Name != opts.FilePath {
			continue
		}

		patch, err := change.Patch()
		if err != nil {
			continue // Skip on error
		}

		var filePath string
		if change.To.Name != "" {
			filePath = change.To.Name
		} else {
			filePath = change.From.Name
		}

		diffText := patch.String()
		if opts.MaxLines > 0 {
			lines := strings.Split(diffText, "\n")
			if len(lines) > opts.MaxLines {
				lines = lines[:opts.MaxLines]
				diffText = strings.Join(lines, "\n") + "\n... (truncated)"
			}
		}

		stats := patch.Stats()
		result := DiffResult{
			FilePath:  filePath,
			Additions: stats[0].Addition,
			Deletions: stats[0].Deletion,
			Diff:      diffText,
			IsBinary:  patch.FilePatches()[0].IsBinary(),
		}
		results = append(results, result)
	}

	return results, nil
}

// BlameOptions configures how blame is retrieved
type BlameOptions struct {
	FilePath string
	// FromCommit specifies the commit to blame from (empty for HEAD)
	FromCommit string
}

// BlameLine represents a line with blame information
type BlameLine struct {
	LineNumber int
	Line       string
	CommitHash string
	Author     string
	AuthorTime time.Time
	Message    string
}

// GetBlame retrieves line-by-line authorship information
func (g *GitOperations) GetBlame(ctx context.Context, opts BlameOptions) ([]BlameLine, error) {
	if opts.FilePath == "" {
		return nil, fmt.Errorf("file path is required")
	}

	var commitHash plumbing.Hash
	if opts.FromCommit != "" {
		commitHash = plumbing.NewHash(opts.FromCommit)
	} else {
		ref, err := g.repo.Head()
		if err != nil {
			return nil, fmt.Errorf("failed to get HEAD: %w", err)
		}
		commitHash = ref.Hash()
	}

	commit, err := g.repo.CommitObject(commitHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	blame, err := git.Blame(commit, opts.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get blame: %w", err)
	}

	var results []BlameLine
	for i, line := range blame.Lines {
		results = append(results, BlameLine{
			LineNumber: i + 1,
			Line:       line.Text,
			CommitHash: line.Hash.String()[:8],
			Author:     line.Author,
			AuthorTime: line.Date,
			Message:    strings.Split(line.Text, "\n")[0], // First line only
		})
	}

	return results, nil
}

// HistoryOptions configures commit history retrieval
type HistoryOptions struct {
	// FilePath optionally filters history to a specific file
	FilePath string
	// FromCommit specifies starting commit (empty for HEAD)
	FromCommit string
	// MaxCount limits number of commits returned
	MaxCount int
	// Author optionally filters by author
	Author string
	// Since optionally filters commits after this time
	Since *time.Time
	// Until optionally filters commits before this time
	Until *time.Time
}

// CommitInfo represents commit information
type CommitInfo struct {
	Hash       string
	ShortHash  string
	Author     string
	AuthorTime time.Time
	Message    string
	ShortMsg   string
	Files      []string
	Stats      CommitStats
}

// CommitStats represents commit statistics
type CommitStats struct {
	FilesChanged int
	Additions    int
	Deletions    int
}

// GetHistory retrieves commit history
func (g *GitOperations) GetHistory(ctx context.Context, opts HistoryOptions) ([]CommitInfo, error) {
	var commitHash plumbing.Hash
	if opts.FromCommit != "" {
		commitHash = plumbing.NewHash(opts.FromCommit)
	} else {
		ref, err := g.repo.Head()
		if err != nil {
			return nil, fmt.Errorf("failed to get HEAD: %w", err)
		}
		commitHash = ref.Hash()
	}

	logOpts := &git.LogOptions{
		From: commitHash,
	}
	if opts.FilePath != "" {
		logOpts.FileName = &opts.FilePath
	}
	if opts.Since != nil {
		logOpts.Since = opts.Since
	}
	if opts.Until != nil {
		logOpts.Until = opts.Until
	}

	commits, err := g.repo.Log(logOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to get log: %w", err)
	}

	var results []CommitInfo
	count := 0
	err = commits.ForEach(func(c *object.Commit) error {
		if opts.MaxCount > 0 && count >= opts.MaxCount {
			return fmt.Errorf("reached max count") // Stop iteration
		}

		if opts.Author != "" && !strings.Contains(c.Author.Name, opts.Author) {
			return nil // Skip this commit
		}

		// Get commit stats
		stats, err := c.Stats()
		if err != nil {
			stats = nil // Continue without stats
		}

		var files []string
		var additions, deletions int
		if stats != nil {
			for _, stat := range stats {
				files = append(files, stat.Name)
				additions += stat.Addition
				deletions += stat.Deletion
			}
		}

		shortMsg := strings.Split(c.Message, "\n")[0]
		if len(shortMsg) > 80 {
			shortMsg = shortMsg[:77] + "..."
		}

		results = append(results, CommitInfo{
			Hash:       c.Hash.String(),
			ShortHash:  c.Hash.String()[:8],
			Author:     c.Author.Name,
			AuthorTime: c.Author.When,
			Message:    c.Message,
			ShortMsg:   shortMsg,
			Files:      files,
			Stats: CommitStats{
				FilesChanged: len(files),
				Additions:    additions,
				Deletions:    deletions,
			},
		})

		count++
		return nil
	})

	if err != nil && err.Error() != "reached max count" {
		return nil, fmt.Errorf("failed to iterate commits: %w", err)
	}

	return results, nil
}
