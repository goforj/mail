//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	if err := run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	fmt.Println("✔ Examples generated in ./examples/")
}

func run() error {
	root, err := findRoot()
	if err != nil {
		return err
	}

	modPath, err := modulePath(root)
	if err != nil {
		return err
	}

	examplesDir := filepath.Join(root, "examples")
	if err := os.MkdirAll(examplesDir, 0o755); err != nil {
		return err
	}
	if err := ensureExamplesModule(examplesDir, modPath); err != nil {
		return err
	}

	targets := []struct {
		dir        string
		importPath string
		slugPrefix string
	}{
		{dir: root, importPath: modPath},
		{dir: filepath.Join(root, "mailfake"), importPath: modPath + "/mailfake", slugPrefix: "mailfake"},
		{dir: filepath.Join(root, "mailmailgun"), importPath: modPath + "/mailmailgun", slugPrefix: "mailmailgun"},
		{dir: filepath.Join(root, "maillog"), importPath: modPath + "/maillog", slugPrefix: "maillog"},
		{dir: filepath.Join(root, "mailpostmark"), importPath: modPath + "/mailpostmark", slugPrefix: "mailpostmark"},
		{dir: filepath.Join(root, "mailresend"), importPath: modPath + "/mailresend", slugPrefix: "mailresend"},
		{dir: filepath.Join(root, "mailses"), importPath: modPath + "/mailses", slugPrefix: "mailses"},
		{dir: filepath.Join(root, "mailsmtp"), importPath: modPath + "/mailsmtp", slugPrefix: "mailsmtp"},
	}

	funcs := map[string]*FuncDoc{}
	for _, target := range targets {
		if !fileExists(target.dir) {
			continue
		}
		if err := collectExamplesFromDir(funcs, target.dir, target.importPath, target.slugPrefix); err != nil {
			return err
		}
	}

	for _, fd := range funcs {
		sort.Slice(fd.Examples, func(i, j int) bool { return fd.Examples[i].Line < fd.Examples[j].Line })
		if err := writeExample(examplesDir, fd); err != nil {
			return err
		}
	}

	return nil
}

func ensureExamplesModule(examplesDir, modPath string) error {
	content := fmt.Sprintf(`module %s/examples

go 1.26.1

require (
	%s v0.0.0
	%s/mailses v0.0.0
)

replace %s => ..
replace %s/mailses => ../mailses
`, modPath, modPath, modPath, modPath, modPath)

	return os.WriteFile(filepath.Join(examplesDir, "go.mod"), []byte(content), 0o644)
}

type FuncDoc struct {
	Name       string
	Slug       string
	ImportPath string
	Examples   []Example
}

type Example struct {
	Label string
	Line  int
	Code  string
}

func findRoot() (string, error) {
	wd, _ := os.Getwd()
	for _, c := range []string{wd, filepath.Join(wd, ".."), filepath.Join(wd, "..", ".."), filepath.Join(wd, "..", "..", "..")} {
		c = filepath.Clean(c)
		if fileExists(filepath.Join(c, "go.mod")) && fileExists(filepath.Join(c, "README.md")) && fileExists(filepath.Join(c, "manager.go")) {
			return c, nil
		}
	}
	return "", fmt.Errorf("could not find project root")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func modulePath(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", fmt.Errorf("module path not found in go.mod")
}

func collectExamplesFromDir(out map[string]*FuncDoc, dir, importPath, slugPrefix string) error {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Doc == nil || !ast.IsExported(fn.Name.Name) {
					continue
				}
				slug := strings.ToLower(fn.Name.Name)
				if recv := extractReceiverName(fn); recv != "" {
					slug = strings.ToLower(recv + "_" + fn.Name.Name)
				}
				if slugPrefix != "" {
					slug = slugPrefix + "_" + slug
				}
				item := &FuncDoc{
					Name:       fn.Name.Name,
					Slug:       slug,
					ImportPath: importPath,
					Examples:   extractExamples(fset, fn.Doc),
				}
				if len(item.Examples) == 0 {
					continue
				}
				if existing, ok := out[slug]; ok {
					existing.Examples = append(existing.Examples, item.Examples...)
					continue
				}
				out[slug] = item
			}
		}
	}
	return nil
}

func extractReceiverName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}
	return receiverTypeName(fn.Recv.List[0].Type)
}

func receiverTypeName(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.StarExpr:
		return receiverTypeName(v.X)
	case *ast.IndexExpr:
		return receiverTypeName(v.X)
	case *ast.IndexListExpr:
		return receiverTypeName(v.X)
	default:
		return ""
	}
}

type docLine struct {
	text string
	pos  token.Pos
}

func extractExamples(fset *token.FileSet, group *ast.CommentGroup) []Example {
	var examples []Example
	lines := make([]docLine, 0, len(group.List))
	for _, c := range group.List {
		line := strings.TrimPrefix(c.Text, "//")
		if strings.HasPrefix(line, " ") {
			line = line[1:]
		}
		if strings.HasPrefix(line, "\t") {
			line = line[1:]
		}
		lines = append(lines, docLine{
			text: line,
			pos:  c.Pos(),
		})
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line.text)
		if !strings.HasPrefix(strings.ToLower(trimmed), "example:") {
			continue
		}
		label := strings.TrimSpace(trimmed[len("Example:"):])
		var block []string
		for j := i + 1; j < len(lines); j++ {
			next := lines[j]
			nextTrimmed := strings.TrimSpace(next.text)
			if strings.HasPrefix(strings.ToLower(nextTrimmed), "example:") || strings.HasPrefix(nextTrimmed, "@group ") {
				break
			}
			if nextTrimmed == "" {
				if len(block) == 0 {
					continue
				}
				break
			}
			block = append(block, next.text)
		}
		if len(block) == 0 {
			continue
		}
		examples = append(examples, Example{
			Label: label,
			Line:  fset.Position(line.pos).Line,
			Code:  strings.Join(normalizeIndent(block), "\n"),
		})
	}
	return examples
}

func normalizeIndent(lines []string) []string {
	minIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := 0
		for indent < len(line) && (line[indent] == ' ' || line[indent] == '\t') {
			indent++
		}
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}
	if minIndent <= 0 {
		return lines
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			out = append(out, "")
			continue
		}
		out = append(out, line[minIndent:])
	}
	return out
}

func writeExample(examplesDir string, fd *FuncDoc) error {
	dir := filepath.Join(examplesDir, fd.Slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	example := fd.Examples[0]
	imports := inferImports(example.Code, fd.ImportPath)

	var buf bytes.Buffer
	buf.WriteString("package main\n\n")
	if len(imports) > 0 {
		buf.WriteString("import (\n")
		for _, imp := range imports {
			buf.WriteString("\t\"" + imp + "\"\n")
		}
		buf.WriteString(")\n\n")
	}
	buf.WriteString("func main() {\n")
	for _, line := range strings.Split(example.Code, "\n") {
		buf.WriteString("\t" + line + "\n")
	}
	buf.WriteString("}\n")

	return os.WriteFile(filepath.Join(dir, "main.go"), buf.Bytes(), 0o644)
}

func inferImports(code, importPath string) []string {
	importSet := map[string]bool{}
	rootImportPath := rootModuleImport(importPath)
	add := func(path string) {
		if path != "" {
			importSet[path] = true
		}
	}

	add(importPath)
	if strings.Contains(code, "mail.") {
		add(rootImportPath)
	}
	if strings.Contains(code, "context.") {
		add("context")
	}
	if strings.Contains(code, "fmt.") {
		add("fmt")
	}
	if strings.Contains(code, "errors.") {
		add("errors")
	}
	if strings.Contains(code, "time.") {
		add("time")
	}
	if strings.Contains(code, "bytes.") {
		add("bytes")
	}
	if strings.Contains(code, "strings.") {
		add("strings")
	}
	if strings.Contains(code, "mailfake.") {
		addSubpackageImport(importSet, importPath, "mailfake")
	}
	if strings.Contains(code, "mailmailgun.") {
		addSubpackageImport(importSet, importPath, "mailmailgun")
	}
	if strings.Contains(code, "maillog.") {
		addSubpackageImport(importSet, importPath, "maillog")
	}
	if strings.Contains(code, "mailpostmark.") {
		addSubpackageImport(importSet, importPath, "mailpostmark")
	}
	if strings.Contains(code, "mailresend.") {
		addSubpackageImport(importSet, importPath, "mailresend")
	}
	if strings.Contains(code, "mailses.") {
		addSubpackageImport(importSet, importPath, "mailses")
	}
	if strings.Contains(code, "mailsmtp.") {
		addSubpackageImport(importSet, importPath, "mailsmtp")
	}

	imports := make([]string, 0, len(importSet))
	for imp := range importSet {
		imports = append(imports, imp)
	}
	sort.Strings(imports)
	return imports
}

func addSubpackageImport(importSet map[string]bool, importPath, subpackage string) {
	if strings.HasSuffix(importPath, "/"+subpackage) {
		importSet[importPath] = true
		return
	}
	importSet[importPath+"/"+subpackage] = true
}

func rootModuleImport(importPath string) string {
	for _, suffix := range []string{"/mailfake", "/mailmailgun", "/maillog", "/mailpostmark", "/mailresend", "/mailses", "/mailsmtp"} {
		if strings.HasSuffix(importPath, suffix) {
			return strings.TrimSuffix(importPath, suffix)
		}
	}
	return importPath
}
