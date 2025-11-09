package main

import (
	"context"
	"fmt"
	"log"

	"github.com/go-git/go-git/v6/ai"
)

func main() {
	ctx := context.Background()

	// Open AI repository
	repo, err := ai.Open(".")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== AI Tools Demo ===\n")

	// 1. Git Operations (Read)
	demonstrateGitOperations(ctx, repo)

	// 2. Symbol Analysis
	demonstrateSymbolAnalysis(ctx, repo)

	// 3. Security Scanning
	demonstrateSecurityScanning(ctx, repo)

	// 4. File Modifications (with confirmation)
	demonstrateFileModifications(ctx, repo)

	// 5. Git Write Operations (with confirmation)
	demonstrateGitWriteOps(ctx, repo)
}

func demonstrateGitOperations(ctx context.Context, repo *ai.Repository) {
	fmt.Println("=== 1. Git Operations ===\n")

	// Get recent history
	fmt.Println("📜 Recent Commits:")
	history, err := repo.QuickHistory(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	for i, commit := range history {
		if i >= 5 { // Show only 5
			break
		}
		fmt.Printf("  %s %s - %s\n", 
			commit.ShortHash, 
			commit.AuthorTime.Format("2006-01-02"),
			commit.ShortMsg)
	}
	fmt.Println()

	// Get diff
	fmt.Println("🔍 Recent Changes:")
	diffs, err := repo.GetDiff(ctx, ai.DiffOptions{
		MaxLines: 50,
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else if len(diffs) > 0 {
		for i, diff := range diffs {
			if i >= 3 {
				break
			}
			fmt.Printf("  %s (+%d -%d)\n", diff.FilePath, diff.Additions, diff.Deletions)
		}
	} else {
		fmt.Println("  No uncommitted changes")
	}
	fmt.Println()

	// Get blame for a file
	fmt.Println("👤 File Authorship (ai/config.go):")
	blameLines, err := repo.GetBlame(ctx, ai.BlameOptions{
		FilePath: "ai/config.go",
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		// Show first 5 lines
		for i, line := range blameLines {
			if i >= 5 {
				break
			}
			fmt.Printf("  L%d %s %s: %s\n",
				line.LineNumber,
				line.CommitHash,
				line.Author,
				truncate(line.Line, 60))
		}
	}
	fmt.Println()
}

func demonstrateSymbolAnalysis(ctx context.Context, repo *ai.Repository) {
	fmt.Println("=== 2. Symbol Analysis ===\n")

	// Get symbols from a specific file
	fmt.Println("🔧 Symbols in ai/repo.go:")
	symbols, err := repo.GetSymbols(ctx, ai.GetSymbolsOptions{
		FilePath:    "ai/repo.go",
		SymbolTypes: []ai.SymbolType{ai.SymbolTypeFunction, ai.SymbolTypeMethod, ai.SymbolTypeStruct},
		IncludeDocs: false,
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		for i, sym := range symbols {
			if i >= 10 { // Show only 10
				break
			}
			fmt.Printf("  %s %s (line %d)\n", sym.Type, sym.Name, sym.Line)
		}
	}
	fmt.Println()

	// Find references to a symbol
	fmt.Println("🔎 References to 'Repository':")
	refs, err := repo.FindReferences(ctx, ai.FindReferencesOptions{
		SymbolName: "Repository",
		FilePath:   "ai/repo.go", // Limit to one file
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		for i, ref := range refs {
			if i >= 5 { // Show only 5
				break
			}
			fmt.Printf("  %s:%d - %s\n", ref.FilePath, ref.Line, truncate(ref.Context, 60))
		}
	}
	fmt.Println()

	// Get definition
	fmt.Println("📍 Definition of 'Repository':")
	def, err := repo.GetDefinition(ctx, "Repository")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("  Found in %s at line %d\n", def.FilePath, def.Line)
		fmt.Printf("  Type: %s\n", def.Type)
	}
	fmt.Println()
}

func demonstrateSecurityScanning(ctx context.Context, repo *ai.Repository) {
	fmt.Println("=== 3. Security Scanning ===\n")

	// Quick security scan
	fmt.Println("🔐 Security Scan (high severity only):")
	scanResult, err := repo.QuickSecurityScan(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("  Files scanned: %d\n", scanResult.FilesScanned)
	fmt.Printf("  High severity issues: %d\n", scanResult.HighSeverity)
	
	if len(scanResult.Findings) > 0 {
		fmt.Println("\n  ⚠️  Findings:")
		for i, finding := range scanResult.Findings {
			if i >= 3 { // Show only 3
				break
			}
			fmt.Printf("    %s in %s:%d\n", finding.Type, finding.FilePath, finding.Line)
			fmt.Printf("    Redacted: %s\n", finding.Redacted)
		}
	} else {
		fmt.Println("  ✅ No high-severity secrets found")
	}
	fmt.Println()

	// Analyze dependencies
	fmt.Println("📦 Dependencies:")
	deps, err := repo.AnalyzeDependencies(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("  Total dependencies: %d\n", len(deps))
		for i, dep := range deps {
			if i >= 5 { // Show only 5
				break
			}
			fmt.Printf("  - %s %s (%s)\n", dep.Name, dep.Version, dep.Source)
		}
	}
	fmt.Println()
}

func demonstrateFileModifications(ctx context.Context, repo *ai.Repository) {
	fmt.Println("=== 4. File Modifications ===\n")

	// Example: Edit a file (returns operation that needs confirmation)
	fmt.Println("✏️  Edit File Example:")
	editOp, err := repo.EditFile(ctx, ai.EditFileOptions{
		FilePath: "ai/example_temp.go",
		Content: `package ai

// Example temporary file
// This demonstrates file modification

func ExampleFunction() string {
	return "Hello from AI tools!"
}
`,
		CreateIfMissing: true,
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("  Operation: %s\n", editOp.Type)
		fmt.Printf("  File: %s\n", editOp.FilePath)
		fmt.Printf("  %s\n", editOp.Description)
		fmt.Println("  ⚠️  Needs confirmation: true")
		fmt.Println("  (Not applying - demo mode)")
	}
	fmt.Println()

	// Example: Create a file
	fmt.Println("📄 Create File Example:")
	createOp, err := repo.CreateFile(ctx, ai.CreateFileOptions{
		FilePath: "ai/another_example.go",
		Content:  "package ai\n\n// Another example file\n",
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("  %s\n", createOp.Description)
		fmt.Println("  ⚠️  Needs confirmation: true")
		fmt.Println("  (Not applying - demo mode)")
	}
	fmt.Println()
}

func demonstrateGitWriteOps(ctx context.Context, repo *ai.Repository) {
	fmt.Println("=== 5. Git Write Operations ===\n")

	// Example: Create a commit (with AI-generated message)
	fmt.Println("💾 Commit Example:")
	commitResult, err := repo.CommitChanges(ctx, ai.CommitOptions{
		Message: "", // Empty = AI generates message
		Files:   []string{}, // Empty = all modified files
	})
	if err != nil {
		if err.Error() == "no changes to commit" {
			fmt.Println("  ℹ️  No changes to commit")
		} else {
			fmt.Printf("Error: %v\n", err)
		}
	} else {
		fmt.Printf("  Generated message: %s\n", commitResult.Message)
		fmt.Printf("  Files to commit: %d\n", len(commitResult.FilesChanged))
		fmt.Println("  ⚠️  Needs confirmation: true")
		fmt.Println("  (Not applying - demo mode)")
	}
	fmt.Println()

	// Example: Create a branch
	fmt.Println("🌿 Branch Creation Example:")
	branchResult, err := repo.CreateBranch(ctx, ai.BranchOptions{
		Name:     "feature/ai-tools-demo",
		Checkout: false,
		Force:    false,
	})
	if err != nil {
		if err.Error() == "branch feature/ai-tools-demo already exists" {
			fmt.Println("  ℹ️  Branch already exists")
		} else {
			fmt.Printf("Error: %v\n", err)
		}
	} else {
		fmt.Printf("  %s\n", branchResult.Description)
		fmt.Println("  ⚠️  Needs confirmation: true")
		fmt.Println("  (Not applying - demo mode)")
	}
	fmt.Println()

	// Example: Show workflow with confirmation
	fmt.Println("🔄 Complete Workflow Example:")
	fmt.Println(`
  User: "Add a new feature to ai/search.go"
  
  LLM Agent Workflow:
  1. search_codebase("search") → Find existing code
  2. get_symbols("ai/search.go") → Understand structure
  3. edit_file() → Propose changes
  4. Show diff to user → "Here's what I'll change..."
  5. Wait for confirmation → User says "yes"
  6. apply_file_operation() → Apply changes
  7. commit_changes() → Propose commit
  8. Show commit message → "feat: add X to search"
  9. Wait for confirmation → User says "yes"
  10. apply_commit() → Execute commit
  11. Done! → "Committed as abc123"
	`)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
