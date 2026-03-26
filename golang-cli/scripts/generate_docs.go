// Package main provides documentation generation for the Cline Go CLI.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/doc"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// DocConfig holds configuration for documentation generation.
type DocConfig struct {
	// SourceDir is the directory containing Go source code
	SourceDir string

	// OutputDir is where documentation will be written
	OutputDir string

	// TemplateDir contains custom templates (optional)
	TemplateDir string

	// Format is the output format (markdown, html)
	Format string

	// Title is the documentation title
	Title string

	// Version is the CLI version
	Version string
}

// PackageDocs represents documentation for a Go package.
type PackageDocs struct {
	Name        string
	ImportPath  string
	Doc         string
	Types       []TypeDoc
	Functions   []FunctionDoc
	Variables   []VariableDoc
	Constants   []ConstantDoc
	Subpackages []string
}

// TypeDoc represents documentation for a type.
type TypeDoc struct {
	Name    string
	Doc     string
	Methods []FunctionDoc
}

// FunctionDoc represents documentation for a function.
type FunctionDoc struct {
	Name       string
	Doc        string
	Signature  string
	IsExported bool
	Receiver   string // For methods
}

// VariableDoc represents documentation for a variable.
type VariableDoc struct {
	Name string
	Doc  string
	Type string
}

// ConstantDoc represents documentation for a constant.
type ConstantDoc struct {
	Name  string
	Doc   string
	Value string
}

// ParseFlags parses command-line flags.
func ParseFlags() *DocConfig {
	config := &DocConfig{}

	flag.StringVar(&config.SourceDir, "source", ".", "Source directory containing Go code")
	flag.StringVar(&config.OutputDir, "output", "./docs", "Output directory for generated documentation")
	flag.StringVar(&config.TemplateDir, "templates", "", "Directory containing custom templates")
	flag.StringVar(&config.Format, "format", "markdown", "Output format (markdown, html)")
	flag.StringVar(&config.Title, "title", "Cline CLI Documentation", "Documentation title")
	flag.StringVar(&config.Version, "version", "dev", "CLI version")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Generate API documentation for Cline CLI.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Generate markdown docs\n")
		fmt.Fprintf(os.Stderr, "  %s -source ./internal -output ./docs/api\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Generate with custom title\n")
		fmt.Fprintf(os.Stderr, "  %s -title \"Cline CLI v1.0\" -version 1.0.0\n", os.Args[0])
	}

	flag.Parse()

	return config
}

// GenerateDocs generates documentation from Go source files.
func GenerateDocs(config *DocConfig) error {
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Parse all packages in the source directory
	fset := token.NewFileSet()
	packages, err := parser.ParseDir(fset, config.SourceDir, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse source directory: %w", err)
	}

	var docs []PackageDocs

	for name, pkg := range packages {
		// Skip test packages
		if strings.HasSuffix(name, "_test") {
			continue
		}

		docPkg := doc.New(pkg, config.SourceDir, 0)
		pkgDocs := PackageDocs{
			Name:       docPkg.Name,
			ImportPath: docPkg.ImportPath,
			Doc:        docPkg.Doc,
		}

		// Extract types
		for _, t := range docPkg.Types {
			typeDoc := TypeDoc{
				Name: t.Name,
				Doc:  t.Doc,
			}

			// Extract methods
			for _, m := range t.Methods {
				sig, _ := getFuncSignature(m.Decl)
				typeDoc.Methods = append(typeDoc.Methods, FunctionDoc{
					Name:       m.Name,
					Doc:        m.Doc,
					Signature:  sig,
					IsExported: ast.IsExported(m.Name),
					Receiver:   t.Name,
				})
			}

			pkgDocs.Types = append(pkgDocs.Types, typeDoc)
		}

		// Extract functions
		for _, f := range docPkg.Funcs {
			sig, _ := getFuncSignature(f.Decl)
			pkgDocs.Functions = append(pkgDocs.Functions, FunctionDoc{
				Name:       f.Name,
				Doc:        f.Doc,
				Signature:  sig,
				IsExported: ast.IsExported(f.Name),
			})
		}

		// Extract variables
		for _, v := range docPkg.Vars {
			for _, spec := range v.Decl.Specs {
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					for _, name := range valueSpec.Names {
						pkgDocs.Variables = append(pkgDocs.Variables, VariableDoc{
							Name: name.Name,
							Doc:  v.Doc,
						})
					}
				}
			}
		}

		// Extract constants
		for _, c := range docPkg.Consts {
			for _, spec := range c.Decl.Specs {
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					for _, name := range valueSpec.Names {
						var value string
						if len(valueSpec.Values) > 0 {
							value = valueSpec.Values[0].(*ast.BasicLit).Value
						}
						pkgDocs.Constants = append(pkgDocs.Constants, ConstantDoc{
							Name:  name.Name,
							Doc:   c.Doc,
							Value: value,
						})
					}
				}
			}
		}

		docs = append(docs, pkgDocs)
	}

	// Generate output based on format
	switch config.Format {
	case "markdown":
		return generateMarkdownDocs(config, docs)
	case "html":
		return generateHTMLDocs(config, docs)
	default:
		return fmt.Errorf("unsupported format: %s", config.Format)
	}
}

func generateMarkdownDocs(config *DocConfig, docs []PackageDocs) error {
	// Generate main README
	readmePath := filepath.Join(config.OutputDir, "README.md")
	readmeContent := generateMarkdownREADME(config, docs)
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to write README: %w", err)
	}

	// Generate package documentation
	for _, pkg := range docs {
		pkgPath := filepath.Join(config.OutputDir, pkg.Name+".md")
		pkgContent := generateMarkdownPackage(config, pkg)
		if err := os.WriteFile(pkgPath, []byte(pkgContent), 0644); err != nil {
			return fmt.Errorf("failed to write package doc: %w", err)
		}
	}

	fmt.Printf("Generated markdown documentation in %s\n", config.OutputDir)
	return nil
}

func generateMarkdownREADME(config *DocConfig, docs []PackageDocs) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# %s\n\n", config.Title))
	b.WriteString(fmt.Sprintf("**Version:** %s\n\n", config.Version))
	b.WriteString(fmt.Sprintf("**Generated:** %s\n\n", time.Now().Format("2006-01-02")))
	b.WriteString("## Packages\n\n")

	for _, pkg := range docs {
		b.WriteString(fmt.Sprintf("- [%s](./%s.md) - %s\n", pkg.Name, pkg.Name, strings.Split(pkg.Doc, "\n")[0]))
	}

	b.WriteString("\n## Overview\n\n")
	b.WriteString("This documentation covers the Cline Go CLI internal packages and APIs.\n\n")
	b.WriteString("## Quick Links\n\n")
	b.WriteString("- [Commands](./cmd/)\n")
	b.WriteString("- [Internal Packages](./internal/)\n")
	b.WriteString("- [Public API](./pkg/)\n")

	return b.String()
}

func generateMarkdownPackage(config *DocConfig, pkg PackageDocs) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# Package %s\n\n", pkg.Name))
	b.WriteString(fmt.Sprintf("```go\nimport \"%s\"\n```\n\n", pkg.ImportPath))

	if pkg.Doc != "" {
		b.WriteString(pkg.Doc)
		b.WriteString("\n\n")
	}

	if len(pkg.Constants) > 0 {
		b.WriteString("## Constants\n\n")
		for _, c := range pkg.Constants {
			b.WriteString(fmt.Sprintf("### %s\n\n", c.Name))
			if c.Doc != "" {
				b.WriteString(c.Doc)
				b.WriteString("\n\n")
			}
			b.WriteString(fmt.Sprintf("```go\nconst %s = %s\n```\n\n", c.Name, c.Value))
		}
	}

	if len(pkg.Variables) > 0 {
		b.WriteString("## Variables\n\n")
		for _, v := range pkg.Variables {
			b.WriteString(fmt.Sprintf("### %s\n\n", v.Name))
			if v.Doc != "" {
				b.WriteString(v.Doc)
				b.WriteString("\n\n")
			}
		}
	}

	if len(pkg.Types) > 0 {
		b.WriteString("## Types\n\n")
		for _, t := range pkg.Types {
			b.WriteString(fmt.Sprintf("### %s\n\n", t.Name))
			if t.Doc != "" {
				b.WriteString(t.Doc)
				b.WriteString("\n\n")
			}

			if len(t.Methods) > 0 {
				b.WriteString("#### Methods\n\n")
				for _, m := range t.Methods {
					b.WriteString(fmt.Sprintf("- `%s` - %s\n", m.Name, strings.Split(m.Doc, "\n")[0]))
				}
				b.WriteString("\n")
			}
		}
	}

	if len(pkg.Functions) > 0 {
		b.WriteString("## Functions\n\n")
		for _, f := range pkg.Functions {
			b.WriteString(fmt.Sprintf("### %s\n\n", f.Name))
			if f.Doc != "" {
				b.WriteString(f.Doc)
				b.WriteString("\n\n")
			}
			b.WriteString(fmt.Sprintf("```go\n%s\n```\n\n", f.Signature))
		}
	}

	return b.String()
}

func generateHTMLDocs(config *DocConfig, docs []PackageDocs) error {
	tmpl, err := template.New("docs").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}

	data := struct {
		Config   *DocConfig
		Packages []PackageDocs
		Generated string
	}{
		Config:    config,
		Packages:  docs,
		Generated: time.Now().Format("2006-01-02"),
	}

	indexPath := filepath.Join(config.OutputDir, "index.html")
	f, err := os.Create(indexPath)
	if err != nil {
		return fmt.Errorf("failed to create index.html: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	fmt.Printf("Generated HTML documentation in %s\n", config.OutputDir)
	return nil
}

const htmlTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>{{.Config.Title}}</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; max-width: 1200px; margin: 0 auto; padding: 20px; }
        h1 { color: #333; border-bottom: 2px solid #eee; padding-bottom: 10px; }
        h2 { color: #555; margin-top: 30px; }
        h3 { color: #666; }
        .package { background: #f8f9fa; padding: 15px; margin: 10px 0; border-radius: 5px; }
        .meta { color: #666; font-size: 0.9em; }
        pre { background: #f4f4f4; padding: 10px; border-radius: 3px; overflow-x: auto; }
        code { background: #f4f4f4; padding: 2px 5px; border-radius: 3px; font-size: 0.9em; }
        a { color: #0066cc; text-decoration: none; }
        a:hover { text-decoration: underline; }
        .toc { background: #f8f9fa; padding: 15px; border-radius: 5px; margin: 20px 0; }
    </style>
</head>
<body>
    <h1>{{.Config.Title}}</h1>
    <p class="meta">Version: {{.Config.Version}} | Generated: {{.Generated}}</p>
    
    <div class="toc">
        <h2>Table of Contents</h2>
        <ul>
            {{range .Packages}}
            <li><a href="#{{.Name}}">{{.Name}}</a></li>
            {{end}}
        </ul>
    </div>

    {{range .Packages}}
    <div class="package" id="{{.Name}}">
        <h2>Package {{.Name}}</h2>
        <p><code>import "{{.ImportPath}}"</code></p>
        {{if .Doc}}<p>{{.Doc}}</p>{{end}}
        
        {{if .Types}}
        <h3>Types</h3>
        {{range .Types}}
        <h4>{{.Name}}</h4>
        {{if .Doc}}<p>{{.Doc}}</p>{{end}}
        {{end}}
        {{end}}
        
        {{if .Functions}}
        <h3>Functions</h3>
        {{range .Functions}}
        <p><code>{{.Name}}</code></p>
        {{if .Doc}}<p>{{.Doc}}</p>{{end}}
        {{end}}
        {{end}}
    </div>
    {{end}}
</body>
</html>
`

// getFuncSignature extracts the function signature from an AST FuncDecl.
func getFuncSignature(fn *ast.FuncDecl) (string, error) {
	// Create a minimal file set for formatting
	fset := token.NewFileSet()
	
	// Create a copy of just the function signature (without body)
	sig := &ast.FuncDecl{
		Doc:  fn.Doc,
		Recv: fn.Recv,
		Name: fn.Name,
		Type: fn.Type,
	}
	
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, sig); err != nil {
		// Fallback: return just the function name
		return fn.Name.Name + "(...)", err
	}
	
	return buf.String(), nil
}

func main() {
	config := ParseFlags()

	if err := GenerateDocs(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
