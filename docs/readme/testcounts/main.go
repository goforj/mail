//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	testCountStart    = "<!-- test-count:embed:start -->"
	testCountEnd      = "<!-- test-count:embed:end -->"
	packageCoverStart = "<!-- package-coverage:embed:start -->"
	packageCoverEnd   = "<!-- package-coverage:embed:end -->"
)

func main() {
	if err := run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	fmt.Println("✔ Test badges and package coverage updated in README.md")
}

func run() error {
	root, err := findRoot()
	if err != nil {
		return err
	}

	unitCount, err := countRunEvents(root)
	if err != nil {
		return err
	}
	packageCoverage, err := collectPackageCoverage(root)
	if err != nil {
		return err
	}

	readmePath := filepath.Join(root, "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		return err
	}

	out, err := replaceSection(string(data), testCountStart, testCountEnd, renderBadges(unitCount))
	if err != nil {
		return err
	}
	out, err = replaceSection(out, packageCoverStart, packageCoverEnd, renderPackageCoverageBadges(packageCoverage))
	if err != nil {
		return err
	}

	return os.WriteFile(readmePath, []byte(out), 0o644)
}

type packageCoverage struct {
	Name    string
	Covered int
	Total   int
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

func countRunEvents(root string) (int, error) {
	cmd := exec.Command("go", "test", "./...", "-run", "Test|Example", "-count=1", "-json")
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"GOCACHE=/tmp/gocache",
		"GOMODCACHE=/tmp/gomodcache",
	)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("go test ./... -json: %w\n%s", err, out.String())
	}

	var total int
	dec := json.NewDecoder(bytes.NewReader(out.Bytes()))
	for dec.More() {
		var event struct {
			Action string `json:"Action"`
			Test   string `json:"Test"`
		}
		if err := dec.Decode(&event); err != nil {
			return 0, err
		}
		if event.Action == "run" && event.Test != "" {
			total++
		}
	}
	return total, nil
}

func collectPackageCoverage(root string) ([]packageCoverage, error) {
	coverFile := filepath.Join(root, "coverage.txt")
	cmd := exec.Command("bash", "scripts/coverage-codecov.sh")
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"GOCACHE=/tmp/gocache",
		"GOMODCACHE=/tmp/gomodcache",
	)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("bash scripts/coverage-codecov.sh: %w\n%s", err, out.String())
	}

	data, err := os.ReadFile(coverFile)
	if err != nil {
		return nil, err
	}

	totals := map[string]*packageCoverage{}
	for _, line := range strings.Split(string(data), "\n")[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 3 {
			return nil, fmt.Errorf("unexpected coverage line: %q", line)
		}
		fileAndRange := parts[0]
		colon := strings.Index(fileAndRange, ":")
		if colon < 0 {
			return nil, fmt.Errorf("unexpected coverage file segment: %q", line)
		}
		file := fileAndRange[:colon]
		numStmts, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, err
		}
		count, err := strconv.Atoi(parts[2])
		if err != nil {
			return nil, err
		}

		pkg := packageLabelForCoverageFile(file)
		entry := totals[pkg]
		if entry == nil {
			entry = &packageCoverage{Name: pkg}
			totals[pkg] = entry
		}
		entry.Total += numStmts
		if count > 0 {
			entry.Covered += numStmts
		}
	}

	order := []string{
		"mail",
		"mailfake",
		"maillog",
		"mailmailgun",
		"mailpostmark",
		"mailresend",
		"mailsendgrid",
		"mailses",
		"mailsmtp",
	}
	outCov := make([]packageCoverage, 0, len(order))
	for _, name := range order {
		if entry, ok := totals[name]; ok {
			outCov = append(outCov, *entry)
		}
	}
	return outCov, nil
}

func packageLabelForCoverageFile(file string) string {
	const (
		rootPrefix = "github.com/goforj/mail/"
		sesPrefix  = "github.com/goforj/mail/mailses/"
	)
	switch {
	case strings.HasPrefix(file, sesPrefix):
		return "mailses"
	case strings.HasPrefix(file, rootPrefix):
		rest := strings.TrimPrefix(file, rootPrefix)
		switch {
		case strings.HasPrefix(rest, "mailfake/"):
			return "mailfake"
		case strings.HasPrefix(rest, "maillog/"):
			return "maillog"
		case strings.HasPrefix(rest, "mailmailgun/"):
			return "mailmailgun"
		case strings.HasPrefix(rest, "mailpostmark/"):
			return "mailpostmark"
		case strings.HasPrefix(rest, "mailresend/"):
			return "mailresend"
		case strings.HasPrefix(rest, "mailsendgrid/"):
			return "mailsendgrid"
		case strings.HasPrefix(rest, "mailsmtp/"):
			return "mailsmtp"
		default:
			return "mail"
		}
	case strings.HasPrefix(file, "github.com/goforj/mail"):
		return "mail"
	default:
		return filepath.Dir(file)
	}
}

func renderBadges(unitCount int) string {
	return strings.Join([]string{
		fmt.Sprintf(`<img src="https://img.shields.io/badge/unit_tests-%d-brightgreen" alt="Unit tests (executed count)">`, unitCount),
	}, "\n")
}

func renderPackageCoverageBadges(items []packageCoverage) string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		percent := 0.0
		if item.Total > 0 {
			percent = (float64(item.Covered) / float64(item.Total)) * 100
		}
		lines = append(lines, fmt.Sprintf(`<img src="https://img.shields.io/badge/%s-%.1f%%25-4c9a2a" alt="%s coverage">`, badgeLabel(item.Name), percent, item.Name))
	}
	return "<br>\n" + strings.Join(lines, "\n")
}

func badgeLabel(name string) string {
	replacer := strings.NewReplacer(
		"/", "--",
		" ", "_",
	)
	return replacer.Replace(name)
}

func replaceSection(input, start, end, replacement string) (string, error) {
	si := strings.Index(input, start)
	ei := strings.Index(input, end)
	if si < 0 || ei < 0 || ei < si {
		return "", fmt.Errorf("missing marker pair %q ... %q", start, end)
	}
	var out strings.Builder
	out.WriteString(input[:si+len(start)])
	out.WriteString("\n")
	out.WriteString(replacement)
	out.WriteString("\n")
	out.WriteString(input[ei:])
	return out.String(), nil
}
