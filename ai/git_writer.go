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

// GitWriter provides Git write operations (commit, branch, etc.)
type GitWriter struct {
	repo *git.Repository
	// RequireConfirmation when true, returns actions that need confirmation
	RequireConfirmation bool
}

// NewGitWriter creates a new GitWriter instance
func NewGitWriter(repo *git.Repository) *GitWriter {
	return &GitWriter{
		repo:                repo,
		RequireConfirmation: true, // Default to safe mode
	}
}

// CommitOptions configures commit creation
type CommitOptions struct {
	// Message is the commit message (if empty, will be generated)
	Message string
	// Files to add (empty means add all modified files)
	Files []string
	// Author information
	AuthorName  string
	AuthorEmail string
	// AllowEmpty allows creating empty commits
	AllowEmpty bool
	// AmendPrevious amends the previous commit
	AmendPrevious bool
}

// CommitResult represents the result of a commit operation
type CommitResult struct {
	Hash          string
	ShortHash     string
	Message       string
	FilesChanged  []string
	NeedsConfirm  bool
	Description   string
}

// CommitChanges creates a new commit
func (g *GitWriter) CommitChanges(ctx context.Context, opts CommitOptions) (*CommitResult, error) {
	wt, err := g.repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	// Get current status
	status, err := wt.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	// Check if there are changes
	if !opts.AllowEmpty && status.IsClean() {
		return nil, fmt.Errorf("no changes to commit")
	}

	// Add files to staging
	var filesToCommit []string
	if len(opts.Files) > 0 {
		// Add specific files
		for _, file := range opts.Files {
			if _, err := wt.Add(file); err != nil {
				return nil, fmt.Errorf("failed to add file %s: %w", file, err)
			}
			filesToCommit = append(filesToCommit, file)
		}
	} else {
		// Add all modified files
		for file, fileStatus := range status {
			if fileStatus.Worktree != git.Unmodified || fileStatus.Staging != git.Unmodified {
				if _, err := wt.Add(file); err != nil {
					return nil, fmt.Errorf("failed to add file %s: %w", file, err)
				}
				filesToCommit = append(filesToCommit, file)
			}
		}
	}

	// Generate commit message if not provided
	message := opts.Message
	if message == "" {
		message = g.generateCommitMessage(filesToCommit, status)
	}

	// Build commit description
	description := g.buildCommitDescription(message, filesToCommit)

	result := &CommitResult{
		Message:      message,
		FilesChanged: filesToCommit,
		NeedsConfirm: g.RequireConfirmation,
		Description:  description,
	}

	// If confirmation not required, commit immediately
	if !g.RequireConfirmation {
		hash, err := g.executeCommit(opts, message)
		if err != nil {
			return nil, err
		}
		result.Hash = hash.String()
		result.ShortHash = hash.String()[:8]
	}

	return result, nil
}

func (g *GitWriter) generateCommitMessage(files []string, status git.Status) string {
	// AI-generated commit message based on changes
	var parts []string
	
	// Categorize changes
	added := 0
	modified := 0
	deleted := 0
	
	for _, fileStatus := range status {
		switch fileStatus.Worktree {
		case git.Added:
			added++
		case git.Modified:
			modified++
		case git.Deleted:
			deleted++
		}
	}

	// Generate message
	if added > 0 && modified == 0 && deleted == 0 {
		parts = append(parts, fmt.Sprintf("feat: add %d new file(s)", added))
	} else if modified > 0 && added == 0 && deleted == 0 {
		parts = append(parts, fmt.Sprintf("chore: update %d file(s)", modified))
	} else if deleted > 0 && added == 0 && modified == 0 {
		parts = append(parts, fmt.Sprintf("chore: remove %d file(s)", deleted))
	} else {
		// Mixed changes
		var changes []string
		if added > 0 {
			changes = append(changes, fmt.Sprintf("%d added", added))
		}
		if modified > 0 {
			changes = append(changes, fmt.Sprintf("%d modified", modified))
		}
		if deleted > 0 {
			changes = append(changes, fmt.Sprintf("%d deleted", deleted))
		}
		parts = append(parts, fmt.Sprintf("chore: update files (%s)", strings.Join(changes, ", ")))
	}

	// Add file details if small number
	if len(files) <= 3 {
		parts = append(parts, "\n\nFiles:")
		for _, file := range files {
			parts = append(parts, fmt.Sprintf("- %s", file))
		}
	}

	return strings.Join(parts, "\n")
}

func (g *GitWriter) buildCommitDescription(message string, files []string) string {
	var desc strings.Builder
	
	desc.WriteString("Commit changes:\n\n")
	desc.WriteString(fmt.Sprintf("Message: %s\n", strings.Split(message, "\n")[0]))
	desc.WriteString(fmt.Sprintf("Files: %d\n", len(files)))
	
	if len(files) <= 10 {
		desc.WriteString("\nChanged files:\n")
		for _, file := range files {
			desc.WriteString(fmt.Sprintf("  - %s\n", file))
		}
	}
	
	return desc.String()
}

func (g *GitWriter) executeCommit(opts CommitOptions, message string) (plumbing.Hash, error) {
	wt, err := g.repo.Worktree()
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to get worktree: %w", err)
	}

	commitOpts := &git.CommitOptions{
		AllowEmptyCommits: opts.AllowEmpty,
	}

	// Set author if provided
	if opts.AuthorName != "" && opts.AuthorEmail != "" {
		commitOpts.Author = &object.Signature{
			Name:  opts.AuthorName,
			Email: opts.AuthorEmail,
			When:  time.Now(),
		}
	}

	// Handle amend
	if opts.AmendPrevious {
		commitOpts.Amend = true
	}

	hash, err := wt.Commit(message, commitOpts)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to commit: %w", err)
	}

	return hash, nil
}

// ApplyCommit executes a pending commit (after confirmation)
func (g *GitWriter) ApplyCommit(result *CommitResult, opts CommitOptions) error {
	hash, err := g.executeCommit(opts, result.Message)
	if err != nil {
		return err
	}
	result.Hash = hash.String()
	result.ShortHash = hash.String()[:8]
	return nil
}

// BranchOptions configures branch operations
type BranchOptions struct {
	Name string
	// FromCommit creates branch from specific commit (empty for HEAD)
	FromCommit string
	// Checkout switches to branch after creation
	Checkout bool
	// Force overwrites existing branch
	Force bool
}

// BranchResult represents the result of a branch operation
type BranchResult struct {
	Name         string
	Hash         string
	NeedsConfirm bool
	Description  string
}

// CreateBranch creates a new branch
func (g *GitWriter) CreateBranch(ctx context.Context, opts BranchOptions) (*BranchResult, error) {
	if opts.Name == "" {
		return nil, fmt.Errorf("branch name is required")
	}

	// Check if branch already exists
	branches, err := g.repo.Branches()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	exists := false
	err = branches.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().Short() == opts.Name {
			exists = true
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to check branches: %w", err)
	}

	if exists && !opts.Force {
		return nil, fmt.Errorf("branch %s already exists", opts.Name)
	}

	// Get commit to branch from
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

	description := fmt.Sprintf("Create branch '%s' from %s", opts.Name, commitHash.String()[:8])
	if opts.Checkout {
		description += " and checkout"
	}

	result := &BranchResult{
		Name:         opts.Name,
		Hash:         commitHash.String(),
		NeedsConfirm: g.RequireConfirmation,
		Description:  description,
	}

	if !g.RequireConfirmation {
		if err := g.executeBranchCreate(opts, commitHash); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (g *GitWriter) executeBranchCreate(opts BranchOptions, commitHash plumbing.Hash) error {
	wt, err := g.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Create branch reference
	refName := plumbing.NewBranchReferenceName(opts.Name)
	ref := plumbing.NewHashReference(refName, commitHash)
	
	if err := g.repo.Storer.SetReference(ref); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	// Checkout if requested
	if opts.Checkout {
		checkoutOpts := &git.CheckoutOptions{
			Branch: refName,
			Force:  opts.Force,
		}
		if err := wt.Checkout(checkoutOpts); err != nil {
			return fmt.Errorf("failed to checkout branch: %w", err)
		}
	}

	return nil
}

// ApplyBranchCreate executes a pending branch creation (after confirmation)
func (g *GitWriter) ApplyBranchCreate(result *BranchResult, opts BranchOptions) error {
	commitHash := plumbing.NewHash(result.Hash)
	return g.executeBranchCreate(opts, commitHash)
}

// DeleteBranch deletes a branch
func (g *GitWriter) DeleteBranch(ctx context.Context, branchName string, force bool) error {
	if branchName == "" {
		return fmt.Errorf("branch name is required")
	}

	// Check if it's the current branch
	head, err := g.repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	if head.Name().Short() == branchName && !force {
		return fmt.Errorf("cannot delete current branch (use force=true to override)")
	}

	// Delete branch
	refName := plumbing.NewBranchReferenceName(branchName)
	if err := g.repo.Storer.RemoveReference(refName); err != nil {
		return fmt.Errorf("failed to delete branch: %w", err)
	}

	return nil
}
