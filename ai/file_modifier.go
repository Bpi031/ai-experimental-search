package ai

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileModifier provides safe file modification operations
type FileModifier struct {
	repoPath string
	// RequireConfirmation when true, returns actions that need confirmation
	RequireConfirmation bool
}

// NewFileModifier creates a new FileModifier instance
func NewFileModifier(repoPath string) *FileModifier {
	return &FileModifier{
		repoPath:            repoPath,
		RequireConfirmation: true, // Default to safe mode
	}
}

// EditFileOptions configures file editing
type EditFileOptions struct {
	FilePath string
	// Content to write (replaces entire file)
	Content string
	// Or use LineRanges for partial edits
	LineRanges []LineEdit
	// CreateIfMissing creates file if it doesn't exist
	CreateIfMissing bool
}

// LineEdit represents an edit to a range of lines
type LineEdit struct {
	StartLine int    // 1-based line number
	EndLine   int    // 1-based line number (inclusive)
	NewText   string // Text to replace lines with
}

// FileOperation represents a pending file operation
type FileOperation struct {
	Type        string // "edit", "create", "delete"
	FilePath    string
	OldContent  string
	NewContent  string
	Description string
	NeedsConfirm bool
}

// EditFile modifies an existing file or creates a new one
func (f *FileModifier) EditFile(ctx context.Context, opts EditFileOptions) (*FileOperation, error) {
	if opts.FilePath == "" {
		return nil, fmt.Errorf("file path is required")
	}

	fullPath := filepath.Join(f.repoPath, opts.FilePath)
	
	// Check if file exists
	oldContent := ""
	exists := true
	if data, err := os.ReadFile(fullPath); err == nil {
		oldContent = string(data)
	} else {
		exists = false
		if !opts.CreateIfMissing {
			return nil, fmt.Errorf("file does not exist: %s", opts.FilePath)
		}
	}

	// Build new content
	var newContent string
	var err error
	if opts.Content != "" {
		// Replace entire file
		newContent = opts.Content
	} else if len(opts.LineRanges) > 0 {
		// Apply line edits
		newContent, err = f.applyLineEdits(oldContent, opts.LineRanges)
		if err != nil {
			return nil, fmt.Errorf("failed to apply line edits: %w", err)
		}
	} else {
		return nil, fmt.Errorf("either Content or LineRanges must be provided")
	}

	opType := "edit"
	if !exists {
		opType = "create"
	}

	description := f.buildEditDescription(opType, opts.FilePath, oldContent, newContent)

	op := &FileOperation{
		Type:         opType,
		FilePath:     opts.FilePath,
		OldContent:   oldContent,
		NewContent:   newContent,
		Description:  description,
		NeedsConfirm: f.RequireConfirmation,
	}

	// If confirmation not required, apply immediately
	if !f.RequireConfirmation {
		if err := f.applyOperation(op); err != nil {
			return nil, err
		}
	}

	return op, nil
}

func (f *FileModifier) applyLineEdits(content string, edits []LineEdit) (string, error) {
	lines := strings.Split(content, "\n")
	
	// Sort edits by line number (descending) to avoid offset issues
	// Apply from bottom to top
	for i := len(edits) - 1; i >= 0; i-- {
		edit := edits[i]
		
		if edit.StartLine < 1 || edit.EndLine > len(lines) {
			return "", fmt.Errorf("line range %d-%d out of bounds (file has %d lines)", 
				edit.StartLine, edit.EndLine, len(lines))
		}
		
		// Replace lines
		newLines := strings.Split(edit.NewText, "\n")
		before := lines[:edit.StartLine-1]
		after := lines[edit.EndLine:]
		lines = append(before, append(newLines, after...)...)
	}
	
	return strings.Join(lines, "\n"), nil
}

func (f *FileModifier) buildEditDescription(opType, filePath, oldContent, newContent string) string {
	var desc strings.Builder
	
	if opType == "create" {
		desc.WriteString(fmt.Sprintf("Create new file: %s\n", filePath))
		desc.WriteString(fmt.Sprintf("Size: %d bytes\n", len(newContent)))
		lines := strings.Split(newContent, "\n")
		desc.WriteString(fmt.Sprintf("Lines: %d\n", len(lines)))
	} else {
		desc.WriteString(fmt.Sprintf("Edit file: %s\n", filePath))
		oldLines := len(strings.Split(oldContent, "\n"))
		newLines := len(strings.Split(newContent, "\n"))
		desc.WriteString(fmt.Sprintf("Lines: %d → %d\n", oldLines, newLines))
		
		// Count changed lines
		changed := 0
		oldArr := strings.Split(oldContent, "\n")
		newArr := strings.Split(newContent, "\n")
		for i := 0; i < len(oldArr) && i < len(newArr); i++ {
			if oldArr[i] != newArr[i] {
				changed++
			}
		}
		desc.WriteString(fmt.Sprintf("Changed lines: ~%d\n", changed))
	}
	
	return desc.String()
}

// CreateFileOptions configures file creation
type CreateFileOptions struct {
	FilePath string
	Content  string
	// Overwrite if file already exists
	Overwrite bool
}

// CreateFile creates a new file
func (f *FileModifier) CreateFile(ctx context.Context, opts CreateFileOptions) (*FileOperation, error) {
	if opts.FilePath == "" {
		return nil, fmt.Errorf("file path is required")
	}

	fullPath := filepath.Join(f.repoPath, opts.FilePath)
	
	// Check if file exists
	if _, err := os.Stat(fullPath); err == nil && !opts.Overwrite {
		return nil, fmt.Errorf("file already exists: %s", opts.FilePath)
	}

	op := &FileOperation{
		Type:         "create",
		FilePath:     opts.FilePath,
		OldContent:   "",
		NewContent:   opts.Content,
		Description:  f.buildEditDescription("create", opts.FilePath, "", opts.Content),
		NeedsConfirm: f.RequireConfirmation,
	}

	if !f.RequireConfirmation {
		if err := f.applyOperation(op); err != nil {
			return nil, err
		}
	}

	return op, nil
}

// DeleteFileOptions configures file deletion
type DeleteFileOptions struct {
	FilePath string
}

// DeleteFile deletes a file
func (f *FileModifier) DeleteFile(ctx context.Context, opts DeleteFileOptions) (*FileOperation, error) {
	if opts.FilePath == "" {
		return nil, fmt.Errorf("file path is required")
	}

	fullPath := filepath.Join(f.repoPath, opts.FilePath)
	
	// Read content for backup
	oldContent := ""
	if data, err := os.ReadFile(fullPath); err == nil {
		oldContent = string(data)
	} else {
		return nil, fmt.Errorf("file does not exist: %s", opts.FilePath)
	}

	op := &FileOperation{
		Type:         "delete",
		FilePath:     opts.FilePath,
		OldContent:   oldContent,
		NewContent:   "",
		Description:  fmt.Sprintf("Delete file: %s (%d bytes)", opts.FilePath, len(oldContent)),
		NeedsConfirm: f.RequireConfirmation,
	}

	if !f.RequireConfirmation {
		if err := f.applyOperation(op); err != nil {
			return nil, err
		}
	}

	return op, nil
}

// ApplyOperation executes a pending operation (after confirmation)
func (f *FileModifier) ApplyOperation(op *FileOperation) error {
	return f.applyOperation(op)
}

func (f *FileModifier) applyOperation(op *FileOperation) error {
	fullPath := filepath.Join(f.repoPath, op.FilePath)

	switch op.Type {
	case "create", "edit":
		// Ensure directory exists
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// Write file
		if err := os.WriteFile(fullPath, []byte(op.NewContent), 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}

	case "delete":
		if err := os.Remove(fullPath); err != nil {
			return fmt.Errorf("failed to delete file: %w", err)
		}

	default:
		return fmt.Errorf("unknown operation type: %s", op.Type)
	}

	return nil
}

// BatchApply applies multiple operations atomically
func (f *FileModifier) BatchApply(ops []*FileOperation) error {
	// Validate all operations first
	for _, op := range ops {
		if op.Type != "create" && op.Type != "edit" && op.Type != "delete" {
			return fmt.Errorf("invalid operation type: %s", op.Type)
		}
	}

	// Apply all operations
	for _, op := range ops {
		if err := f.applyOperation(op); err != nil {
			// TODO: Implement rollback
			return fmt.Errorf("failed to apply operation on %s: %w", op.FilePath, err)
		}
	}

	return nil
}
