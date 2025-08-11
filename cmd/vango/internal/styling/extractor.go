package styling

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Extractor extracts and processes component styles from Go source files
type Extractor struct {
	outputDir string
	styles    map[string]*ExtractedStyle // hash -> style
}

// ExtractedStyle represents an extracted component style
type ExtractedStyle struct {
	Hash       string
	CSS        string
	ClassNames map[string]string // original -> hashed
	FilePath   string
	Position   token.Pos
}

// NewExtractor creates a new style extractor
func NewExtractor(outputDir string) *Extractor {
	return &Extractor{
		outputDir: outputDir,
		styles:    make(map[string]*ExtractedStyle),
	}
}

// ExtractFromFile extracts styles from a single Go file
func (e *Extractor) ExtractFromFile(filePath string) error {
	// Read file
	src, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	
	// Parse AST
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}
	
	// Find and extract Style() calls
	modified := false
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		
		// Check if this is a Style() call
		if !isStyleCall(call) {
			return true
		}
		
		// Extract CSS from the call
		css, err := extractCSS(call)
		if err != nil {
			// Log error but continue
			fmt.Printf("Warning: failed to extract CSS at %s: %v\n", fset.Position(call.Pos()), err)
			return true
		}
		
		// Process the CSS
		style := e.processStyle(css, filePath, call.Pos())
		
		// Rewrite the call site
		if e.rewriteStyleCall(call, style) {
			modified = true
		}
		
		return true
	})
	
	// If modified, write the updated file
	if modified {
		// Format the modified AST
		var buf strings.Builder
		if err := format.Node(&buf, fset, file); err != nil {
			return fmt.Errorf("failed to format modified AST: %w", err)
		}
		
		// Write back to file
		if err := os.WriteFile(filePath, []byte(buf.String()), 0644); err != nil {
			return fmt.Errorf("failed to write modified file: %w", err)
		}
	}
	
	return nil
}

// ExtractFromDir recursively extracts styles from all Go files in a directory
func (e *Extractor) ExtractFromDir(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip non-Go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		
		// Skip test files
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		
		// Extract from this file
		return e.ExtractFromFile(path)
	})
}

// WriteCSS writes all extracted CSS to files
func (e *Extractor) WriteCSS() error {
	// Create output directory
	if err := os.MkdirAll(e.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Write each style to a separate file
	for hash, style := range e.styles {
		filename := filepath.Join(e.outputDir, fmt.Sprintf("_%s.css", hash))
		if err := os.WriteFile(filename, []byte(style.CSS), 0644); err != nil {
			return fmt.Errorf("failed to write CSS file %s: %w", filename, err)
		}
	}
	
	// Write a master index file that imports all component styles
	if err := e.writeIndexCSS(); err != nil {
		return fmt.Errorf("failed to write index CSS: %w", err)
	}
	
	return nil
}

// processStyle processes CSS and generates hashed class names
func (e *Extractor) processStyle(css, filePath string, pos token.Pos) *ExtractedStyle {
	// Generate hash from CSS content
	hasher := sha256.New()
	hasher.Write([]byte(css))
	hashBytes := hasher.Sum(nil)
	hash := hex.EncodeToString(hashBytes)[:8] // Use first 8 chars
	
	// Extract class names from CSS
	classNames := extractClassNames(css)
	
	// Generate hashed class names
	hashedNames := make(map[string]string)
	processedCSS := css
	
	for _, className := range classNames {
		hashedName := fmt.Sprintf("_%s_%s", hash, className)
		hashedNames[className] = hashedName
		
		// Replace class names in CSS
		// Match .className but not .className-other
		pattern := regexp.MustCompile(`\.` + regexp.QuoteMeta(className) + `\b`)
		processedCSS = pattern.ReplaceAllString(processedCSS, "."+hashedName)
	}
	
	style := &ExtractedStyle{
		Hash:       hash,
		CSS:        processedCSS,
		ClassNames: hashedNames,
		FilePath:   filePath,
		Position:   pos,
	}
	
	// Store the style
	e.styles[hash] = style
	
	return style
}

// rewriteStyleCall rewrites a Style() call to a literal ComponentStyle struct
func (e *Extractor) rewriteStyleCall(call *ast.CallExpr, style *ExtractedStyle) bool {
	// Build the map literal for class names
	mapElts := make([]ast.Expr, 0, len(style.ClassNames))
	for orig, hashed := range style.ClassNames {
		mapElts = append(mapElts, &ast.KeyValueExpr{
			Key:   &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf(`"%s"`, orig)},
			Value: &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf(`"%s"`, hashed)},
		})
	}
	
	// Create the literal ComponentStyle struct
	literal := &ast.UnaryExpr{
		Op: token.AND, // & operator for pointer
		X: &ast.CompositeLit{
			Type: &ast.SelectorExpr{
				X:   ast.NewIdent("styling"),
				Sel: ast.NewIdent("ComponentStyle"),
			},
			Elts: []ast.Expr{
				&ast.KeyValueExpr{
					Key:   ast.NewIdent("Hash"),
					Value: &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf(`"%s"`, style.Hash)},
				},
				&ast.KeyValueExpr{
					Key: ast.NewIdent("names"),
					Value: &ast.CompositeLit{
						Type: &ast.MapType{
							Key:   ast.NewIdent("string"),
							Value: ast.NewIdent("string"),
						},
						Elts: mapElts,
					},
				},
			},
		},
	}
	
	// Replace the call expression with the literal
	*call = ast.CallExpr{
		Fun: &ast.ParenExpr{
			X: &ast.FuncLit{
				Type: &ast.FuncType{
					Results: &ast.FieldList{
						List: []*ast.Field{{
							Type: &ast.StarExpr{
								X: &ast.SelectorExpr{
									X:   ast.NewIdent("styling"),
									Sel: ast.NewIdent("ComponentStyle"),
								},
							},
						}},
					},
				},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.ReturnStmt{
							Results: []ast.Expr{literal},
						},
					},
				},
			},
		},
		Args: []ast.Expr{},
	}
	
	return true
}

// writeIndexCSS writes a master CSS file that imports all component styles
func (e *Extractor) writeIndexCSS() error {
	var builder strings.Builder
	builder.WriteString("/* Generated by Vango - Component Styles */\n\n")
	
	for hash := range e.styles {
		fmt.Fprintf(&builder, "@import './_%s.css';\n", hash)
	}
	
	indexPath := filepath.Join(e.outputDir, "components.css")
	return os.WriteFile(indexPath, []byte(builder.String()), 0644)
}

// Helper functions

// isStyleCall checks if an AST call expression is a Style() call
func isStyleCall(call *ast.CallExpr) bool {
	// Check for direct call: Style(...)
	if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "Style" {
		return true
	}
	
	// Check for qualified call: vango.Style(...) or styling.Style(...)
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if sel.Sel.Name == "Style" {
			if pkg, ok := sel.X.(*ast.Ident); ok {
				return pkg.Name == "vango" || pkg.Name == "styling"
			}
		}
	}
	
	return false
}

// extractCSS extracts the CSS string from a Style() call
func extractCSS(call *ast.CallExpr) (string, error) {
	if len(call.Args) != 1 {
		return "", fmt.Errorf("Style() expects exactly 1 argument, got %d", len(call.Args))
	}
	
	// Extract string literal
	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", fmt.Errorf("Style() argument must be a string literal")
	}
	
	// Remove quotes from the literal value
	css := lit.Value
	if len(css) >= 2 {
		css = css[1 : len(css)-1] // Remove surrounding quotes
	}
	
	// Handle escape sequences
	css = strings.ReplaceAll(css, `\"`, `"`)
	css = strings.ReplaceAll(css, `\\`, `\`)
	css = strings.ReplaceAll(css, `\n`, "\n")
	css = strings.ReplaceAll(css, `\t`, "\t")
	
	return css, nil
}

// extractClassNames extracts all class names from CSS
func extractClassNames(css string) []string {
	// Regular expression to match CSS class selectors
	re := regexp.MustCompile(`\.([a-zA-Z][a-zA-Z0-9_-]*)`)
	matches := re.FindAllStringSubmatch(css, -1)
	
	// Deduplicate class names
	seen := make(map[string]bool)
	classes := make([]string, 0)
	
	for _, match := range matches {
		if len(match) > 1 {
			className := match[1]
			if !seen[className] {
				seen[className] = true
				classes = append(classes, className)
			}
		}
	}
	
	return classes
}

// WriteTailwindContent generates a content file for Tailwind CSS scanning
func (e *Extractor) WriteTailwindContent(outputPath string) error {
	// Collect all class names used in components
	allClasses := make([]string, 0)
	
	for _, style := range e.styles {
		for _, hashedName := range style.ClassNames {
			allClasses = append(allClasses, hashedName)
		}
	}
	
	// Write as JSON for Tailwind to scan
	content := strings.Join(allClasses, " ")
	
	return os.WriteFile(outputPath, []byte(content), 0644)
}

// GetStylesForFile returns all styles extracted from a specific file
func (e *Extractor) GetStylesForFile(filePath string) []*ExtractedStyle {
	styles := make([]*ExtractedStyle, 0)
	for _, style := range e.styles {
		if style.FilePath == filePath {
			styles = append(styles, style)
		}
	}
	return styles
}

// Run executes the style extraction process
func Run(sourceDir, outputDir string) error {
	extractor := NewExtractor(outputDir)
	
	// Extract styles from all Go files
	if err := extractor.ExtractFromDir(sourceDir); err != nil {
		return fmt.Errorf("failed to extract styles: %w", err)
	}
	
	// Write extracted CSS files
	if err := extractor.WriteCSS(); err != nil {
		return fmt.Errorf("failed to write CSS files: %w", err)
	}
	
	fmt.Printf("Extracted %d component styles\n", len(extractor.styles))
	
	return nil
}