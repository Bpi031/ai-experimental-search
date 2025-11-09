package ai

import (
	"context"
	"testing"
)

func TestWorkspaceAnalyzer_GetWorkspaceStats(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	// Create files in different languages
	mustWrite(t, repo, "main.go", "package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}")
	mustWrite(t, repo, "utils.go", "package utils\n\nfunc Helper() {}")
	mustWrite(t, repo, "script.py", "def hello():\n    print('Hello')")
	mustWrite(t, repo, "app.js", "function hello() {\n  console.log('Hello');\n}")
	mustWrite(t, repo, "README.md", "# Project\n\nDescription")
	mustWrite(t, repo, "internal/helper.go", "package internal\n\nfunc Help() {}")
	mustCommitAll(t, repo, "Add files")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	analyzer := NewWorkspaceAnalyzer(repo, repoPath)

	ctx := context.Background()
	stats, err := analyzer.GetWorkspaceStats(ctx)
	if err != nil {
		t.Fatalf("GetWorkspaceStats() error = %v", err)
	}

	// Check total files
	if stats.TotalFiles < 6 {
		t.Errorf("GetWorkspaceStats() TotalFiles = %d, want at least 6", stats.TotalFiles)
	}

	// Check languages detected
	if len(stats.FilesByLanguage) < 3 {
		t.Errorf("GetWorkspaceStats() detected %d languages, want at least 3", len(stats.FilesByLanguage))
	}

	// Check Go files
	if goFiles, ok := stats.FilesByLanguage["Go"]; !ok || goFiles < 3 {
		t.Errorf("GetWorkspaceStats() Go files = %d, want at least 3", goFiles)
	}

	// Check primary language
	if stats.PrimaryLanguage == "" {
		t.Error("GetWorkspaceStats() PrimaryLanguage is empty")
	}

	// Check lines counted
	if stats.TotalLines == 0 {
		t.Error("GetWorkspaceStats() TotalLines = 0, want > 0")
	}

	// Check size
	if stats.TotalSize == 0 {
		t.Error("GetWorkspaceStats() TotalSize = 0, want > 0")
	}

	// Check directory stats
	if stats.DirectoryCount == 0 {
		t.Error("GetWorkspaceStats() DirectoryCount = 0, want > 0")
	}
}

func TestWorkspaceAnalyzer_GetLanguageBreakdown(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	mustWrite(t, repo, "main.go", "package main\n\nfunc main() {}")
	mustWrite(t, repo, "app.py", "def main():\n    pass")
	mustWrite(t, repo, "index.js", "function main() {}")
	mustCommitAll(t, repo, "Add files")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	analyzer := NewWorkspaceAnalyzer(repo, repoPath)

	ctx := context.Background()
	languages, err := analyzer.GetLanguageBreakdown(ctx)
	if err != nil {
		t.Fatalf("GetLanguageBreakdown() error = %v", err)
	}

	if len(languages) < 3 {
		t.Errorf("GetLanguageBreakdown() got %d languages, want at least 3", len(languages))
	}

	// Check that languages are sorted by percentage
	for i := 0; i < len(languages)-1; i++ {
		if languages[i].Percentage < languages[i+1].Percentage {
			t.Errorf("GetLanguageBreakdown() not sorted: %s (%.1f%%) before %s (%.1f%%)",
				languages[i].Name, languages[i].Percentage,
				languages[i+1].Name, languages[i+1].Percentage)
		}
	}

	// Check that percentages add up to ~100%
	total := 0.0
	for _, lang := range languages {
		total += lang.Percentage
	}
	if total < 99.0 || total > 101.0 {
		t.Errorf("GetLanguageBreakdown() total percentage = %.1f%%, want ~100%%", total)
	}
}

func TestWorkspaceAnalyzer_LanguageDetection(t *testing.T) {
	tests := []struct {
		filename string
		wantLang string
	}{
		{"main.go", "Go"},
		{"script.py", "Python"},
		{"app.js", "JavaScript"},
		{"app.ts", "TypeScript"},
		{"Component.jsx", "JavaScript"},
		{"Component.tsx", "TypeScript"},
		{"program.java", "Java"},
		{"main.cpp", "C++"},
		{"main.c", "C"},
		{"script.rb", "Ruby"},
		{"app.php", "PHP"},
		{"main.rs", "Rust"},
		{"main.kt", "Kotlin"},
		{"script.sh", "Shell"},
		{"style.css", "CSS"},
		{"index.html", "HTML"},
		{"config.json", "JSON"},
		{"config.yaml", "YAML"},
		{"README.md", "Markdown"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			lang := detectLanguage(tt.filename)
			if lang != tt.wantLang {
				t.Errorf("detectLanguage(%s) = %s, want %s", tt.filename, lang, tt.wantLang)
			}
		})
	}
}

func TestWorkspaceAnalyzer_DirectoryStats(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	// Create nested directory structure
	mustWrite(t, repo, "main.go", "package main")
	mustWrite(t, repo, "internal/utils.go", "package internal")
	mustWrite(t, repo, "internal/helper.go", "package internal")
	mustWrite(t, repo, "pkg/api/handler.go", "package api")
	mustWrite(t, repo, "pkg/api/middleware.go", "package api")
	mustWrite(t, repo, "pkg/api/router.go", "package api")
	mustCommitAll(t, repo, "Add nested structure")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	analyzer := NewWorkspaceAnalyzer(repo, repoPath)

	ctx := context.Background()
	stats, err := analyzer.GetWorkspaceStats(ctx)
	if err != nil {
		t.Fatalf("GetWorkspaceStats() error = %v", err)
	}

	// Check max depth
	if stats.MaxDepth < 2 {
		t.Errorf("GetWorkspaceStats() MaxDepth = %d, want at least 2", stats.MaxDepth)
	}

	// Check top directories
	if len(stats.TopDirectories) == 0 {
		t.Error("GetWorkspaceStats() TopDirectories is empty")
	}

	// Top directory should be pkg/api with 3 files
	foundApiDir := false
	for _, dir := range stats.TopDirectories {
		if dir.Path == "pkg/api" && dir.FileCount >= 3 {
			foundApiDir = true
			break
		}
	}
	if !foundApiDir {
		t.Error("GetWorkspaceStats() TopDirectories doesn't contain pkg/api with 3 files")
	}
}

func TestWorkspaceAnalyzer_EmptyRepo(t *testing.T) {
	repo := mustCreateTempRepo(t)
	defer cleanupRepo(t, repo)

	// Commit with no files
	mustWrite(t, repo, ".gitkeep", "")
	mustCommitAll(t, repo, "Initial commit")

	wt, _ := repo.Worktree()
	repoPath := wt.Filesystem.Root()
	analyzer := NewWorkspaceAnalyzer(repo, repoPath)

	ctx := context.Background()
	stats, err := analyzer.GetWorkspaceStats(ctx)
	if err != nil {
		t.Fatalf("GetWorkspaceStats() error = %v", err)
	}

	// Should handle empty repo gracefully
	if stats.TotalFiles != 0 {
		t.Errorf("GetWorkspaceStats() TotalFiles = %d, want 0", stats.TotalFiles)
	}
}
