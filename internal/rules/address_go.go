package rules

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"

	"github.com/Protocol-30/stellar-p30-check/internal/walker"
)

const goAddressNodeLimit = 200000

const goAddressStrkeyPackage = "github.com/stellar/go-stellar-sdk/strkey"

// analyzeGoAddress recognizes only a deliberately small local-flow proof. It
// does not build or type-check a target, resolve imports, or run its regexes.
func analyzeGoAddress(file walker.File) ([]addressCandidate, []string, error) {
	findings := []addressCandidate{}
	notes := []string{}
	fset := token.NewFileSet()
	tree, err := parser.ParseFile(fset, file.Path, file.Data, parser.SkipObjectResolution)
	if err != nil {
		return findings, []string{fmt.Sprintf("Go address analysis skipped %q because its syntax is invalid or unsupported.", file.Path)}, nil
	}

	imports := map[string]string{}
	blocked := map[string]bool{}
	for _, spec := range tree.Imports {
		name, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		binding := name[strings.LastIndexByte(name, '/')+1:]
		if spec.Name != nil {
			binding = spec.Name.Name
		}
		if binding == "_" || binding == "." {
			continue
		}
		if previous, exists := imports[binding]; exists && previous != name {
			blocked[binding] = true
		}
		imports[binding] = name
	}

	nodes, exceeded := 0, false
	ast.Inspect(tree, func(node ast.Node) bool {
		if node == nil || exceeded {
			return false
		}
		nodes++
		if nodes > goAddressNodeLimit {
			exceeded = true
			return false
		}
		// A file-wide conservative block avoids mistaking any local or package
		// declaration with an imported/builtin spelling for the known symbol.
		block := func(id *ast.Ident) {
			if id != nil {
				blocked[id.Name] = true
			}
		}
		switch node := node.(type) {
		case *ast.FuncDecl:
			block(node.Name)
		case *ast.TypeSpec:
			block(node.Name)
		case *ast.ValueSpec:
			for _, name := range node.Names {
				block(name)
			}
		case *ast.Field:
			for _, name := range node.Names {
				block(name)
			}
		case *ast.AssignStmt:
			for _, lhs := range node.Lhs {
				if id, ok := lhs.(*ast.Ident); ok {
					// Assignment cannot replace imported packages in valid Go;
					// treating it as ambiguous also covers invalid/shadowed input.
					block(id)
				}
			}
		case *ast.RangeStmt:
			if id, ok := node.Key.(*ast.Ident); ok {
				block(id)
			}
			if id, ok := node.Value.(*ast.Ident); ok {
				block(id)
			}
		}
		return true
	})
	if exceeded {
		return findings, notes, fmt.Errorf("Go source syntax tree exceeds %d nodes", goAddressNodeLimit)
	}

	context := goAddressContext{imports: imports, blocked: blocked}
	if !context.hasStrkeyImport() || blocked["string"] {
		return findings, notes, nil
	}
	for _, declaration := range tree.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv != nil || function.Body == nil || function.Type.TypeParams != nil {
			continue
		}
		parameter := goAddressParameter(function)
		if parameter == "" {
			continue
		}
		statements := function.Body.List
		if len(statements) != 2 {
			continue
		}
		guard, ok := statements[0].(*ast.IfStmt)
		if !ok || guard.Init != nil || guard.Else != nil || len(guard.Body.List) != 1 {
			continue
		}
		result, ok := statements[1].(*ast.ReturnStmt)
		if !ok || len(result.Results) != 1 {
			continue
		}
		call, ok := goAddressUnparen(result.Results[0]).(*ast.CallExpr)
		if !ok {
			continue
		}
		api, boolean := context.wOnlySink(call, parameter)
		if api == "" || !context.rejects(guard.Body.List[0], boolean) || !context.purePredicate(guard.Cond, parameter) || context.truth(guard.Cond, parameter) != wTrue {
			continue
		}
		start, end := fset.Position(guard.Pos()), fset.Position(call.End())
		if start.Offset < 0 || end.Offset > len(file.Data) || start.Offset > end.Offset {
			continue
		}
		findings = append(findings, addressCandidate{
			path: file.Path, line: start.Line, parameter: parameter,
			api: api, ecosystem: "go", kind: addressWOnly,
			evidence: string(file.Data[start.Offset:end.Offset]),
		})
	}
	return findings, notes, nil
}

type goAddressContext struct {
	imports map[string]string
	blocked map[string]bool
}

func (c goAddressContext) hasStrkeyImport() bool {
	for binding, path := range c.imports {
		if path == goAddressStrkeyPackage && !c.blocked[binding] {
			return true
		}
	}
	return false
}

func goAddressParameter(function *ast.FuncDecl) string {
	if len(function.Type.Params.List) != 1 {
		return ""
	}
	field := function.Type.Params.List[0]
	typ, ok := field.Type.(*ast.Ident)
	if !ok || typ.Name != "string" || len(field.Names) != 1 || field.Names[0].Name == "_" {
		return ""
	}
	return field.Names[0].Name
}

func (c goAddressContext) packageSelector(expression ast.Expr, packagePath, member string) bool {
	selector, ok := goAddressUnparen(expression).(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != member {
		return false
	}
	identifier, ok := goAddressUnparen(selector.X).(*ast.Ident)
	return ok && !c.blocked[identifier.Name] && c.imports[identifier.Name] == packagePath
}

func (c goAddressContext) wOnlySink(call *ast.CallExpr, parameter string) (string, bool) {
	if call.Ellipsis.IsValid() {
		return "", false
	}
	for _, method := range []string{"DecodeMuxedContract", "IsValidMuxedContractAddress"} {
		if len(call.Args) == 1 && goAddressIdent(call.Args[0], parameter) && c.packageSelector(call.Fun, goAddressStrkeyPackage, method) {
			return "strkey." + method, method == "IsValidMuxedContractAddress"
		}
	}
	if len(call.Args) == 2 && c.packageSelector(call.Fun, goAddressStrkeyPackage, "Decode") && c.packageSelector(call.Args[0], goAddressStrkeyPackage, "VersionByteMuxedContract") && goAddressIdent(call.Args[1], parameter) {
		return "strkey.Decode(VersionByteMuxedContract)", false
	}
	return "", false
}

func (c goAddressContext) rejects(statement ast.Stmt, boolean bool) bool {
	if expression, ok := statement.(*ast.ExprStmt); ok && !c.blocked["panic"] {
		call, ok := goAddressUnparen(expression.X).(*ast.CallExpr)
		if ok && goAddressIdent(call.Fun, "panic") && len(call.Args) == 1 {
			_, literal := goAddressString(call.Args[0])
			return literal
		}
	}
	result, ok := statement.(*ast.ReturnStmt)
	if !ok {
		return false
	}
	if boolean {
		return len(result.Results) == 1 && goAddressIdent(result.Results[0], "false") && !c.blocked["false"]
	}
	if len(result.Results) != 2 || !goAddressIdent(result.Results[0], "nil") || c.blocked["nil"] {
		return false
	}
	call, ok := goAddressUnparen(result.Results[1]).(*ast.CallExpr)
	if !ok || call.Ellipsis.IsValid() || len(call.Args) == 0 {
		return false
	}
	return (len(call.Args) == 1 && c.packageSelector(call.Fun, "errors", "New")) || c.packageSelector(call.Fun, "fmt", "Errorf")
}

func (c goAddressContext) truth(expression ast.Expr, parameter string) wTruth {
	expression = goAddressUnparen(expression)
	switch expression := expression.(type) {
	case *ast.UnaryExpr:
		if expression.Op == token.NOT {
			return wNot(c.truth(expression.X, parameter))
		}
	case *ast.BinaryExpr:
		switch expression.Op {
		case token.LAND:
			return wAnd(c.truth(expression.X, parameter), c.truth(expression.Y, parameter))
		case token.LOR:
			return wOr(c.truth(expression.X, parameter), c.truth(expression.Y, parameter))
		case token.EQL, token.NEQ:
			value, known := goAddressPrefixComparison(expression.X, expression.Y, parameter)
			if !known {
				value, known = goAddressPrefixComparison(expression.Y, expression.X, parameter)
			}
			if known {
				if expression.Op == token.NEQ {
					return wNot(value)
				}
				return value
			}
		}
	case *ast.CallExpr:
		if expression.Ellipsis.IsValid() {
			return wUnknown
		}
		if len(expression.Args) == 2 && c.packageSelector(expression.Fun, "strings", "HasPrefix") && goAddressIdent(expression.Args[0], parameter) {
			if prefix, ok := goAddressString(expression.Args[1]); ok {
				return prefixWTruth(prefix)
			}
		}
		selector, ok := goAddressUnparen(expression.Fun).(*ast.SelectorExpr)
		if ok && selector.Sel.Name == "MatchString" && len(expression.Args) == 1 && goAddressIdent(expression.Args[0], parameter) {
			compile, ok := goAddressUnparen(selector.X).(*ast.CallExpr)
			if ok && !compile.Ellipsis.IsValid() && len(compile.Args) == 1 && c.packageSelector(compile.Fun, "regexp", "MustCompile") {
				if pattern, ok := goAddressString(compile.Args[0]); ok {
					if rejects, supported := regexRejectsW(pattern, ""); supported && rejects {
						return wFalse
					}
				}
			}
		}
	}
	return wUnknown
}

// An unknown boolean operand is not necessarily pure: an unsupported call can
// mutate a parameter through a pointer before a later prefix test. Require the
// whole expression to belong to the supported side-effect-free grammar before
// combining abstract truth values with &&/||.
func (c goAddressContext) purePredicate(expression ast.Expr, parameter string) bool {
	switch expression := goAddressUnparen(expression).(type) {
	case *ast.UnaryExpr:
		return expression.Op == token.NOT && c.purePredicate(expression.X, parameter)
	case *ast.BinaryExpr:
		switch expression.Op {
		case token.LAND, token.LOR:
			return c.purePredicate(expression.X, parameter) && c.purePredicate(expression.Y, parameter)
		case token.EQL, token.NEQ:
			if _, known := goAddressPrefixComparison(expression.X, expression.Y, parameter); known {
				return true
			}
			_, known := goAddressPrefixComparison(expression.Y, expression.X, parameter)
			return known
		}
	case *ast.CallExpr:
		if expression.Ellipsis.IsValid() {
			return false
		}
		if len(expression.Args) == 2 && c.packageSelector(expression.Fun, "strings", "HasPrefix") && goAddressIdent(expression.Args[0], parameter) {
			_, literal := goAddressString(expression.Args[1])
			return literal
		}
		selector, ok := goAddressUnparen(expression.Fun).(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "MatchString" || len(expression.Args) != 1 || !goAddressIdent(expression.Args[0], parameter) {
			return false
		}
		compile, ok := goAddressUnparen(selector.X).(*ast.CallExpr)
		if ok && !compile.Ellipsis.IsValid() && len(compile.Args) == 1 && c.packageSelector(compile.Fun, "regexp", "MustCompile") {
			if pattern, literal := goAddressString(compile.Args[0]); literal {
				_, supported := regexRejectsW(pattern, "")
				return supported
			}
		}
	}
	return false
}

func goAddressPrefixComparison(left, right ast.Expr, parameter string) (wTruth, bool) {
	switch left := goAddressUnparen(left).(type) {
	case *ast.IndexExpr:
		if !goAddressIdent(left.X, parameter) || !goAddressInteger(left.Index, "0") {
			break
		}
		literal, ok := goAddressUnparen(right).(*ast.BasicLit)
		if !ok || literal.Kind != token.CHAR {
			break
		}
		value, err := strconv.Unquote(literal.Value)
		if err != nil || len(value) != 1 {
			break
		}
		if value[0] == 'W' {
			return wTrue, true
		}
		return wFalse, true
	case *ast.SliceExpr:
		if !goAddressIdent(left.X, parameter) || left.Slice3 || (left.Low != nil && !goAddressInteger(left.Low, "0")) || !goAddressInteger(left.High, "1") {
			break
		}
		value, ok := goAddressString(right)
		if ok && len(value) == 1 {
			if value == "W" {
				return wTrue, true
			}
			return wFalse, true
		}
	}
	return wUnknown, false
}

func goAddressUnparen(expression ast.Expr) ast.Expr {
	for {
		parenthesized, ok := expression.(*ast.ParenExpr)
		if !ok {
			return expression
		}
		expression = parenthesized.X
	}
}

func goAddressIdent(expression ast.Expr, name string) bool {
	identifier, ok := goAddressUnparen(expression).(*ast.Ident)
	return ok && identifier.Name == name
}

func goAddressString(expression ast.Expr) (string, bool) {
	literal, ok := goAddressUnparen(expression).(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(literal.Value)
	return value, err == nil
}

func goAddressInteger(expression ast.Expr, value string) bool {
	literal, ok := goAddressUnparen(expression).(*ast.BasicLit)
	return ok && literal.Kind == token.INT && literal.Value == value
}
