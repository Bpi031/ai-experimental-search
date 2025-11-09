package ai

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v6"
)

// SecurityScanner provides security scanning capabilities
type SecurityScanner struct {
	repo *git.Repository
}

// NewSecurityScanner creates a new SecurityScanner instance
func NewSecurityScanner(repo *git.Repository) *SecurityScanner {
	return &SecurityScanner{repo: repo}
}

// SecretPattern defines a pattern for detecting secrets
type SecretPattern struct {
	Name        string
	Pattern     *regexp.Regexp
	Description string
	Severity    string // "high", "medium", "low"
}

// Common secret patterns (from Amazon CodeWhisperer and GitHub Secret Scanning)
var defaultSecretPatterns = []SecretPattern{
	{
		Name:        "AWS Access Key",
		Pattern:     regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
		Description: "AWS Access Key ID",
		Severity:    "high",
	},
	{
		Name:        "AWS Secret Key",
		Pattern:     regexp.MustCompile(`(?i)aws(.{0,20})?(?-i)['\"][0-9a-zA-Z/+]{40}['\"]`),
		Description: "AWS Secret Access Key",
		Severity:    "high",
	},
	{
		Name:        "GitHub Token",
		Pattern:     regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`),
		Description: "GitHub Personal Access Token",
		Severity:    "high",
	},
	{
		Name:        "GitHub OAuth Token",
		Pattern:     regexp.MustCompile(`gho_[a-zA-Z0-9]{36}`),
		Description: "GitHub OAuth Access Token",
		Severity:    "high",
	},
	{
		Name:        "Generic API Key",
		Pattern:     regexp.MustCompile(`(?i)(api[_-]?key|apikey|api[_-]?secret)['"]?\s*[:=]\s*['"]?[a-zA-Z0-9_\-]{32,}`),
		Description: "Generic API Key Pattern",
		Severity:    "medium",
	},
	{
		Name:        "Generic Secret",
		Pattern:     regexp.MustCompile(`(?i)(secret|password|passwd|pwd)['"]?\s*[:=]\s*['"][^'"]{8,}['"]`),
		Description: "Generic Secret or Password",
		Severity:    "medium",
	},
	{
		Name:        "Private Key",
		Pattern:     regexp.MustCompile(`-----BEGIN (RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`),
		Description: "Private Key",
		Severity:    "high",
	},
	{
		Name:        "JWT Token",
		Pattern:     regexp.MustCompile(`eyJ[a-zA-Z0-9_-]+\.eyJ[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+`),
		Description: "JSON Web Token",
		Severity:    "medium",
	},
	{
		Name:        "Google API Key",
		Pattern:     regexp.MustCompile(`AIza[0-9A-Za-z\\-_]{35}`),
		Description: "Google API Key",
		Severity:    "high",
	},
	{
		Name:        "Slack Token",
		Pattern:     regexp.MustCompile(`xox[baprs]-[0-9]{10,13}-[0-9]{10,13}-[a-zA-Z0-9]{24,32}`),
		Description: "Slack Token",
		Severity:    "high",
	},
	{
		Name:        "Database Connection String",
		Pattern:     regexp.MustCompile(`(?i)(mongodb|mysql|postgresql|postgres)://[^:]+:[^@]+@[^/]+`),
		Description: "Database Connection String with Credentials",
		Severity:    "high",
	},
}

// SecretFinding represents a detected secret
type SecretFinding struct {
	Type        string
	Description string
	Severity    string
	FilePath    string
	Line        int
	Column      int
	Match       string
	Redacted    string
	Context     string
}

// ScanOptions configures security scanning
type ScanOptions struct {
	// FilePath optionally limits scan to specific file
	FilePath string
	// CustomPatterns adds additional patterns to scan for
	CustomPatterns []SecretPattern
	// ExcludePatterns excludes files matching patterns
	ExcludePatterns []string
}

// ScanResult represents scan results
type ScanResult struct {
	Findings      []SecretFinding
	FilesScanned  int
	SecretsFound  int
	HighSeverity  int
	MediumSeverity int
	LowSeverity   int
}

// ScanForSecrets scans repository for secrets and credentials
func (s *SecurityScanner) ScanForSecrets(ctx context.Context, opts ScanOptions) (*ScanResult, error) {
	wt, err := s.repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	// Build patterns list
	patterns := append([]SecretPattern{}, defaultSecretPatterns...)
	if len(opts.CustomPatterns) > 0 {
		patterns = append(patterns, opts.CustomPatterns...)
	}

	// Build exclude patterns
	excludeRegexps := []*regexp.Regexp{
		regexp.MustCompile(`/\.git/`),
		regexp.MustCompile(`/node_modules/`),
		regexp.MustCompile(`/vendor/`),
		regexp.MustCompile(`\.min\.js$`),
		regexp.MustCompile(`\.min\.css$`),
	}
	for _, pattern := range opts.ExcludePatterns {
		if re, err := regexp.Compile(pattern); err == nil {
			excludeRegexps = append(excludeRegexps, re)
		}
	}

	result := &ScanResult{
		Findings: []SecretFinding{},
	}

	// Scan files
	var files []string
	if opts.FilePath != "" {
		files = []string{filepath.Join(wt.Filesystem.Root(), opts.FilePath)}
	} else {
		// Scan all files in repository
		err := filepath.Walk(wt.Filesystem.Root(), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath, _ := filepath.Rel(wt.Filesystem.Root(), path)
			
			// Skip excluded patterns
			for _, exclude := range excludeRegexps {
				if exclude.MatchString(relPath) {
					if info.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}

			if !info.IsDir() && info.Size() < 1*1024*1024 { // Skip files > 1MB
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to walk directory: %w", err)
		}
	}

	// Scan each file
	for _, file := range files {
		relPath, _ := filepath.Rel(wt.Filesystem.Root(), file)
		
		findings, err := s.scanFile(file, relPath, patterns)
		if err != nil {
			continue // Skip files with errors
		}

		result.Findings = append(result.Findings, findings...)
		result.FilesScanned++
	}

	// Count severity levels
	for _, finding := range result.Findings {
		result.SecretsFound++
		switch finding.Severity {
		case "high":
			result.HighSeverity++
		case "medium":
			result.MediumSeverity++
		case "low":
			result.LowSeverity++
		}
	}

	return result, nil
}

func (s *SecurityScanner) scanFile(path, relPath string, patterns []SecretPattern) ([]SecretFinding, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Skip binary files
	if isBinary(content) {
		return nil, nil
	}

	lines := strings.Split(string(content), "\n")
	var findings []SecretFinding

	for i, line := range lines {
		for _, pattern := range patterns {
			matches := pattern.Pattern.FindAllStringIndex(line, -1)
			for _, match := range matches {
				matchStr := line[match[0]:match[1]]
				redacted := redactSecret(matchStr)

				// Get context (surrounding lines)
				context := s.getContext(lines, i, 2)

				findings = append(findings, SecretFinding{
					Type:        pattern.Name,
					Description: pattern.Description,
					Severity:    pattern.Severity,
					FilePath:    relPath,
					Line:        i + 1,
					Column:      match[0] + 1,
					Match:       matchStr,
					Redacted:    redacted,
					Context:     context,
				})
			}
		}
	}

	return findings, nil
}

func (s *SecurityScanner) getContext(lines []string, lineIdx, contextLines int) string {
	start := lineIdx - contextLines
	if start < 0 {
		start = 0
	}
	end := lineIdx + contextLines + 1
	if end > len(lines) {
		end = len(lines)
	}

	var contextLines_arr []string
	for i := start; i < end; i++ {
		prefix := "  "
		if i == lineIdx {
			prefix = "> "
		}
		contextLines_arr = append(contextLines_arr, fmt.Sprintf("%s%d: %s", prefix, i+1, lines[i]))
	}

	return strings.Join(contextLines_arr, "\n")
}

func isBinary(content []byte) bool {
	// Simple heuristic: if file contains null bytes, consider it binary
	for _, b := range content {
		if b == 0 {
			return true
		}
	}
	return false
}

func redactSecret(secret string) string {
	if len(secret) <= 8 {
		return "[REDACTED]"
	}
	// Keep first and last 4 characters for context
	return secret[:4] + strings.Repeat("*", len(secret)-8) + secret[len(secret)-4:]
}

// DependencyInfo represents a code dependency
type DependencyInfo struct {
	Name    string
	Version string
	Type    string // "direct", "indirect"
	Source  string // File where declared
}

// AnalyzeDependencies analyzes project dependencies
func (s *SecurityScanner) AnalyzeDependencies(ctx context.Context) ([]DependencyInfo, error) {
	wt, err := s.repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	var dependencies []DependencyInfo

	// Check for go.mod (Go projects)
	goModPath := filepath.Join(wt.Filesystem.Root(), "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		deps, err := s.parseGoMod(goModPath)
		if err == nil {
			dependencies = append(dependencies, deps...)
		}
	}

	// Check for package.json (Node.js projects)
	packageJSONPath := filepath.Join(wt.Filesystem.Root(), "package.json")
	if _, err := os.Stat(packageJSONPath); err == nil {
		deps, err := s.parsePackageJSON(packageJSONPath)
		if err == nil {
			dependencies = append(dependencies, deps...)
		}
	}

	// Check for requirements.txt (Python projects)
	requirementsPath := filepath.Join(wt.Filesystem.Root(), "requirements.txt")
	if _, err := os.Stat(requirementsPath); err == nil {
		deps, err := s.parseRequirementsTxt(requirementsPath)
		if err == nil {
			dependencies = append(dependencies, deps...)
		}
	}

	return dependencies, nil
}

func (s *SecurityScanner) parseGoMod(path string) ([]DependencyInfo, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var deps []DependencyInfo
	lines := strings.Split(string(content), "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "require") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				deps = append(deps, DependencyInfo{
					Name:    parts[1],
					Version: parts[2],
					Type:    "direct",
					Source:  "go.mod",
				})
			}
		}
	}

	return deps, nil
}

func (s *SecurityScanner) parsePackageJSON(path string) ([]DependencyInfo, error) {
	// Simplified parsing - would use encoding/json in real implementation
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var deps []DependencyInfo
	// Simple pattern matching for dependencies
	depRegex := regexp.MustCompile(`"([^"]+)":\s*"([^"]+)"`)
	matches := depRegex.FindAllStringSubmatch(string(content), -1)
	
	for _, match := range matches {
		if len(match) >= 3 {
			deps = append(deps, DependencyInfo{
				Name:    match[1],
				Version: match[2],
				Type:    "direct",
				Source:  "package.json",
			})
		}
	}

	return deps, nil
}

func (s *SecurityScanner) parseRequirementsTxt(path string) ([]DependencyInfo, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var deps []DependencyInfo
	lines := strings.Split(string(content), "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse package==version or package>=version
		parts := strings.FieldsFunc(line, func(r rune) bool {
			return r == '=' || r == '>' || r == '<'
		})
		
		if len(parts) >= 1 {
			name := strings.TrimSpace(parts[0])
			version := ""
			if len(parts) >= 2 {
				version = strings.TrimSpace(parts[len(parts)-1])
			}

			deps = append(deps, DependencyInfo{
				Name:    name,
				Version: version,
				Type:    "direct",
				Source:  "requirements.txt",
			})
		}
	}

	return deps, nil
}
