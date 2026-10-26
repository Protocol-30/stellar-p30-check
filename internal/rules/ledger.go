package rules

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/Protocol-30/stellar-p30-check/internal/deps"
	"github.com/Protocol-30/stellar-p30-check/internal/report"
	"github.com/Protocol-30/stellar-p30-check/internal/walker"
	treesitter "github.com/tree-sitter/go-tree-sitter"
	javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

const ledgerCAP88 = "https://github.com/stellar/stellar-protocol/blob/1ae39233055a0fb94bb737650a39ede8d77d924c/core/cap-0088.md"
const ledgerSchema = "https://github.com/stellar/stellar-xdr/blob/4f524bbac80c781c06e4fb5fc93a2d0d331d6705/Stellar-ledger.x"

// AnalyzeLedgers recognizes a bounded syntax subset, never target execution.
// Findings describe decoder risk or timing assumptions, not runtime failures.
func AnalyzeLedgers(files []walker.File, dependencies []deps.Dependency) ([]report.Finding, []string, error) {
	findings := []report.Finding{}
	notes := []string{}
	ordered := slices.Clone(files)
	slices.SortFunc(ordered, func(a, b walker.File) int { return strings.Compare(a.Path, b.Path) })
	for _, f := range ordered {
		var got []report.Finding
		var warnings []string
		var err error
		switch f.Kind {
		case "js_source", "ts_source":
			got, warnings, err = ledgerJS(f, files, dependencies)
		case "go_source":
			got, warnings, err = ledgerGo(f, files, dependencies)
		default:
			continue
		}
		notes = append(notes, warnings...)
		if err != nil {
			return nil, notes, fmt.Errorf("ledger analysis %q: %w", f.Path, err)
		}
		findings = append(findings, got...)
	}
	slices.Sort(notes)
	return findings, slices.Compact(notes), nil
}

func ledgerFinding(file walker.File, line int, evidence, ecosystem string, files []walker.File, dependencies []deps.Dependency) *report.Finding {
	scope, manifest := addressScope(file.Path, ecosystem, files)
	relevant := []deps.Dependency{}
	for _, d := range dependencies {
		if d.Path == manifest && d.Scope == scope && d.Ecosystem == ecosystem && (d.Name == jsSDK || d.Name == goSDK) {
			relevant = append(relevant, d)
		}
	}
	// All relevant records must agree on known modern capabilities to suppress
	// the risk; a sibling/parent dependency cannot mask a nested decoder.
	modern := len(relevant) > 0
	legacyResolved := len(relevant) > 0
	for _, d := range relevant {
		row, known := matrix[d.Ecosystem+":"+d.Name+":"+d.Version]
		if !known || row.cap88 != supported || d.Alias != "" {
			modern = false
		}
		if !known || row.cap88 != unsupported || !d.Effective || len(d.Uncertainty) != 0 || d.Alias != "" || d.Resolution != "lockfile" || d.Source == "" {
			legacyResolved = false
		}
	}
	if modern {
		return nil
	}
	confidence := report.ConfidenceLow
	detail := "Selected decoder capability is unresolved; inspect the actual build/lock and post-upgrade input requirements."
	if legacyResolved {
		confidence = report.ConfidenceHigh
		detail = "Nominal scoped lock metadata selects an exact official release whose CAP-88 decoder capability is verified missing."
	}
	return &report.Finding{RuleID: "P30-XDR-001", Severity: report.SeverityMedium, Confidence: confidence, Path: file.Path, Line: line,
		Message:     "Raw ledger XDR decoding needs CAP-88 millisecond union support; review the decoder used for post-upgrade inputs.",
		Evidence:    fmt.Sprintf("%s; scope=%q; manifest=%q; %s", evidence, scope, manifest, detail),
		Remediation: "Use verified CAP-88-capable XDR definitions for post-upgrade LedgerHeader/LedgerCloseMeta inputs. Preserve historical decoding needs. Static use and metadata do not prove that an input is post-upgrade or that a runtime failure occurs.", References: []string{ledgerCAP88, ledgerSchema, js164, go080}}
}

func ledgerJS(f walker.File, files []walker.File, dependencies []deps.Dependency) ([]report.Finding, []string, error) {
	out := []report.Finding{}
	p := treesitter.NewParser()
	defer p.Close()
	language := treesitter.NewLanguage(javascript.Language())
	switch path.Ext(f.Path) {
	case ".ts":
		language = treesitter.NewLanguage(typescript.LanguageTypescript())
	case ".tsx":
		language = treesitter.NewLanguage(typescript.LanguageTSX())
	}
	if err := p.SetLanguage(language); err != nil {
		return nil, nil, err
	}
	p.SetTimeoutMicros(2_000_000)
	tree := p.Parse(f.Data, nil)
	if tree == nil {
		return nil, nil, fmt.Errorf("source parsing exhausted its two-second budget")
	}
	defer tree.Close()
	root := tree.RootNode()
	nodes := []*treesitter.Node{}
	stack := []*treesitter.Node{root}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		nodes = append(nodes, n)
		if len(nodes) > jsAddressNodeLimit {
			return nil, nil, fmt.Errorf("source AST exceeds %d nodes", jsAddressNodeLimit)
		}
		for i := n.ChildCount(); i > 0; i-- {
			stack = append(stack, n.Child(i-1))
		}
	}
	if root.HasError() {
		return out, []string{fmt.Sprintf("Ledger analysis skipped %q because its source syntax is unsupported or malformed.", f.Path)}, nil
	}
	a := jsAddressAnalyzer{data: f.Data, bindings: map[string]string{}}
	// Only named ESM xdr imports; namespace/CommonJS and indirect aliases are
	// deliberately outside this increment's decoder-use proof.
	for _, stmt := range jsNamed(root) {
		if stmt.Kind() != "import_statement" || jsHasToken(stmt, "type") {
			continue
		}
		source, ok := a.stringLiteral(stmt.ChildByFieldName("source"))
		if !ok || (source != jsSDK && source != jsSDK+"/base") {
			continue
		}
		for _, clause := range jsNamed(stmt) {
			if clause.Kind() != "import_clause" {
				continue
			}
			for _, imports := range jsNamed(clause) {
				if imports.Kind() != "named_imports" {
					continue
				}
				for _, s := range jsNamed(imports) {
					if s.Kind() != "import_specifier" || jsHasToken(s, "type") || a.text(s.ChildByFieldName("name")) != "xdr" {
						continue
					}
					alias := s.ChildByFieldName("alias")
					if alias == nil {
						alias = s.ChildByFieldName("name")
					}
					if jsIsIdentifier(alias) {
						a.bindings[a.text(alias)] = "xdr"
					}
				}
			}
		}
	}
	a.rejectUnsafeBindings(root, nodes)
	// Duplicate import bindings are semantically ambiguous even when the
	// syntax grammar accepts them; never borrow an SDK identity in that case.
	for local := range a.bindings {
		count := 0
		for _, n := range nodes {
			if n.Kind() == "import_specifier" {
				alias := n.ChildByFieldName("alias")
				if alias == nil {
					alias = n.ChildByFieldName("name")
				}
				if a.text(alias) == local {
					count++
				}
			}
			if n.Kind() == "namespace_import" {
				for _, id := range jsNamed(n) {
					if a.text(id) == local {
						count++
					}
				}
			}
			if n.Kind() == "import_clause" {
				for _, id := range jsNamed(n) {
					if jsIsIdentifier(id) && a.text(id) == local {
						count++
					}
				}
			}
		}
		if count != 1 {
			delete(a.bindings, local)
		}
	}
	for _, stmt := range jsNamed(root) {
		if stmt.Kind() == "export_statement" {
			stmt = stmt.ChildByFieldName("declaration")
		}
		if stmt == nil || stmt.Kind() != "function_declaration" {
			continue
		}
		params, valid := a.parameters(stmt)
		if !valid {
			continue
		}
		shadow := false
		for _, name := range params {
			if a.bindings[name] != "" {
				shadow = true
			}
		}
		if a.bindings[a.text(stmt.ChildByFieldName("name"))] != "" {
			shadow = true
		}
		if shadow {
			continue
		}
		body := jsNamed(stmt.ChildByFieldName("body"))
		if len(body) != 1 || body[0].Kind() != "return_statement" {
			continue
		}
		returned := jsNamed(body[0])
		if len(returned) != 1 {
			continue
		}
		call := jsUnwrap(returned[0])
		if call == nil || call.Kind() != "call_expression" || call.ChildByFieldName("optional_chain") != nil {
			continue
		}
		callee := call.ChildByFieldName("function")
		if callee == nil || callee.Kind() != "member_expression" || a.text(callee.ChildByFieldName("property")) != "fromXDR" {
			continue
		}
		typ := callee.ChildByFieldName("object")
		if typ == nil || typ.Kind() != "member_expression" {
			continue
		}
		kind := a.text(typ.ChildByFieldName("property"))
		if kind != "LedgerHeader" && kind != "LedgerCloseMeta" {
			continue
		}
		binding := typ.ChildByFieldName("object")
		if !jsIsIdentifier(binding) || a.bindings[a.text(binding)] != "xdr" {
			continue
		}
		args := jsNamed(call.ChildByFieldName("arguments"))
		if len(args) < 1 || len(args) > 2 || !jsIsIdentifier(args[0]) || !jsStringContains(params, a.text(args[0])) {
			continue
		}
		if len(args) == 2 {
			format, ok := a.stringLiteral(args[1])
			if !ok || (format != "base64" && format != "hex" && format != "raw") {
				continue
			}
		}
		if finding := ledgerFinding(f, int(call.StartPosition().Row)+1, a.text(call), "npm", files, dependencies); finding != nil {
			out = append(out, *finding)
		}
	}
	return out, nil, nil
}

func ledgerGo(f walker.File, files []walker.File, dependencies []deps.Dependency) ([]report.Finding, []string, error) {
	out := []report.Finding{}
	fs := token.NewFileSet()
	tree, err := parser.ParseFile(fs, f.Path, f.Data, parser.SkipObjectResolution)
	if err != nil {
		return out, []string{fmt.Sprintf("Ledger analysis skipped %q because its Go syntax is unsupported or malformed.", f.Path)}, nil
	}
	bindings := map[string]string{}
	blocked := map[string]bool{}
	for _, i := range tree.Imports {
		value, _ := strconv.Unquote(i.Path.Value)
		name := path.Base(value)
		if i.Name != nil {
			name = i.Name.Name
		}
		if name != "_" && name != "." {
			bindings[name] = value
		}
	}
	count := 0
	over := false
	ast.Inspect(tree, func(n ast.Node) bool {
		if n == nil || over {
			return false
		}
		count++
		if count > goAddressNodeLimit {
			over = true
			return false
		}
		switch x := n.(type) {
		case *ast.Field:
			for _, id := range x.Names {
				blocked[id.Name] = true
			}
		case *ast.ValueSpec:
			for _, id := range x.Names {
				blocked[id.Name] = true
			}
		case *ast.FuncDecl:
			blocked[x.Name.Name] = true
		case *ast.AssignStmt:
			for _, left := range x.Lhs {
				if id, ok := left.(*ast.Ident); ok {
					blocked[id.Name] = true
				}
			}
		}
		return true
	})
	if over {
		return nil, nil, fmt.Errorf("Go source syntax tree exceeds %d nodes", goAddressNodeLimit)
	}
	ctx := goAddressContext{imports: bindings, blocked: blocked}
	xdrType := func(e ast.Expr) bool {
		return ctx.packageSelector(e, "github.com/stellar/go-stellar-sdk/xdr", "LedgerHeader") || ctx.packageSelector(e, "github.com/stellar/go-stellar-sdk/xdr", "LedgerCloseMeta")
	}
	for _, decl := range tree.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Body == nil || fn.Type.TypeParams != nil {
			continue
		}
		// Pointer-parameter decoding is a direct, typed use of the official decoder.
		targets := map[string]bool{}
		headers := map[string]bool{}
		for _, field := range fn.Type.Params.List {
			if ptr, ok := field.Type.(*ast.StarExpr); ok && xdrType(ptr.X) {
				for _, id := range field.Names {
					targets[id.Name] = true
				}
			}
			if ctx.packageSelector(field.Type, "github.com/stellar/go-stellar-sdk/xdr", "LedgerHeader") {
				for _, id := range field.Names {
					headers[id.Name] = true
				}
			}
		}
		if len(fn.Body.List) != 1 {
			continue
		}
		ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 1 {
			continue
		}
		if call, ok := ret.Results[0].(*ast.CallExpr); ok && len(call.Args) == 2 && call.Ellipsis == token.NoPos && ctx.packageSelector(call.Fun, "github.com/stellar/go-stellar-sdk/xdr", "SafeUnmarshalBase64") {
			if id, ok := call.Args[1].(*ast.Ident); ok && targets[id.Name] {
				if got := ledgerFinding(f, fs.Position(call.Pos()).Line, string(f.Data[fs.Position(call.Pos()).Offset:fs.Position(call.End()).Offset]), "go", files, dependencies); got != nil {
					out = append(out, *got)
				}
			}
		}
		// Exact scalar comparison on typed LedgerHeader.scpValue.closeTime fields.
		// No names like "timestamp" or unrelated Unix clocks are treated as evidence.
		comparison, ok := ret.Results[0].(*ast.BinaryExpr)
		if !ok || (comparison.Op != token.GTR && comparison.Op != token.LSS) {
			continue
		}
		fieldRoot := func(e ast.Expr) string {
			close, ok := e.(*ast.SelectorExpr)
			if !ok || close.Sel.Name != "CloseTime" {
				return ""
			}
			value, ok := close.X.(*ast.SelectorExpr)
			if !ok || value.Sel.Name != "ScpValue" {
				return ""
			}
			id, ok := value.X.(*ast.Ident)
			if !ok || !headers[id.Name] {
				return ""
			}
			return id.Name
		}
		left, right := fieldRoot(comparison.X), fieldRoot(comparison.Y)
		if left == "" || right == "" || left == right {
			continue
		}
		out = append(out, report.Finding{RuleID: "P30-TS-001", Severity: report.SeverityInfo, Confidence: report.ConfidenceLow, Path: f.Path, Line: fs.Position(comparison.Pos()).Line, Message: "A strict ordering comparison uses whole-second ledger close times; review assumptions about distinct consecutive seconds.", Evidence: string(f.Data[fs.Position(comparison.Pos()).Offset:fs.Position(comparison.End()).Offset]), Remediation: "CAP-88 permits consecutive ledgers in the same whole second. Use closeTimeMs when millisecond ordering is required, or ledger sequence for ledger ordering. A scalar comparison may be intentional; this low-confidence clue does not establish a bug or change time-bound semantics.", References: []string{ledgerCAP88}})
	}
	return out, nil, nil
}
