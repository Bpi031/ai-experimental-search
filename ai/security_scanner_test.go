package ai

import (
	"context"
	"strings"
	"testing"
)

func TestSecurityScanner_ScanForSecrets(t *testing.T) {
	repo := mustCreateTempRepo(t)
	
	// Create files with various secrets
	testFiles := map[string]string{
		"aws.txt": `
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
`,
		"github.txt": `
github_pat_11ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890
GITHUB_TOKEN=ghp_1234567890abcdefghijklmnopqrstuvwxyz
`,
		"api.txt": `
API_KEY=sk-1234567890abcdefghijklmnopqrstuvwxyz
api-key: AIzaSyD1234567890abcdefghijklmnopqrst
`,
		"clean.txt": `
This file has no secrets.
Just some normal content.
`,
	}
	
	for path, content := range testFiles {
		mustWrite(t, repo, path, content)
	}
	mustCommitAll(t, repo, "Add test files")
	
	scanner := NewSecurityScanner(repo)
	
	// Test: Scan for secrets
	result, err := scanner.ScanForSecrets(context.Background(), ScanOptions{})
	if err != nil {
		t.Fatalf("ScanForSecrets failed: %v", err)
	}
	
	if len(result.Findings) == 0 {
		t.Error("Expected to find secrets, got none")
	}
	
	// Verify findings structure
	for _, finding := range result.Findings {
		if finding.Type == "" {
			t.Error("Finding type is empty")
		}
		if finding.FilePath == "" {
			t.Error("Finding file path is empty")
		}
		if finding.Line == 0 {
			t.Error("Finding line number is zero")
		}
		if finding.Severity == "" {
			t.Error("Finding severity is empty")
		}
		
		// Verify redaction
		if strings.Contains(finding.Match, "EXAMPLE") || strings.Contains(finding.Match, "1234567890") {
			if !strings.Contains(finding.Redacted, "****") {
				t.Errorf("Secret not properly redacted: %s", finding.Redacted)
			}
		}
	}
}

func TestSecurityScanner_ScanForSecrets_SpecificFile(t *testing.T) {
	repo := mustCreateTempRepo(t)
	
	// Create multiple files
	mustWrite(t, repo, "file1.txt", "API_KEY=sk-1234567890abcdef")
	mustWrite(t, repo, "file2.txt", "AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE")
	mustCommitAll(t, repo, "Add files")
	
	scanner := NewSecurityScanner(repo)
	
	// Test: Scan specific file
	result, err := scanner.ScanForSecrets(context.Background(), ScanOptions{
		FilePath: "file1.txt",
	})
	if err != nil {
		t.Fatalf("ScanForSecrets with FilePath failed: %v", err)
	}
	
	// Should only find secrets in file1
	for _, finding := range result.Findings {
		if !strings.HasSuffix(finding.FilePath, "file1.txt") {
			t.Errorf("Expected only file1.txt, got %s", finding.FilePath)
		}
	}
}

func TestSecurityScanner_AnalyzeDependencies(t *testing.T) {
	repo := mustCreateTempRepo(t)
	
	// Create go.mod
	goModContent := `module example.com/test

go 1.21

require (
	github.com/go-git/go-git/v6 v6.0.0
	golang.org/x/crypto v0.14.0
)
`
	mustWrite(t, repo, "go.mod", goModContent)
	mustCommitAll(t, repo, "Add go.mod")
	
	scanner := NewSecurityScanner(repo)
	
	// Test: Analyze dependencies
	deps, err := scanner.AnalyzeDependencies(context.Background())
	if err != nil {
		t.Fatalf("AnalyzeDependencies failed: %v", err)
	}
	
	if len(deps) == 0 {
		t.Error("Expected to find dependencies, got none")
	}
	
	// Verify we found go-git
	foundGoGit := false
	for _, dep := range deps {
		if strings.Contains(dep.Name, "go-git") {
			foundGoGit = true
			if dep.Version == "" {
				t.Error("Dependency version is empty")
			}
			if dep.Type != "go" {
				t.Errorf("Expected type 'go', got '%s'", dep.Type)
			}
			if dep.Source != "go.mod" {
				t.Errorf("Expected source 'go.mod', got '%s'", dep.Source)
			}
		}
	}
	
	if !foundGoGit {
		t.Error("Did not find go-git dependency")
	}
}

func TestSecurityScanner_AnalyzeDependencies_PackageJSON(t *testing.T) {
	repo := mustCreateTempRepo(t)
	
	// Create package.json
	pkgJSONContent := `{
  "name": "test-project",
  "version": "1.0.0",
  "dependencies": {
    "express": "^4.18.0",
    "lodash": "4.17.21"
  },
  "devDependencies": {
    "jest": "^29.0.0"
  }
}
`
	mustWrite(t, repo, "package.json", pkgJSONContent)
	mustCommitAll(t, repo, "Add package.json")
	
	scanner := NewSecurityScanner(repo)
	
	// Test: Analyze npm dependencies
	deps, err := scanner.AnalyzeDependencies(context.Background())
	if err != nil {
		t.Fatalf("AnalyzeDependencies for npm failed: %v", err)
	}
	
	if len(deps) == 0 {
		t.Error("Expected to find npm dependencies, got none")
	}
	
	// Verify we found express
	foundExpress := false
	for _, dep := range deps {
		if dep.Name == "express" {
			foundExpress = true
			if dep.Type != "npm" {
				t.Errorf("Expected type 'npm', got '%s'", dep.Type)
			}
			if dep.Source != "package.json" {
				t.Errorf("Expected source 'package.json', got '%s'", dep.Source)
			}
		}
	}
	
	if !foundExpress {
		t.Error("Did not find express dependency")
	}
}

func TestSecurityScanner_NoSecretsFound(t *testing.T) {
	repo := mustCreateTempRepo(t)
	
	// Create clean file
	mustWrite(t, repo, "clean.txt", "This is just normal text\nNo secrets here\n")
	mustCommitAll(t, repo, "Add clean file")
	
	scanner := NewSecurityScanner(repo)
	
	// Test: Should find no secrets
	result, err := scanner.ScanForSecrets(context.Background(), ScanOptions{})
	if err != nil {
		t.Fatalf("ScanForSecrets failed: %v", err)
	}
	
	if len(result.Findings) != 0 {
		t.Errorf("Expected no findings for clean file, got %d", len(result.Findings))
	}
}

func TestSecurityScanner_NoDependenciesFound(t *testing.T) {
	repo := mustCreateTempRepo(t)
	
	// No dependency files - just create empty commit
	mustWrite(t, repo, "README.md", "# Test")
	mustCommitAll(t, repo, "Initial commit")
	
	scanner := NewSecurityScanner(repo)
	
	// Test: Should return empty list
	deps, err := scanner.AnalyzeDependencies(context.Background())
	if err != nil {
		t.Fatalf("AnalyzeDependencies failed: %v", err)
	}
	
	if len(deps) != 0 {
		t.Errorf("Expected no dependencies, got %d", len(deps))
	}
}
