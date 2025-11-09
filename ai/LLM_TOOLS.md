# AI Tool Definitions for LLM Integration

This document defines all AI tools available for LLM agents to use with go-git repositories.

## Tool Categories

1. **Search & Discovery** - Find code, files, and information
2. **Understanding** - Analyze code structure and history
3. **Modification** - Edit, create, and delete files
4. **Git Operations** - Commit, branch, and version control
5. **Security** - Scan for secrets and vulnerabilities

---

## 1. Search & Discovery Tools

### search_codebase (Keyword)

**Description**: Search for code using keyword matching (fast, local, no AI required).

**Use When**:
- User mentions specific filename or code pattern
- Need exact string matching
- Fast results required

**Parameters**:
```json
{
  "query": "string (required) - Search query",
  "topK": "number (optional) - Max results, default 10",
  "filePatterns": "[]string (optional) - Filter by patterns like *.go"
}
```

**Returns**: Array of SearchResult with file paths, line numbers, and context

**Example**:
```json
{
  "query": "CloneOptions",
  "topK": 5
}
```

---

### search_codebase (Semantic)

**Description**: AI-powered semantic search using embeddings (finds conceptually similar code).

**Use When**:
- User describes functionality conceptually
- Need to find similar implementations
- Keyword search returns no results

**Parameters**:
```json
{
  "query": "string (required) - Conceptual query",
  "topK": "number (optional) - Max results, default 5",
  "rerank": "boolean (optional) - Use reranker for better accuracy"
}
```

**Returns**: Array of SearchResult with relevance scores

**Example**:
```json
{
  "query": "clone repository with authentication",
  "topK": 3,
  "rerank": true
}
```

---

### get_symbols

**Description**: Extract code symbols (functions, types, methods) from Go files.

**Use When**:
- Need to understand code structure
- Looking for specific function/type
- Building code navigation

**Parameters**:
```json
{
  "filePath": "string (optional) - Specific file or all files",
  "symbolTypes": "[]string (optional) - Filter: function, method, struct, interface",
  "includeDocs": "boolean (optional) - Include documentation comments"
}
```

**Returns**: Array of Symbol with name, type, location, signature, docs

**Example**:
```json
{
  "filePath": "repository.go",
  "symbolTypes": ["function", "method"],
  "includeDocs": true
}
```

---

### find_references

**Description**: Find all references to a symbol across the codebase.

**Use When**:
- Need to see where a function/type is used
- Understanding code dependencies
- Planning refactoring

**Parameters**:
```json
{
  "symbolName": "string (required) - Symbol to find",
  "filePath": "string (optional) - Limit to specific file"
}
```

**Returns**: Array of Reference with file path, line, column, context

**Example**:
```json
{
  "symbolName": "Repository",
  "filePath": ""
}
```

---

### get_definition

**Description**: Find the definition of a symbol.

**Use When**:
- User asks "where is X defined"
- Need to jump to definition
- Understanding symbol origin

**Parameters**:
```json
{
  "symbolName": "string (required) - Symbol name"
}
```

**Returns**: Single Symbol with definition location and details

**Example**:
```json
{
  "symbolName": "PlainClone"
}
```

---

## 2. Understanding Tools

### get_diff

**Description**: View differences between commits or working tree.

**Use When**:
- User asks "what changed"
- Need to review modifications
- Understanding commit content

**Parameters**:
```json
{
  "fromCommit": "string (optional) - Start commit hash, default HEAD",
  "toCommit": "string (optional) - End commit hash, empty for working tree",
  "filePath": "string (optional) - Limit to specific file",
  "maxLines": "number (optional) - Limit diff lines"
}
```

**Returns**: Array of DiffResult with file paths, additions, deletions, diff text

**Example**:
```json
{
  "fromCommit": "abc123",
  "toCommit": "",
  "maxLines": 100
}
```

---

### get_blame

**Description**: Show line-by-line authorship information.

**Use When**:
- User asks "who wrote this"
- Need to understand code history
- Finding when line was added

**Parameters**:
```json
{
  "filePath": "string (required) - File to blame",
  "fromCommit": "string (optional) - Commit to blame from, default HEAD"
}
```

**Returns**: Array of BlameLine with line number, author, commit, timestamp

**Example**:
```json
{
  "filePath": "repository.go",
  "fromCommit": ""
}
```

---

### get_history

**Description**: Retrieve commit history with filtering.

**Use When**:
- User asks for commit history
- Need to understand changes over time
- Finding specific commits

**Parameters**:
```json
{
  "filePath": "string (optional) - Filter by file",
  "fromCommit": "string (optional) - Start from commit, default HEAD",
  "maxCount": "number (optional) - Limit commits, default unlimited",
  "author": "string (optional) - Filter by author",
  "since": "time (optional) - Only commits after this time",
  "until": "time (optional) - Only commits before this time"
}
```

**Returns**: Array of CommitInfo with hash, author, message, files, stats

**Example**:
```json
{
  "filePath": "ai/",
  "maxCount": 20,
  "author": "john"
}
```

---

## 3. Modification Tools (Dangerous - Require Confirmation)

### edit_file

**Description**: Modify existing file or create new one.

**Use When**:
- User requests code changes
- Fixing bugs or implementing features
- Updating configuration

**Parameters**:
```json
{
  "filePath": "string (required) - File to edit",
  "content": "string (optional) - Replace entire file",
  "lineRanges": "[]LineEdit (optional) - Edit specific lines",
  "createIfMissing": "boolean (optional) - Create if doesn't exist"
}
```

**Returns**: FileOperation with description, needs user confirmation

**Example**:
```json
{
  "filePath": "config.go",
  "lineRanges": [{
    "startLine": 10,
    "endLine": 12,
    "newText": "// Updated comment\nvar DefaultPort = 9000"
  }]
}
```

**⚠️ IMPORTANT**: ALWAYS get user confirmation before applying!

---

### create_file

**Description**: Create a new file.

**Use When**:
- User requests new file creation
- Generating new code
- Adding configuration

**Parameters**:
```json
{
  "filePath": "string (required) - Path for new file",
  "content": "string (required) - File content",
  "overwrite": "boolean (optional) - Overwrite if exists"
}
```

**Returns**: FileOperation with description, needs user confirmation

**Example**:
```json
{
  "filePath": "ai/new_feature.go",
  "content": "package ai\n\n// NewFeature implements...",
  "overwrite": false
}
```

**⚠️ IMPORTANT**: ALWAYS get user confirmation before applying!

---

### delete_file

**Description**: Delete a file.

**Use When**:
- User requests file deletion
- Cleaning up code
- Removing obsolete files

**Parameters**:
```json
{
  "filePath": "string (required) - File to delete"
}
```

**Returns**: FileOperation with description, needs user confirmation

**Example**:
```json
{
  "filePath": "old_code.go"
}
```

**⚠️ IMPORTANT**: ALWAYS get user confirmation before applying! File content is backed up in operation.

---

## 4. Git Operations (Dangerous - Require Confirmation)

### commit_changes

**Description**: Create a new commit with optional AI-generated message.

**Use When**:
- User requests commit
- Saving changes after modifications
- Checkpoint progress

**Parameters**:
```json
{
  "message": "string (optional) - Commit message, auto-generated if empty",
  "files": "[]string (optional) - Specific files, empty for all modified",
  "authorName": "string (optional) - Author name",
  "authorEmail": "string (optional) - Author email",
  "allowEmpty": "boolean (optional) - Allow empty commits",
  "amendPrevious": "boolean (optional) - Amend last commit"
}
```

**Returns**: CommitResult with hash, message, files changed, needs confirmation

**Example**:
```json
{
  "message": "feat: add AI search capabilities",
  "files": ["ai/search.go", "README.md"]
}
```

**⚠️ IMPORTANT**: ALWAYS get user confirmation before committing!

---

### create_branch

**Description**: Create a new branch.

**Use When**:
- User requests new branch
- Starting new feature
- Creating checkpoint

**Parameters**:
```json
{
  "name": "string (required) - Branch name",
  "fromCommit": "string (optional) - Create from commit, default HEAD",
  "checkout": "boolean (optional) - Switch to branch after creation",
  "force": "boolean (optional) - Overwrite existing branch"
}
```

**Returns**: BranchResult with name, commit hash, needs confirmation

**Example**:
```json
{
  "name": "feature/new-search",
  "fromCommit": "",
  "checkout": true,
  "force": false
}
```

**⚠️ IMPORTANT**: ALWAYS get user confirmation before creating branches!

---

### delete_branch

**Description**: Delete a branch.

**Use When**:
- User requests branch deletion
- Cleaning up old branches
- Removing feature branches

**Parameters**:
```json
{
  "branchName": "string (required) - Branch to delete",
  "force": "boolean (optional) - Force delete even if current branch"
}
```

**Returns**: Success/error

**Example**:
```json
{
  "branchName": "old-feature",
  "force": false
}
```

**⚠️ IMPORTANT**: Cannot delete current branch without force flag!

---

## 5. Security Tools

### scan_for_secrets

**Description**: Scan repository for hardcoded secrets and credentials.

**Use When**:
- User requests security scan
- Before committing changes
- Security audit

**Parameters**:
```json
{
  "filePath": "string (optional) - Scan specific file or all files",
  "customPatterns": "[]SecretPattern (optional) - Additional patterns",
  "excludePatterns": "[]string (optional) - Files to exclude"
}
```

**Returns**: ScanResult with findings, severity levels, statistics

**Example**:
```json
{
  "filePath": "",
  "excludePatterns": ["test_*"]
}
```

**Detects**:
- AWS keys
- GitHub tokens
- API keys
- Passwords
- Private keys
- Database credentials
- JWT tokens

---

### analyze_dependencies

**Description**: Analyze project dependencies from package manifests.

**Use When**:
- User asks about dependencies
- Security audit
- Understanding project structure

**Parameters**: None

**Returns**: Array of DependencyInfo with name, version, type, source

**Example**: No parameters needed

**Supports**:
- Go (go.mod)
- Node.js (package.json)
- Python (requirements.txt)

---

## Tool Selection Decision Tree

```
User Request:
│
├─ "find", "search", "locate" → search_codebase
│  ├─ Mentions filename → Keyword search
│  └─ Describes concept → Semantic search
│
├─ "show me", "explain", "what does" → search + get_symbols + read_file
│
├─ "who wrote", "when was" → get_blame or get_history
│
├─ "what changed" → get_diff
│
├─ "find function", "where is defined" → get_definition or find_references
│
├─ "edit", "change", "modify" → edit_file (needs confirmation!)
│
├─ "create", "add new" → create_file (needs confirmation!)
│
├─ "delete", "remove" → delete_file (needs confirmation!)
│
├─ "commit" → commit_changes (needs confirmation!)
│
├─ "create branch" → create_branch (needs confirmation!)
│
├─ "check security", "scan secrets" → scan_for_secrets
│
└─ "show dependencies" → analyze_dependencies
```

---

## Safety Guidelines

### ALWAYS Require Confirmation For:
1. ✅ File modifications (edit, create, delete)
2. ✅ Git write operations (commit, branch creation)
3. ✅ Destructive operations (delete branch, force push)

### Safe Operations (No Confirmation):
1. ✅ All search operations
2. ✅ Reading files and code
3. ✅ Viewing diffs, blame, history
4. ✅ Symbol analysis
5. ✅ Security scanning

### Confirmation Workflow:
```
1. User: "edit config.go to change port to 9000"
2. AI: Call edit_file() → Returns FileOperation
3. AI: "I'll edit config.go to change the port. Here's what will change:
       [show diff]
       Do you want me to apply this change?"
4. User: "yes"
5. AI: Call apply_file_operation() → Execute change
6. AI: "Done! File updated."
```

---

## Example LLM System Prompt

```
You are an AI coding assistant with access to a Git repository.

YOUR CAPABILITIES:
- search_codebase: Find code by keyword or semantic meaning
- get_symbols: Extract functions, types, etc.
- find_references: See where code is used
- get_diff/blame/history: Understand changes
- edit_file/create_file/delete_file: Modify code (REQUIRES CONFIRMATION)
- commit_changes/create_branch: Git operations (REQUIRES CONFIRMATION)
- scan_for_secrets: Security scanning

WORKFLOW FOR CODE CHANGES:
1. Search for relevant code
2. Understand current implementation
3. Propose changes and show diff
4. Get user confirmation
5. Apply changes
6. Offer to commit

SAFETY RULES:
- NEVER modify files without user confirmation
- NEVER commit without user confirmation
- ALWAYS show what will change before applying
- ALWAYS scan for secrets before committing

Example interaction:
User: "add a new search feature"
You:
1. search_codebase("search") to find existing search code
2. get_symbols() to understand structure
3. Propose new file/changes with create_file()
4. Show user the proposed code
5. Wait for confirmation
6. Apply changes
7. Offer to commit with generated message
```

---

## Integration with LLM APIs

### OpenAI Function Calling

```json
{
  "name": "search_codebase",
  "description": "Search repository code using keyword or semantic matching",
  "parameters": {
    "type": "object",
    "properties": {
      "query": {
        "type": "string",
        "description": "Search query - filename, keyword, or concept"
      },
      "searchType": {
        "type": "string",
        "enum": ["keyword", "semantic"],
        "description": "Search strategy"
      },
      "topK": {
        "type": "integer",
        "description": "Max results to return"
      }
    },
    "required": ["query"]
  }
}
```

### Anthropic Claude Tool Use

```json
{
  "name": "edit_file",
  "description": "Modify an existing file. REQUIRES USER CONFIRMATION before applying.",
  "input_schema": {
    "type": "object",
    "properties": {
      "filePath": {
        "type": "string",
        "description": "Path to file to edit"
      },
      "content": {
        "type": "string",
        "description": "New file content (replaces entire file)"
      }
    },
    "required": ["filePath", "content"]
  }
}
```

---

**Version**: 1.0  
**Last Updated**: 2025-11-09  
**Status**: Ready for LLM Integration
