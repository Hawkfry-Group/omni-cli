package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

type openAPIDoc struct {
	Paths map[string]map[string]openAPIOperation `json:"paths"`
}

type openAPIOperation struct {
	OperationID string   `json:"operationId"`
	Summary     string   `json:"summary"`
	Tags        []string `json:"tags"`
}

type endpoint struct {
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	OperationID string   `json:"operation_id"`
	Summary     string   `json:"summary,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type reportEndpoint struct {
	endpoint
	SourceWrappers []string `json:"source_wrappers,omitempty"`
}

type coverageReport struct {
	GeneratedAt          string           `json:"generated_at"`
	TotalEndpoints       int              `json:"total_endpoints"`
	TypedCoveredCount    int              `json:"typed_covered_count"`
	TypedMissingCount    int              `json:"typed_missing_count"`
	TypedCoveragePercent float64          `json:"typed_coverage_percent"`
	TypedCovered         []reportEndpoint `json:"typed_covered_endpoints"`
	TypedMissing         []endpoint       `json:"typed_missing_endpoints"`
	CLIWrappersUsed      []string         `json:"cli_wrappers_used"`
	WrappersWithoutMap   []string         `json:"wrappers_without_mapping"`
}

var openAPIMethods = map[string]struct{}{
	"get":     {},
	"post":    {},
	"put":     {},
	"patch":   {},
	"delete":  {},
	"options": {},
	"head":    {},
	"trace":   {},
}

func main() {
	openapiPath := flag.String("openapi", "api/openapi.json", "OpenAPI JSON path")
	clientPath := flag.String("client", "internal/client/client.go", "Client wrapper file path")
	cliDir := flag.String("cli-dir", "internal/cli", "CLI source directory")
	outMD := flag.String("out-md", "docs/endpoint-coverage.md", "Markdown report path")
	outJSON := flag.String("out-json", "docs/endpoint-coverage.json", "JSON report path")
	flag.Parse()

	allEndpoints, opIndex, err := loadOpenAPI(*openapiPath)
	if err != nil {
		failf("load openapi: %v", err)
	}

	clientWrappers, err := listClientWrappers(*clientPath)
	if err != nil {
		failf("list client wrappers: %v", err)
	}

	usedWrappers, err := scanCLIWrappers(*cliDir, clientWrappers)
	if err != nil {
		failf("scan cli wrappers: %v", err)
	}

	wrapperToGenerated, err := mapWrappersToGeneratedOps(*clientPath, usedWrappers)
	if err != nil {
		failf("map wrappers to generated ops: %v", err)
	}

	typedCoveredOps := make(map[string]struct{})
	opToWrappers := make(map[string]map[string]struct{})
	missingWrapperMap := make([]string, 0)

	for _, wrapper := range usedWrappers {
		genOps := wrapperToGenerated[wrapper]
		if len(genOps) == 0 {
			missingWrapperMap = append(missingWrapperMap, wrapper)
			continue
		}
		for _, genOp := range genOps {
			opID := generatedNameToOperationID(genOp)
			if opID == "" {
				continue
			}
			if _, ok := opIndex[opID]; !ok {
				continue
			}
			typedCoveredOps[opID] = struct{}{}
			if _, ok := opToWrappers[opID]; !ok {
				opToWrappers[opID] = make(map[string]struct{})
			}
			opToWrappers[opID][wrapper] = struct{}{}
		}
	}

	typedCovered := make([]reportEndpoint, 0)
	typedMissing := make([]endpoint, 0)

	for _, ep := range allEndpoints {
		if _, ok := typedCoveredOps[ep.OperationID]; ok {
			typedCovered = append(typedCovered, reportEndpoint{
				endpoint:       ep,
				SourceWrappers: sortedKeys(opToWrappers[ep.OperationID]),
			})
			continue
		}
		typedMissing = append(typedMissing, ep)
	}

	sort.Slice(typedCovered, func(i, j int) bool {
		if typedCovered[i].Path != typedCovered[j].Path {
			return typedCovered[i].Path < typedCovered[j].Path
		}
		return typedCovered[i].Method < typedCovered[j].Method
	})
	sort.Slice(typedMissing, func(i, j int) bool {
		if typedMissing[i].Path != typedMissing[j].Path {
			return typedMissing[i].Path < typedMissing[j].Path
		}
		return typedMissing[i].Method < typedMissing[j].Method
	})
	sort.Strings(missingWrapperMap)

	total := len(allEndpoints)
	covered := len(typedCovered)
	pct := 0.0
	if total > 0 {
		pct = (float64(covered) / float64(total)) * 100
	}

	report := coverageReport{
		GeneratedAt:          time.Now().UTC().Format(time.RFC3339),
		TotalEndpoints:       total,
		TypedCoveredCount:    covered,
		TypedMissingCount:    len(typedMissing),
		TypedCoveragePercent: pct,
		TypedCovered:         typedCovered,
		TypedMissing:         typedMissing,
		CLIWrappersUsed:      usedWrappers,
		WrappersWithoutMap:   missingWrapperMap,
	}

	if err := writeJSON(*outJSON, report); err != nil {
		failf("write json report: %v", err)
	}
	if err := writeMarkdown(*outMD, report); err != nil {
		failf("write markdown report: %v", err)
	}

	fmt.Printf(
		"endpoint coverage report generated: total=%d typed_covered=%d typed_missing=%d coverage=%.1f%%\n",
		report.TotalEndpoints,
		report.TypedCoveredCount,
		report.TypedMissingCount,
		report.TypedCoveragePercent,
	)
}

func loadOpenAPI(path string) ([]endpoint, map[string]endpoint, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	var doc openAPIDoc
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, nil, err
	}

	out := make([]endpoint, 0)
	index := make(map[string]endpoint)
	for p, methods := range doc.Paths {
		for method, op := range methods {
			if _, ok := openAPIMethods[strings.ToLower(method)]; !ok {
				continue
			}
			opID := strings.TrimSpace(op.OperationID)
			if opID == "" {
				continue
			}
			ep := endpoint{
				Method:      strings.ToUpper(method),
				Path:        p,
				OperationID: opID,
				Summary:     strings.TrimSpace(op.Summary),
				Tags:        append([]string(nil), op.Tags...),
			}
			out = append(out, ep)
			index[opID] = ep
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Method < out[j].Method
	})

	return out, index, nil
}

func scanCLIWrappers(cliDir string, knownWrappers map[string]struct{}) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(cliDir, "*.go"))
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{})
	fset := token.NewFileSet()

	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if _, ok := knownWrappers[sel.Sel.Name]; !ok {
				return true
			}
			set[sel.Sel.Name] = struct{}{}
			return true
		})
	}
	return sortedKeys(set), nil
}

func listClientWrappers(clientPath string) (map[string]struct{}, error) {
	out := make(map[string]struct{})
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, clientPath, nil, 0)
	if err != nil {
		return nil, err
	}
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Recv == nil {
			continue
		}
		if len(fd.Recv.List) != 1 {
			continue
		}
		if !isClientReceiver(fd.Recv.List[0].Type) {
			continue
		}
		out[fd.Name.Name] = struct{}{}
	}
	return out, nil
}

func mapWrappersToGeneratedOps(clientPath string, wrappers []string) (map[string][]string, error) {
	want := make(map[string]struct{}, len(wrappers))
	for _, w := range wrappers {
		want[w] = struct{}{}
	}

	out := make(map[string][]string)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, clientPath, nil, 0)
	if err != nil {
		return nil, err
	}

	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Recv == nil || fd.Body == nil {
			continue
		}
		if len(fd.Recv.List) != 1 {
			continue
		}
		if !isClientReceiver(fd.Recv.List[0].Type) {
			continue
		}
		wrapperName := fd.Name.Name
		if _, ok := want[wrapperName]; !ok {
			continue
		}

		genSet := make(map[string]struct{})
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			inner, ok := sel.X.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			root, ok := inner.X.(*ast.Ident)
			if !ok || root.Name != "c" || inner.Sel.Name != "api" {
				return true
			}
			name := trimGeneratedMethodSuffix(sel.Sel.Name)
			if name != "" {
				genSet[name] = struct{}{}
			}
			return true
		})
		out[wrapperName] = sortedKeys(genSet)
	}

	return out, nil
}

func isClientReceiver(expr ast.Expr) bool {
	star, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}
	id, ok := star.X.(*ast.Ident)
	return ok && id.Name == "Client"
}

func trimGeneratedMethodSuffix(name string) string {
	switch {
	case strings.HasSuffix(name, "WithBodyWithResponse"):
		return strings.TrimSuffix(name, "WithBodyWithResponse")
	case strings.HasSuffix(name, "WithResponse"):
		return strings.TrimSuffix(name, "WithResponse")
	default:
		return name
	}
}

func generatedNameToOperationID(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	runes := []rune(name)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func writeJSON(path string, report coverageReport) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func writeMarkdown(path string, report coverageReport) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("# Endpoint Coverage Report\n\n")
	b.WriteString(fmt.Sprintf("- Generated: `%s`\n", report.GeneratedAt))
	b.WriteString(fmt.Sprintf("- Total OpenAPI endpoints: `%d`\n", report.TotalEndpoints))
	b.WriteString(fmt.Sprintf("- Typed CLI-covered endpoints: `%d`\n", report.TypedCoveredCount))
	b.WriteString(fmt.Sprintf("- Missing typed coverage: `%d`\n", report.TypedMissingCount))
	b.WriteString(fmt.Sprintf("- Typed coverage: `%.1f%%`\n", report.TypedCoveragePercent))
	b.WriteString("- Note: all missing typed endpoints are still callable via `omni api call`.\n\n")

	if len(report.WrappersWithoutMap) > 0 {
		b.WriteString("## Wrapper Mapping Gaps\n\n")
		b.WriteString("These CLI wrapper calls were detected but could not be mapped to a generated operation in `internal/client/client.go`:\n\n")
		for _, w := range report.WrappersWithoutMap {
			b.WriteString(fmt.Sprintf("- `%s`\n", w))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Missing Typed Coverage (Gap List)\n\n")
	b.WriteString("| Method | Path | Operation ID | Tags |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, ep := range report.TypedMissing {
		b.WriteString(fmt.Sprintf(
			"| `%s` | `%s` | `%s` | `%s` |\n",
			ep.Method, ep.Path, ep.OperationID, strings.Join(ep.Tags, ", "),
		))
	}
	b.WriteString("\n")

	b.WriteString("## Typed-Covered Endpoints\n\n")
	b.WriteString("| Method | Path | Operation ID | Source Wrapper(s) |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, ep := range report.TypedCovered {
		b.WriteString(fmt.Sprintf(
			"| `%s` | `%s` | `%s` | `%s` |\n",
			ep.Method, ep.Path, ep.OperationID, strings.Join(ep.SourceWrappers, ", "),
		))
	}
	b.WriteString("\n")

	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func sortedKeys[T any](set map[string]T) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func failf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
