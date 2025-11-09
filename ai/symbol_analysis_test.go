package ai

import (
	"context"
	"testing"
)

func TestSymbolAnalyzer_GetSymbols(t *testing.T) {
	repo := mustCreateTempRepo(t)
	
	// Write test Go file
	content := `package main

import "fmt"

// TestStruct is a test struct
type TestStruct struct {
	Field1 string
	Field2 int
}

// TestInterface is a test interface
type TestInterface interface {
	Method1() error
	Method2(arg string) int
}

const MaxValue = 100

var GlobalVar = "test"

// TestFunc is a test function
func TestFunc(a int, b string) error {
	return nil
}

func (t *TestStruct) Method() {
	fmt.Println("method")
}
`
	mustWrite(t, repo, "test.go", content)
	mustCommitAll(t, repo, "Add test file")
	
	// Test: Get symbols
	analyzer := NewSymbolAnalyzer(repo)
	symbols, err := analyzer.GetSymbols(context.Background(), GetSymbolsOptions{
		FilePath: "test.go",
	})
	if err != nil {
		t.Fatalf("GetSymbols failed: %v", err)
	}
	
	// Verify we found various symbol types
	foundTypes := make(map[SymbolType]bool)
	for _, sym := range symbols {
		foundTypes[sym.Type] = true
	}
	
	expectedTypes := []SymbolType{
		SymbolTypeStruct,
		SymbolTypeInterface,
		SymbolTypeConstant,
		SymbolTypeVariable,
		SymbolTypeFunction,
		SymbolTypeMethod,
		SymbolTypeImport,
	}
	
	for _, expectedType := range expectedTypes {
		if !foundTypes[expectedType] {
			t.Errorf("Did not find symbol type: %v", expectedType)
		}
	}
	
	// Verify specific symbols
	symbolNames := make(map[string]*Symbol)
	for i := range symbols {
		symbolNames[symbols[i].Name] = &symbols[i]
	}
	
	if _, ok := symbolNames["TestStruct"]; !ok {
		t.Error("Did not find TestStruct")
	}
	
	if _, ok := symbolNames["TestFunc"]; !ok {
		t.Error("Did not find TestFunc")
	}
	
	if sym, ok := symbolNames["MaxValue"]; ok {
		if sym.Type != SymbolTypeConstant {
			t.Errorf("MaxValue should be constant, got %v", sym.Type)
		}
	}
}

func TestSymbolAnalyzer_GetSymbolsWithTypeFilter(t *testing.T) {
	repo := mustCreateTempRepo(t)
	
	content := `package main

type MyStruct struct {}

func MyFunc() {}

const MyConst = 1
`
	mustWrite(t, repo, "test.go", content)
	mustCommitAll(t, repo, "Add test file")
	
	// Test: Filter by type
	analyzer := NewSymbolAnalyzer(repo)
	symbols, err := analyzer.GetSymbols(context.Background(), GetSymbolsOptions{
		FilePath:    "test.go",
		SymbolTypes: []SymbolType{SymbolTypeFunction},
	})
	if err != nil {
		t.Fatalf("GetSymbols with filter failed: %v", err)
	}
	
	// Should only have functions
	for _, sym := range symbols {
		if sym.Type != SymbolTypeFunction {
			t.Errorf("Expected only functions, got %v", sym.Type)
		}
	}
	
	if len(symbols) != 1 {
		t.Errorf("Expected 1 function, got %d", len(symbols))
	}
}

func TestSymbolAnalyzer_GetSymbolsWithDocs(t *testing.T) {
	repo := mustCreateTempRepo(t)
	
	content := `package main

// DocumentedFunc performs an important operation
// It takes no parameters and returns nothing
func DocumentedFunc() {
}

func UndocumentedFunc() {
}
`
	mustWrite(t, repo, "test.go", content)
	mustCommitAll(t, repo, "Add test file")
	
	// Test: Get symbols with documentation
	analyzer := NewSymbolAnalyzer(repo)
	symbols, err := analyzer.GetSymbols(context.Background(), GetSymbolsOptions{
		FilePath:    "test.go",
		IncludeDocs: true,
	})
	if err != nil {
		t.Fatalf("GetSymbols failed: %v", err)
	}
	
	// Find DocumentedFunc
	var documentedFunc *Symbol
	for i := range symbols {
		if symbols[i].Name == "DocumentedFunc" {
			documentedFunc = &symbols[i]
			break
		}
	}
	
	if documentedFunc == nil {
		t.Fatal("Did not find DocumentedFunc")
	}
	
	if documentedFunc.DocComment == "" {
		t.Error("DocumentedFunc should have documentation")
	}
	
	// Verify doc content
	if len(documentedFunc.DocComment) < 10 {
		t.Errorf("Documentation seems too short: %s", documentedFunc.DocComment)
	}
}

func TestSymbolAnalyzer_FindReferences(t *testing.T) {
	repo := mustCreateTempRepo(t)
	
	// Create multiple files
	file1Content := `package main

func TargetFunc() {
	println("target")
}
`
	file2Content := `package main

func Caller() {
	TargetFunc()
	TargetFunc()
}
`
	mustWrite(t, repo, "define.go", file1Content)
	mustWrite(t, repo, "use.go", file2Content)
	mustCommitAll(t, repo, "Add files")
	
	// Test: Find references
	analyzer := NewSymbolAnalyzer(repo)
	refs, err := analyzer.FindReferences(context.Background(), FindReferencesOptions{
		SymbolName: "TargetFunc",
	})
	if err != nil {
		t.Fatalf("FindReferences failed: %v", err)
	}
	
	if len(refs) < 2 {
		t.Errorf("Expected at least 2 references, got %d", len(refs))
	}
	
	// Verify references have location info
	for _, ref := range refs {
		if ref.FilePath == "" {
			t.Error("Reference has empty file path")
		}
		if ref.Line == 0 {
			t.Error("Reference has zero line number")
		}
		if ref.Context == "" {
			t.Error("Reference has empty context")
		}
	}
}

func TestSymbolAnalyzer_FindReferencesWithFilePath(t *testing.T) {
	repo := mustCreateTempRepo(t)
	
	file1Content := `package main
func Target() {}
func Use1() { Target() }
`
	file2Content := `package main
func Use2() { Target() }
`
	mustWrite(t, repo, "file1.go", file1Content)
	mustWrite(t, repo, "file2.go", file2Content)
	mustCommitAll(t, repo, "Add files")
	
	// Test: Find references in specific file
	analyzer := NewSymbolAnalyzer(repo)
	refs, err := analyzer.FindReferences(context.Background(), FindReferencesOptions{
		SymbolName: "Target",
		FilePath:   "file1.go",
	})
	if err != nil {
		t.Fatalf("FindReferences with FilePath failed: %v", err)
	}
	
	// Should only find references in file1.go
	for _, ref := range refs {
		if ref.FilePath != "file1.go" {
			t.Errorf("Expected only file1.go references, got %s", ref.FilePath)
		}
	}
}

func TestSymbolAnalyzer_GetDefinition(t *testing.T) {
	repo := mustCreateTempRepo(t)
	
	content := `package main

// TargetFunc is the function we're looking for
func TargetFunc(arg1 int, arg2 string) error {
	return nil
}

func Other() {
	TargetFunc(1, "test")
}
`
	mustWrite(t, repo, "test.go", content)
	mustCommitAll(t, repo, "Add test file")
	
	// Test: Get definition
	analyzer := NewSymbolAnalyzer(repo)
	def, err := analyzer.GetDefinition(context.Background(), "TargetFunc")
	if err != nil {
		t.Fatalf("GetDefinition failed: %v", err)
	}
	
	if def.Name != "TargetFunc" {
		t.Errorf("Expected name TargetFunc, got %s", def.Name)
	}
	
	if def.Type != SymbolTypeFunction {
		t.Errorf("Expected function type, got %v", def.Type)
	}
	
	if def.Line == 0 {
		t.Error("Definition line should not be zero")
	}
	
	if def.DocComment == "" {
		t.Log("Warning: Documentation not found (may be expected)")
	}
}

func TestSymbolAnalyzer_GetDefinitionNotFound(t *testing.T) {
	repo := mustCreateTempRepo(t)
	
	content := `package main
func SomeFunc() {}
`
	mustWrite(t, repo, "test.go", content)
	mustCommitAll(t, repo, "Add test file")
	
	// Test: Definition not found should error
	analyzer := NewSymbolAnalyzer(repo)
	_, err := analyzer.GetDefinition(context.Background(), "NonExistent")
	if err == nil {
		t.Error("Expected error for non-existent symbol, got nil")
	}
}

func TestSymbolAnalyzer_ComplexStructs(t *testing.T) {
	repo := mustCreateTempRepo(t)
	
	content := `package main

type ComplexStruct struct {
	Field1 string
	Field2 int
	Nested struct {
		Inner string
	}
}

type GenericStruct[T any] struct {
	Value T
}
`
	mustWrite(t, repo, "test.go", content)
	mustCommitAll(t, repo, "Add test file")
	
	// Test: Parse complex structs
	analyzer := NewSymbolAnalyzer(repo)
	symbols, err := analyzer.GetSymbols(context.Background(), GetSymbolsOptions{
		FilePath:    "test.go",
		SymbolTypes: []SymbolType{SymbolTypeStruct},
	})
	if err != nil {
		t.Fatalf("GetSymbols failed: %v", err)
	}
	
	if len(symbols) < 2 {
		t.Errorf("Expected at least 2 structs, got %d", len(symbols))
	}
	
	// Verify we found both structs
	found := make(map[string]bool)
	for _, sym := range symbols {
		found[sym.Name] = true
	}
	
	if !found["ComplexStruct"] {
		t.Error("Did not find ComplexStruct")
	}
	if !found["GenericStruct"] {
		t.Error("Did not find GenericStruct")
	}
}

func TestSymbolAnalyzer_Methods(t *testing.T) {
	repo := mustCreateTempRepo(t)
	
	content := `package main

type MyType struct {}

func (m *MyType) PointerMethod() {}

func (m MyType) ValueMethod() {}
`
	mustWrite(t, repo, "test.go", content)
	mustCommitAll(t, repo, "Add test file")
	
	// Test: Find methods
	analyzer := NewSymbolAnalyzer(repo)
	symbols, err := analyzer.GetSymbols(context.Background(), GetSymbolsOptions{
		FilePath:    "test.go",
		SymbolTypes: []SymbolType{SymbolTypeMethod},
	})
	if err != nil {
		t.Fatalf("GetSymbols failed: %v", err)
	}
	
	if len(symbols) != 2 {
		t.Errorf("Expected 2 methods, got %d", len(symbols))
	}
	
	// Verify both methods found
	methods := make(map[string]bool)
	for _, sym := range symbols {
		methods[sym.Name] = true
		
		// Verify receiver is set
		if sym.Receiver == "" {
			t.Errorf("Method %s should have receiver set", sym.Name)
		}
	}
	
	if !methods["PointerMethod"] {
		t.Error("Did not find PointerMethod")
	}
	if !methods["ValueMethod"] {
		t.Error("Did not find ValueMethod")
	}
}

func TestSymbolAnalyzer_InterfaceMethods(t *testing.T) {
	repo := mustCreateTempRepo(t)
	
	content := `package main

type Reader interface {
	Read(p []byte) (n int, err error)
}

type Writer interface {
	Write(p []byte) (n int, err error)
}
`
	mustWrite(t, repo, "test.go", content)
	mustCommitAll(t, repo, "Add test file")
	
	// Test: Parse interfaces
	analyzer := NewSymbolAnalyzer(repo)
	symbols, err := analyzer.GetSymbols(context.Background(), GetSymbolsOptions{
		FilePath:    "test.go",
		SymbolTypes: []SymbolType{SymbolTypeInterface},
	})
	if err != nil {
		t.Fatalf("GetSymbols failed: %v", err)
	}
	
	if len(symbols) != 2 {
		t.Errorf("Expected 2 interfaces, got %d", len(symbols))
	}
	
	found := make(map[string]bool)
	for _, sym := range symbols {
		found[sym.Name] = true
	}
	
	if !found["Reader"] {
		t.Error("Did not find Reader interface")
	}
	if !found["Writer"] {
		t.Error("Did not find Writer interface")
	}
}

func TestSymbolAnalyzer_MultipleFiles(t *testing.T) {
	repo := mustCreateTempRepo(t)
	
	// Create multiple Go files
	file1 := `package main
func Func1() {}
`
	file2 := `package main
func Func2() {}
`
	file3 := `package main
func Func3() {}
`
	mustWrite(t, repo, "file1.go", file1)
	mustWrite(t, repo, "file2.go", file2)
	mustWrite(t, repo, "file3.go", file3)
	mustCommitAll(t, repo, "Add files")
	
	// Test: Get symbols from all files
	analyzer := NewSymbolAnalyzer(repo)
	symbols, err := analyzer.GetSymbols(context.Background(), GetSymbolsOptions{
		SymbolTypes: []SymbolType{SymbolTypeFunction},
	})
	if err != nil {
		t.Fatalf("GetSymbols failed: %v", err)
	}
	
	// Should find all 3 functions
	if len(symbols) < 3 {
		t.Errorf("Expected at least 3 functions across all files, got %d", len(symbols))
	}
	
	found := make(map[string]bool)
	for _, sym := range symbols {
		found[sym.Name] = true
	}
	
	if !found["Func1"] || !found["Func2"] || !found["Func3"] {
		t.Error("Did not find all expected functions")
	}
}

func TestSymbolAnalyzer_Imports(t *testing.T) {
	repo := mustCreateTempRepo(t)
	
	content := `package main

import (
	"fmt"
	"os"
	custom "path/to/custom"
)

func main() {
	fmt.Println("test")
}
`
	mustWrite(t, repo, "test.go", content)
	mustCommitAll(t, repo, "Add test file")
	
	// Test: Get imports
	analyzer := NewSymbolAnalyzer(repo)
	symbols, err := analyzer.GetSymbols(context.Background(), GetSymbolsOptions{
		FilePath:    "test.go",
		SymbolTypes: []SymbolType{SymbolTypeImport},
	})
	if err != nil {
		t.Fatalf("GetSymbols failed: %v", err)
	}
	
	if len(symbols) < 3 {
		t.Errorf("Expected at least 3 imports, got %d", len(symbols))
	}
	
	// Verify we found expected imports
	found := make(map[string]bool)
	for _, sym := range symbols {
		found[sym.Name] = true
	}
	
	if !found["fmt"] {
		t.Error("Did not find fmt import")
	}
	if !found["os"] {
		t.Error("Did not find os import")
	}
}

func TestSymbolAnalyzer_EmptyFile(t *testing.T) {
	repo := mustCreateTempRepo(t)
	
	content := `package main
`
	mustWrite(t, repo, "empty.go", content)
	mustCommitAll(t, repo, "Add empty file")
	
	// Test: Empty file should not error
	analyzer := NewSymbolAnalyzer(repo)
	symbols, err := analyzer.GetSymbols(context.Background(), GetSymbolsOptions{
		FilePath: "empty.go",
	})
	if err != nil {
		t.Fatalf("GetSymbols on empty file failed: %v", err)
	}
	
	// Should only have package declaration, no symbols
	for _, sym := range symbols {
		if sym.Type != SymbolTypeImport {
			t.Logf("Found unexpected symbol in empty file: %s (type: %v)", sym.Name, sym.Type)
		}
	}
}
