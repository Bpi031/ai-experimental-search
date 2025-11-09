package ai

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v6"
	_ "github.com/go-git/go-git/v6/plumbing"
	_ "github.com/go-git/go-git/v6/plumbing/filemode"
	_ "github.com/go-git/go-git/v6/plumbing/format/index"
	_ "github.com/go-git/go-git/v6/plumbing/object"
)

// FileMetadata contains comprehensive file metadata.
type FileMetadata struct {
	// Basic info
	FilePath    string      `json:"file_path"`
	FileName    string      `json:"file_name"`
	Size        int64       `json:"size"`
	SizeHuman   string      `json:"size_human"`
	
	// Timestamps
	LastModified time.Time  `json:"last_modified"`
	Created      time.Time  `json:"created"`
	
	// File system info
	Permissions  os.FileMode `json:"permissions"`
	IsExecutable bool        `json:"is_executable"`
	IsSymlink    bool        `json:"is_symlink"`
	
	// Git info
	GitStatus    string      `json:"git_status"` // "tracked", "modified", "staged", "untracked"
	BlobHash     string      `json:"blob_hash"`
	LastCommit   *CommitInfo `json:"last_commit,omitempty"`
	
	// Content info
	Language     string      `json:"language"`
	Extension    string      `json:"extension"`
	LineCount    int64       `json:"line_count"`
	IsBinary     bool        `json:"is_binary"`
	Encoding     string      `json:"encoding"`
}

// FileMetadataProvider provides file metadata operations.
type FileMetadataProvider struct {
	repo *git.Repository
	path string
}

// NewFileMetadataProvider creates a new file metadata provider.
func NewFileMetadataProvider(repo *git.Repository, repoPath string) *FileMetadataProvider {
	return &FileMetadataProvider{
		repo: repo,
		path: repoPath,
	}
}

// GetFileMetadata returns comprehensive metadata for a file.
func (fmp *FileMetadataProvider) GetFileMetadata(ctx context.Context, filePath string) (*FileMetadata, error) {
	metadata := &FileMetadata{
		FilePath:  filePath,
		FileName:  filepath.Base(filePath),
		Extension: filepath.Ext(filePath),
		Language:  detectLanguage(filePath),
	}

	// Get file system info
	fullPath := filepath.Join(fmp.path, filePath)
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		// File might be in git but not in working tree (deleted)
		return fmp.getMetadataFromGit(ctx, filePath)
	}

	metadata.Size = fileInfo.Size()
	metadata.SizeHuman = formatBytes(fileInfo.Size())
	metadata.LastModified = fileInfo.ModTime()
	metadata.Permissions = fileInfo.Mode()
	metadata.IsExecutable = fileInfo.Mode()&0111 != 0
	metadata.IsSymlink = fileInfo.Mode()&os.ModeSymlink != 0

	// Check if binary
	metadata.IsBinary, metadata.Encoding = detectFileType(fullPath)

	// Count lines for text files
	if !metadata.IsBinary {
		lines, err := countFileLines(fullPath)
		if err == nil {
			metadata.LineCount = lines
		}
	}

	// Get Git status
	metadata.GitStatus = fmp.getGitStatus(filePath)

	// Get Git info (blob hash, last commit)
	if err := fmp.enrichWithGitInfo(ctx, metadata); err == nil {
		// Successfully got Git info
	}

	return metadata, nil
}

// GetBulkMetadata returns metadata for multiple files efficiently.
func (fmp *FileMetadataProvider) GetBulkMetadata(ctx context.Context, filePaths []string) ([]*FileMetadata, error) {
	results := make([]*FileMetadata, 0, len(filePaths))

	for _, filePath := range filePaths {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		metadata, err := fmp.GetFileMetadata(ctx, filePath)
		if err != nil {
			// Skip files with errors
			continue
		}
		results = append(results, metadata)
	}

	return results, nil
}

// getMetadataFromGit gets metadata for a file that only exists in Git.
func (fmp *FileMetadataProvider) getMetadataFromGit(ctx context.Context, filePath string) (*FileMetadata, error) {
	ref, err := fmp.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	commit, err := fmp.repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	file, err := commit.File(filePath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	size := file.Size
	
	metadata := &FileMetadata{
		FilePath:  filePath,
		FileName:  filepath.Base(filePath),
		Extension: filepath.Ext(filePath),
		Language:  detectLanguage(filePath),
		Size:      size,
		SizeHuman: formatBytes(size),
		GitStatus: "tracked",
		BlobHash:  file.Hash.String(),
	}

	// Check if binary
	isBinary, _ := file.IsBinary()
	metadata.IsBinary = isBinary

	// Count lines if text
	if !isBinary {
		content, err := file.Contents()
		if err == nil {
			metadata.LineCount = countLines(content)
		}
	}

	return metadata, nil
}

// enrichWithGitInfo adds Git-specific information to metadata.
func (fmp *FileMetadataProvider) enrichWithGitInfo(ctx context.Context, metadata *FileMetadata) error {
	ref, err := fmp.repo.Head()
	if err != nil {
		return err
	}

	commit, err := fmp.repo.CommitObject(ref.Hash())
	if err != nil {
		return err
	}

	file, err := commit.File(metadata.FilePath)
	if err != nil {
		return err
	}

	metadata.BlobHash = file.Hash.String()

	// Get last commit for this file
	commitIter, err := fmp.repo.Log(&git.LogOptions{
		From:     ref.Hash(),
		FileName: &metadata.FilePath,
	})
	if err != nil {
		return err
	}
	defer commitIter.Close()

	lastCommit, err := commitIter.Next()
	if err == nil && lastCommit != nil {
		metadata.LastCommit = &CommitInfo{
			Hash:       lastCommit.Hash.String(),
			ShortHash:  lastCommit.Hash.String()[:7],
			Author:     lastCommit.Author.Name,
			AuthorTime: lastCommit.Author.When,
			Message:    lastCommit.Message,
			ShortMsg:   getShortMessage(lastCommit.Message),
		}
		metadata.Created = lastCommit.Author.When
	}

	return nil
}

// getGitStatus determines the Git status of a file.
func (fmp *FileMetadataProvider) getGitStatus(filePath string) string {
	w, err := fmp.repo.Worktree()
	if err != nil {
		return "unknown"
	}

	status, err := w.Status()
	if err != nil {
		return "unknown"
	}

	// Check if file is in the status map
	fileStatus, exists := status[filePath]
	if !exists {
		// File is tracked and clean (not in status means no changes)
		return "tracked"
	}
	
	// If file is unmodified in both staging and worktree
	if fileStatus.Staging == git.Unmodified && fileStatus.Worktree == git.Unmodified {
		return "tracked"
	}
	
	switch fileStatus.Staging {
	case git.Added:
		return "staged-new"
	case git.Modified:
		return "staged-modified"
	case git.Deleted:
		return "staged-deleted"
	case git.Renamed:
		return "staged-renamed"
	case git.Copied:
		return "staged-copied"
	}

	switch fileStatus.Worktree {
	case git.Untracked:
		return "untracked"
	case git.Modified:
		return "modified"
	case git.Deleted:
		return "deleted"
	case git.Renamed:
		return "renamed"
	case git.Added:
		return "added"
	}

	return "tracked"
}

// detectFileType determines if a file is binary and its encoding.
func detectFileType(filePath string) (bool, string) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, "unknown"
	}
	defer file.Close()

	// Read first 512 bytes for detection
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && n == 0 {
		return false, "unknown"
	}

	// Check for null bytes (binary indicator)
	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			return true, "binary"
		}
	}

	// Simple encoding detection
	encoding := "UTF-8" // Default assumption
	
	// Check for UTF-8 BOM
	if n >= 3 && buffer[0] == 0xEF && buffer[1] == 0xBB && buffer[2] == 0xBF {
		encoding = "UTF-8 BOM"
	}

	return false, encoding
}

// countFileLines counts lines in a file.
func countFileLines(filePath string) (int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	buf := make([]byte, 32*1024)
	lines := int64(0)
	lastByte := byte(0)
	hasContent := false

	for {
		c, err := file.Read(buf)
		if c > 0 {
			hasContent = true
			for i := 0; i < c; i++ {
				if buf[i] == '\n' {
					lines++
				}
			}
			lastByte = buf[c-1]
		}

		if err != nil {
			break
		}
	}

	// If file has content and doesn't end with newline, count the last line
	if hasContent && lastByte != '\n' {
		lines++
	}

	return lines, nil
}

// countLines counts lines in a string.
func countLines(content string) int64 {
	if content == "" {
		return 0
	}

	lines := int64(0)
	for i := 0; i < len(content); i++ {
		if content[i] == '\n' {
			lines++
		}
	}

	// If content doesn't end with newline, add 1 for the last line
	if len(content) > 0 && content[len(content)-1] != '\n' {
		lines++
	}

	return lines
}

// formatBytes formats byte count as human-readable string.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB", "PB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// Repository methods for file metadata

// GetFileMetadata returns metadata for a file.
func (r *Repository) GetFileMetadata(ctx context.Context, filePath string) (*FileMetadata, error) {
	repo, err := git.PlainOpen(r.Path)
	if err != nil {
		return nil, err
	}
	provider := NewFileMetadataProvider(repo, r.Path)
	return provider.GetFileMetadata(ctx, filePath)
}

// GetBulkMetadata returns metadata for multiple files.
func (r *Repository) GetBulkMetadata(ctx context.Context, filePaths []string) ([]*FileMetadata, error) {
	repo, err := git.PlainOpen(r.Path)
	if err != nil {
		return nil, err
	}
	provider := NewFileMetadataProvider(repo, r.Path)
	return provider.GetBulkMetadata(ctx, filePaths)
}
