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

// main renders the reproducible documentation artifacts for this module.
func main() {
	if err := run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	fmt.Println("✔ Examples generated in ./examples/")
}

// run regenerates standalone examples from the documented API in a deterministic order.
func run() error {
	root, err := findRoot()
	if err != nil {
		return err
	}

	modPath, err := modulePath(root)
	if err != nil {
		return err
	}
	goVersion, err := moduleGoVersion(root)
	if err != nil {
		return err
	}
	rootReleaseVersion, err := moduleRequirementVersion(filepath.Join(root, "mailses", "go.mod"), modPath)
	if err != nil {
		return err
	}

	examplesDir := filepath.Join(root, "examples")
	if err := os.MkdirAll(examplesDir, 0o755); err != nil {
		return err
	}
	if err := ensureExamplesModule(examplesDir, modPath, goVersion, rootReleaseVersion); err != nil {
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
		{dir: filepath.Join(root, "mailsendgrid"), importPath: modPath + "/mailsendgrid", slugPrefix: "mailsendgrid"},
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

// ensureExamplesModule keeps generated examples aligned with the staged root release without discarding tooling dependencies.
func ensureExamplesModule(examplesDir, modPath, goVersion, rootReleaseVersion string) error {
	moduleFile := filepath.Join(examplesDir, "go.mod")
	if existing, err := os.ReadFile(moduleFile); err == nil {
		lines := strings.Split(string(existing), "\n")
		inRequire := false
		updatedRootRequirement := false
		for index, line := range lines {
			trimmed := strings.TrimSpace(line)
			fields := strings.Fields(trimmed)
			switch {
			case strings.HasPrefix(trimmed, "module "):
				lines[index] = "module " + modPath + "/examples"
			case strings.HasPrefix(trimmed, "go "):
				lines[index] = "go " + goVersion
			case len(fields) >= 3 && fields[0] == "require" && fields[1] == modPath:
				indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
				lines[index] = indent + "require " + modPath + " " + rootReleaseVersion
				updatedRootRequirement = true
			case len(fields) == 2 && fields[0] == "require" && fields[1] == "(":
				inRequire = true
			case inRequire && len(fields) == 1 && fields[0] == ")":
				inRequire = false
			case inRequire && len(fields) >= 2 && fields[0] == modPath:
				indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
				lines[index] = indent + modPath + " " + rootReleaseVersion
				updatedRootRequirement = true
			}
		}
		if !updatedRootRequirement {
			return fmt.Errorf("root requirement not found in %s", moduleFile)
		}
		return os.WriteFile(moduleFile, []byte(strings.Join(lines, "\n")), 0o644)
	} else if !os.IsNotExist(err) {
		return err
	}

	content := fmt.Sprintf(`module %s/examples

go %s

require (
	%s %s
	%s/mailses v0.0.0
)

replace %s => ..
replace %s/mailses => ../mailses
`, modPath, goVersion, modPath, rootReleaseVersion, modPath, modPath, modPath)

	return os.WriteFile(moduleFile, []byte(content), 0o644)
}

// FuncDoc captures the metadata needed to render one documented function.
type FuncDoc struct {
	Name       string
	Slug       string
	ImportPath string
	Examples   []Example
}

// Example captures an executable snippet and its source location.
type Example struct {
	Label string
	Line  int
	Code  string
}

// findRoot anchors generation to the library checkout even when invoked from a docs subdirectory.
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

// fileExists lets root discovery ignore candidate paths that are not present.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// modulePath reads the canonical import prefix so generated code never hard-codes a checkout identity.
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

// moduleGoVersion keeps generated examples on the same language baseline as the root module.
func moduleGoVersion(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "go ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "go ")), nil
		}
	}
	return "", fmt.Errorf("go version not found in go.mod")
}

// moduleRequirementVersion reads a direct requirement from a module file without resolving unavailable staged tags.
func moduleRequirementVersion(moduleFile, dependencyPath string) (string, error) {
	data, err := os.ReadFile(moduleFile)
	if err != nil {
		return "", err
	}
	inRequire := false
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(strings.SplitN(line, "//", 2)[0])
		switch {
		case len(fields) == 2 && fields[0] == "require" && fields[1] == "(":
			inRequire = true
		case inRequire && len(fields) == 1 && fields[0] == ")":
			inRequire = false
		case inRequire && len(fields) >= 2 && fields[0] == dependencyPath:
			return fields[1], nil
		case len(fields) >= 3 && fields[0] == "require" && fields[1] == dependencyPath:
			return fields[2], nil
		}
	}
	return "", fmt.Errorf("requirement for %s not found in %s", dependencyPath, moduleFile)
}

// collectExamplesFromDir discovers examples only on exported API surfaces that consumers can call.
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
				receiverName := extractReceiverName(fn)
				if receiverName != "" && !ast.IsExported(receiverName) {
					continue
				}
				slug := strings.ToLower(fn.Name.Name)
				if receiverName != "" {
					slug = strings.ToLower(receiverName + "_" + fn.Name.Name)
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

// extractReceiverName identifies method ownership so generated slugs remain collision-free across types.
func extractReceiverName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}
	return receiverTypeName(fn.Recv.List[0].Type)
}

// receiverTypeName unwraps pointer and generic receiver syntax to its declared type name.
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

// extractExamples parses the repository's Example labels while retaining source order for reproducible output.
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

// normalizeIndent removes shared documentation padding without changing relative code indentation.
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

// writeExample renders the first documented case as the canonical executable for an API slug.
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
	buf.WriteString("// main keeps this example executable so API drift fails during compilation.\n")
	buf.WriteString("func main() {\n")
	for _, line := range strings.Split(example.Code, "\n") {
		buf.WriteString("\t" + line + "\n")
	}
	buf.WriteString("}\n")

	return os.WriteFile(filepath.Join(dir, "main.go"), buf.Bytes(), 0o644)
}

// inferImports derives a sorted minimal import set so generated examples compile and remain stable.
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
	if strings.Contains(code, "os.") {
		add("os")
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
	if strings.Contains(code, "mailsendgrid.") {
		addSubpackageImport(importSet, importPath, "mailsendgrid")
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

// addSubpackageImport reuses the current package path when it already names the requested mail subpackage.
func addSubpackageImport(importSet map[string]bool, importPath, subpackage string) {
	if strings.HasSuffix(importPath, "/"+subpackage) {
		importSet[importPath] = true
		return
	}
	importSet[importPath+"/"+subpackage] = true
}

// rootModuleImport maps a mail subpackage import back to the module root for shared API references.
func rootModuleImport(importPath string) string {
	for _, suffix := range []string{"/mailfake", "/mailmailgun", "/maillog", "/mailpostmark", "/mailresend", "/mailsendgrid", "/mailses", "/mailsmtp"} {
		if strings.HasSuffix(importPath, suffix) {
			return strings.TrimSuffix(importPath, suffix)
		}
	}
	return importPath
}
