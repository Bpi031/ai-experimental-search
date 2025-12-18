package ai

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v6"
)

// SymbolAnalyzer provides code symbol analysis for Go files
type SymbolAnalyzer struct {
	repo *git.Repository
}

// NewSymbolAnalyzer creates a new SymbolAnalyzer instance
func NewSymbolAnalyzer(repo *git.Repository) *SymbolAnalyzer {
	return &SymbolAnalyzer{repo: repo}
}

// SymbolType represents different kinds of symbols
type SymbolType string

const (
	SymbolTypeFunction   SymbolType = "function"
	SymbolTypeMethod     SymbolType = "method"
	SymbolTypeStruct     SymbolType = "struct"
	SymbolTypeInterface  SymbolType = "interface"
	SymbolTypeVariable   SymbolType = "variable"
	SymbolTypeConstant   SymbolType = "constant"
	SymbolTypeType       SymbolType = "type"
	SymbolTypeImport     SymbolType = "import"
)

// Symbol represents a code symbol
type Symbol struct {
	Name       string
	Type       SymbolType
	FilePath   string
	Line       int
	EndLine    int      // End line of the symbol
	Column     int
	Signature  string
	DocComment string
	Receiver   string   // For methods
	Calls      []string // Functions called by this symbol
}

// GetSymbolsOptions configures symbol extraction
type GetSymbolsOptions struct {
	// FilePath specifies which file to analyze (empty for all Go files)
	FilePath string
	// SymbolTypes filters by symbol type (empty for all types)
	SymbolTypes []SymbolType
	// IncludeDocs includes documentation comments
	IncludeDocs bool
}

// GetSymbols extracts symbols from Go source files
func (s *SymbolAnalyzer) GetSymbols(ctx context.Context, opts GetSymbolsOptions) ([]Symbol, error) {
	wt, err := s.repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	var files []string
	if opts.FilePath != "" {
		files = []string{filepath.Join(wt.Filesystem.Root(), opts.FilePath)}
	} else {
		// Find all Go files in repository
		err := filepath.Walk(wt.Filesystem.Root(), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(path, ".go") && !strings.Contains(path, "/vendor/") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to walk directory: %w", err)
		}
	}

	var allSymbols []Symbol
	fset := token.NewFileSet()

	for _, file := range files {
		relPath, _ := filepath.Rel(wt.Filesystem.Root(), file)
		
		node, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			continue // Skip files with parse errors
		}

		symbols := s.extractSymbols(fset, node, relPath, opts)
		allSymbols = append(allSymbols, symbols...)
	}

	return allSymbols, nil
}

// ExtractSymbolsFromContent extracts symbols from a file content string/byte slice
func (s *SymbolAnalyzer) ExtractSymbolsFromContent(filename string, content []byte, opts GetSymbolsOptions) ([]Symbol, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, content, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	return s.extractSymbols(fset, node, filename, opts), nil
}

func (s *SymbolAnalyzer) extractSymbols(fset *token.FileSet, node *ast.File, filePath string, opts GetSymbolsOptions) []Symbol {
	var symbols []Symbol
	typeFilter := make(map[SymbolType]bool)
	for _, t := range opts.SymbolTypes {
		typeFilter[t] = true
	}

	ast.Inspect(node, func(n ast.Node) bool {
		if n == nil {
			return false
		}

		pos := fset.Position(n.Pos())
		endPos := fset.Position(n.End())

		switch decl := n.(type) {
		case *ast.FuncDecl:
			symType := SymbolTypeFunction
			receiver := ""
			if decl.Recv != nil && len(decl.Recv.List) > 0 {
				symType = SymbolTypeMethod
				if starExpr, ok := decl.Recv.List[0].Type.(*ast.StarExpr); ok {
					if ident, ok := starExpr.X.(*ast.Ident); ok {
						receiver = "*" + ident.Name
					}
				} else if ident, ok := decl.Recv.List[0].Type.(*ast.Ident); ok {
					receiver = ident.Name
				}
			}

			if len(typeFilter) == 0 || typeFilter[symType] {
				signature := s.buildFuncSignature(decl)
				doc := ""
				if opts.IncludeDocs && decl.Doc != nil {
					doc = decl.Doc.Text()
				}

				calls := s.extractCalls(decl.Body)

				symbols = append(symbols, Symbol{
					Name:       decl.Name.Name,
					Type:       symType,
					FilePath:   filePath,
					Line:       pos.Line,
					EndLine:    endPos.Line,
					Column:     pos.Column,
					Signature:  signature,
					DocComment: doc,
					Receiver:   receiver,
					Calls:      calls,
				})
			}

		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					var symType SymbolType
					switch s.Type.(type) {
					case *ast.StructType:
						symType = SymbolTypeStruct
					case *ast.InterfaceType:
						symType = SymbolTypeInterface
					default:
						symType = SymbolTypeType
					}

					if len(typeFilter) == 0 || typeFilter[symType] {
						doc := ""
						if opts.IncludeDocs && decl.Doc != nil {
							doc = decl.Doc.Text()
						}

						symbols = append(symbols, Symbol{
							Name:       s.Name.Name,
							Type:       symType,
							FilePath:   filePath,
							Line:       pos.Line,
							EndLine:    endPos.Line,
							Column:     pos.Column,
							Signature:  s.Name.Name,
							DocComment: doc,
						})
					}

				case *ast.ValueSpec:
					symType := SymbolTypeVariable
					if decl.Tok == token.CONST {
						symType = SymbolTypeConstant
					}

					if len(typeFilter) == 0 || typeFilter[symType] {
						for _, name := range s.Names {
							doc := ""
							if opts.IncludeDocs && decl.Doc != nil {
								doc = decl.Doc.Text()
							}

							symbols = append(symbols, Symbol{
								Name:       name.Name,
								Type:       symType,
								FilePath:   filePath,
								Line:       pos.Line,
								EndLine:    endPos.Line,
								Column:     pos.Column,
								DocComment: doc,
							})
						}
					}

				case *ast.ImportSpec:
					if len(typeFilter) == 0 || typeFilter[SymbolTypeImport] {
						importPath := strings.Trim(s.Path.Value, `"`)
						name := importPath
						if s.Name != nil {
							name = s.Name.Name
						}

						symbols = append(symbols, Symbol{
							Name:     name,
							Type:     SymbolTypeImport,
							FilePath: filePath,
							Line:     pos.Line,
							EndLine:  endPos.Line,
							Column:   pos.Column,
							Signature: importPath,
						})
					}
				}
			}
		}

		return true
	})

	return symbols
}

func (s *SymbolAnalyzer) buildFuncSignature(decl *ast.FuncDecl) string {
	var sig strings.Builder
	sig.WriteString("func ")

	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		sig.WriteString("(")
		sig.WriteString(s.formatFieldList(decl.Recv))
		sig.WriteString(") ")
	}

	sig.WriteString(decl.Name.Name)
	sig.WriteString("(")
	if decl.Type.Params != nil {
		sig.WriteString(s.formatFieldList(decl.Type.Params))
	}
	sig.WriteString(")")

	if decl.Type.Results != nil && len(decl.Type.Results.List) > 0 {
		sig.WriteString(" ")
		if len(decl.Type.Results.List) > 1 || len(decl.Type.Results.List[0].Names) > 1 {
			sig.WriteString("(")
			sig.WriteString(s.formatFieldList(decl.Type.Results))
			sig.WriteString(")")
		} else {
			sig.WriteString(s.formatFieldList(decl.Type.Results))
		}
	}

	return sig.String()
}

func (s *SymbolAnalyzer) formatFieldList(fields *ast.FieldList) string {
	var parts []string
	for _, field := range fields.List {
		typeStr := s.exprToString(field.Type)
		if len(field.Names) == 0 {
			parts = append(parts, typeStr)
		} else {
			for _, name := range field.Names {
				parts = append(parts, name.Name+" "+typeStr)
			}
		}
	}
	return strings.Join(parts, ", ")
}

func (s *SymbolAnalyzer) exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + s.exprToString(e.X)
	case *ast.SelectorExpr:
		return s.exprToString(e.X) + "." + e.Sel.Name
	case *ast.ArrayType:
		return "[]" + s.exprToString(e.Elt)
	case *ast.MapType:
		return "map[" + s.exprToString(e.Key) + "]" + s.exprToString(e.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.Ellipsis:
		return "..." + s.exprToString(e.Elt)
	default:
		return "unknown"
	}
}

func (s *SymbolAnalyzer) extractCalls(body *ast.BlockStmt) []string {
	if body == nil {
		return nil
	}

	calls := make(map[string]bool)
	ast.Inspect(body, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			name := s.exprToString(call.Fun)
			if name != "unknown" {
				calls[name] = true
			}
		}
		return true
	})

	var result []string
	for name := range calls {
		result = append(result, name)
	}
	return result
}

// FindReferencesOptions configures reference search
type FindReferencesOptions struct {
	SymbolName string
	FilePath   string // Optional: limit to specific file
}

// Reference represents a symbol reference
type Reference struct {
	FilePath string
	Line     int
	Column   int
	Context  string // Line of code containing the reference
}

// FindReferences finds all references to a symbol
func (s *SymbolAnalyzer) FindReferences(ctx context.Context, opts FindReferencesOptions) ([]Reference, error) {
	if opts.SymbolName == "" {
		return nil, fmt.Errorf("symbol name is required")
	}

	wt, err := s.repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	var files []string
	if opts.FilePath != "" {
		files = []string{filepath.Join(wt.Filesystem.Root(), opts.FilePath)}
	} else {
		// Search all Go files
		err := filepath.Walk(wt.Filesystem.Root(), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(path, ".go") && !strings.Contains(path, "/vendor/") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to walk directory: %w", err)
		}
	}

	var references []Reference
	fset := token.NewFileSet()

	for _, file := range files {
		relPath, _ := filepath.Rel(wt.Filesystem.Root(), file)
		
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		node, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")

		ast.Inspect(node, func(n ast.Node) bool {
			if ident, ok := n.(*ast.Ident); ok {
				if ident.Name == opts.SymbolName {
					pos := fset.Position(ident.Pos())
					context := ""
					if pos.Line > 0 && pos.Line <= len(lines) {
						context = strings.TrimSpace(lines[pos.Line-1])
					}

					references = append(references, Reference{
						FilePath: relPath,
						Line:     pos.Line,
						Column:   pos.Column,
						Context:  context,
					})
				}
			}
			return true
		})
	}

	return references, nil
}

// GetDefinition finds the definition of a symbol
func (s *SymbolAnalyzer) GetDefinition(ctx context.Context, symbolName string) (*Symbol, error) {
	symbols, err := s.GetSymbols(ctx, GetSymbolsOptions{
		IncludeDocs: true,
	})
	if err != nil {
		return nil, err
	}

	for _, sym := range symbols {
		if sym.Name == symbolName && sym.Type != SymbolTypeImport {
			return &sym, nil
		}
	}

	return nil, fmt.Errorf("symbol %q not found", symbolName)
}
