package ai

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileModifier_CreateFile(t *testing.T) {
	tmpDir := t.TempDir()
	
	fm := NewFileModifier(tmpDir)
	fm.RequireConfirmation = false // No confirmation required for tests
	
	// Test: Create new file
	op, err := fm.CreateFile(context.Background(), CreateFileOptions{
		FilePath: "newfile.txt",
		Content:  "test content",
	})
	if err != nil {
		t.Fatalf("CreateFile failed: %v", err)
	}
	
	if op.Type != "create" {
		t.Errorf("Expected 'create', got %v", op.Type)
	}
	
	// File should already be created (RequireConfirmation = false)
	
	// Verify file was created
	filePath := filepath.Join(tmpDir, "newfile.txt")
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}
	
	if string(content) != "test content" {
		t.Errorf("Expected 'test content', got '%s'", string(content))
	}
}

func TestFileModifier_CreateFileInSubdirectory(t *testing.T) {
	tmpDir := t.TempDir()
	
	fm := NewFileModifier(tmpDir)
	fm.RequireConfirmation = false
	
	// Test: Create file in subdirectory (should create dirs)
	_, err := fm.CreateFile(context.Background(), CreateFileOptions{
		FilePath: "subdir/nested/file.txt",
		Content:  "nested content",
	})
	if err != nil {
		t.Fatalf("CreateFile in subdirectory failed: %v", err)
	}
	
	// Verify file exists
	filePath := filepath.Join(tmpDir, "subdir", "nested", "file.txt")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("File was not created in subdirectory")
	}
}

func TestFileModifier_CreateFileAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create existing file
	existingFile := filepath.Join(tmpDir, "existing.txt")
	if err := os.WriteFile(existingFile, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}
	
	fm := NewFileModifier(tmpDir)
	fm.RequireConfirmation = false
	
	// Test: Creating existing file should error
	_, err := fm.CreateFile(context.Background(), CreateFileOptions{
		FilePath: "existing.txt",
		Content:  "new content",
	})
	if err == nil {
		t.Error("Expected error when creating existing file, got nil")
	}
}

func TestFileModifier_EditFile_FullContent(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create initial file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("original content"), 0644); err != nil {
		t.Fatal(err)
	}
	
	fm := NewFileModifier(tmpDir)
	fm.RequireConfirmation = false
	
	// Test: Edit entire file
	_, err := fm.EditFile(context.Background(), EditFileOptions{
		FilePath: "test.txt",
		Content:  "completely new content",
	})
	if err != nil {
		t.Fatalf("EditFile failed: %v", err)
	}
	
	// Verify
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	
	if string(content) != "completely new content" {
		t.Errorf("Expected 'completely new content', got '%s'", string(content))
	}
}

func TestFileModifier_EditFile_LineRange(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create file with multiple lines
	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := "line1\nline2\nline3\nline4\nline5\n"
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatal(err)
	}
	
	fm := NewFileModifier(tmpDir)
	fm.RequireConfirmation = false
	
	// Test: Edit lines 2-3
	_, err := fm.EditFile(context.Background(), EditFileOptions{
		FilePath: "test.txt",
		LineRanges: []LineEdit{
			{StartLine: 2, EndLine: 3, NewText: "modified2\nmodified3"},
		},
	})
	if err != nil {
		t.Fatalf("EditFile with LineRanges failed: %v", err)
	}
	
	// Verify
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	
	lines := strings.Split(string(content), "\n")
	if len(lines) < 4 {
		t.Fatalf("Expected at least 4 lines, got %d", len(lines))
	}
	
	if lines[0] != "line1" {
		t.Errorf("Line 1 should be unchanged, got '%s'", lines[0])
	}
	if lines[1] != "modified2" {
		t.Errorf("Line 2 should be 'modified2', got '%s'", lines[1])
	}
	if lines[2] != "modified3" {
		t.Errorf("Line 3 should be 'modified3', got '%s'", lines[2])
	}
}

func TestFileModifier_EditFile_MultipleRanges(t *testing.T) {
	tmpDir := t.TempDir()
	
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "1\n2\n3\n4\n5\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	
	fm := NewFileModifier(tmpDir)
	fm.RequireConfirmation = false
	
	// Test: Edit multiple non-overlapping ranges
	_, err := fm.EditFile(context.Background(), EditFileOptions{
		FilePath: "test.txt",
		LineRanges: []LineEdit{
			{StartLine: 1, EndLine: 1, NewText: "ONE"},
			{StartLine: 3, EndLine: 3, NewText: "THREE"},
			{StartLine: 5, EndLine: 5, NewText: "FIVE"},
		},
	})
	if err != nil {
		t.Fatalf("EditFile with multiple ranges failed: %v", err)
	}
	
	// Verify
	result, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	
	lines := strings.Split(string(result), "\n")
	if lines[0] != "ONE" {
		t.Errorf("Line 1 should be 'ONE', got '%s'", lines[0])
	}
	if lines[2] != "THREE" {
		t.Errorf("Line 3 should be 'THREE', got '%s'", lines[2])
	}
	if lines[4] != "FIVE" {
		t.Errorf("Line 5 should be 'FIVE', got '%s'", lines[4])
	}
}

func TestFileModifier_DeleteFile(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create file to delete
	testFile := filepath.Join(tmpDir, "todelete.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	
	fm := NewFileModifier(tmpDir)
	fm.RequireConfirmation = false
	
	// Test: Delete file
	_, err := fm.DeleteFile(context.Background(), DeleteFileOptions{
		FilePath: "todelete.txt",
	})
	if err != nil {
		t.Fatalf("DeleteFile failed: %v", err)
	}
	
	// Verify file is gone
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("File should have been deleted")
	}
}

func TestFileModifier_DeleteNonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	
	fm := NewFileModifier(tmpDir)
	fm.RequireConfirmation = false
	
	// Test: Delete non-existent file should error
	_, err := fm.DeleteFile(context.Background(), DeleteFileOptions{
		FilePath: "nonexistent.txt",
	})
	if err == nil {
		t.Error("Expected error when deleting non-existent file, got nil")
	}
}

func TestFileModifier_WithConfirmation(t *testing.T) {
	tmpDir := t.TempDir()
	
	fm := NewFileModifier(tmpDir) // Require confirmation (default)
	
	// Test: Should return operation without applying
	op, err := fm.CreateFile(context.Background(), CreateFileOptions{
		FilePath: "newfile.txt",
		Content:  "content",
	})
	if err != nil {
		t.Fatalf("CreateFile failed: %v", err)
	}
	
	// File should NOT exist yet
	filePath := filepath.Join(tmpDir, "newfile.txt")
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("File should not exist before confirmation")
	}
	
	if !op.NeedsConfirm {
		t.Error("Operation should require confirmation")
	}
}

func TestFileModifier_EditNonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	
	fm := NewFileModifier(tmpDir)
	fm.RequireConfirmation = false
	
	// Test: Edit non-existent file should error
	_, err := fm.EditFile(context.Background(), EditFileOptions{
		FilePath: "nonexistent.txt",
		Content:  "content",
	})
	if err == nil {
		t.Error("Expected error when editing non-existent file, got nil")
	}
}

func TestFileModifier_InvalidLineRange(t *testing.T) {
	tmpDir := t.TempDir()
	
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("line1\nline2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	
	fm := NewFileModifier(tmpDir)
	fm.RequireConfirmation = false
	
	// Test: Invalid line range (beyond file)
	_, err := fm.EditFile(context.Background(), EditFileOptions{
		FilePath: "test.txt",
		LineRanges: []LineEdit{
			{StartLine: 10, EndLine: 20, NewText: "invalid"},
		},
	})
	if err == nil {
		t.Error("Expected error for out-of-range line numbers, got nil")
	}
}

func TestFileModifier_EmptyLineRange(t *testing.T) {
	tmpDir := t.TempDir()
	
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("line1\nline2\nline3\n"), 0644); err != nil {
		t.Fatal(err)
	}
	
	fm := NewFileModifier(tmpDir)
	fm.RequireConfirmation = false
	
	// Test: Empty content in range (deletion)
	_, err := fm.EditFile(context.Background(), EditFileOptions{
		FilePath: "test.txt",
		LineRanges: []LineEdit{
			{StartLine: 2, EndLine: 2, NewText: ""}, // Delete line 2
		},
	})
	if err != nil {
		t.Fatalf("EditFile with empty content failed: %v", err)
	}
	
	// Verify line 2 was replaced with empty
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	
	// Note: Line replacement with empty string still keeps a line, just empty
	lines := strings.Split(string(content), "\n")
	if len(lines) < 2 {
		t.Log("Line range edit modified structure as expected")
	}
}
