package rules

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

// Go SDK names that identify address kinds (go-stellar-sdk v0.8.0).
var (
	goStrkeyPredicates = map[string]byte{
		"IsValidEd25519PublicKey":             'G',
		"IsValidMuxedAccountEd25519PublicKey": 'M',
		"IsValidContractAddress":              'C',
		"IsValidMuxedContractAddress":         'W',
	}
	goContractCases      = map[string]bool{"ScAddressTypeScAddressTypeContract": true, "VersionByteContract": true}
	goMuxedContractCases = map[string]bool{"ScAddressTypeScAddressTypeMuxedContract": true, "VersionByteMuxedContract": true}
	goStrkeyPaths        = []string{"github.com/stellar/go-stellar-sdk/strkey", "github.com/stellar/go/strkey"}
	goRegexpFuncs        = map[string]bool{"MustCompile": true, "Compile": true, "MatchString": true, "MustCompilePOSIX": true, "CompilePOSIX": true}
	goLetterSetFuncs     = map[string]bool{"ContainsRune": true, "ContainsAny": true, "IndexByte": true, "IndexRune": true, "Contains": true, "IndexAny": true}
)

// goAddressObservations parses one Go file and returns the address shapes
// it contains. Package qualifiers are matched by import path, so renamed
// imports work and look-alike packages do not.
func goAddressObservations(filename string, src []byte) ([]observation, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	imports := map[string]string{} // local name -> import path
	for _, imp := range file.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		name := path0(p)
		if imp.Name != nil {
			name = imp.Name.Name
		}
		imports[name] = p
	}
	pkgIs := func(e ast.Expr, paths ...string) bool {
		id, ok := e.(*ast.Ident)
		if !ok {
			return false
		}
		for _, p := range paths {
			if imports[id.Name] == p {
				return true
			}
		}
		return false
	}
	line := func(n ast.Node) int { return fset.Position(n.Pos()).Line }

	var obs []observation
	nested := map[*ast.BinaryExpr]bool{} // inner && / || already grouped with their root
	ast.Inspect(file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.BinaryExpr:
			if (n.Op != token.LOR && n.Op != token.LAND) || nested[n] {
				return true
			}
			obs = append(obs, goLogical(n, nested, pkgIs, line)...)
		case *ast.CallExpr:
			sel, ok := n.Fun.(*ast.SelectorExpr)
			if !ok || len(n.Args) == 0 {
				return true
			}
			lit, ok := stringLit(n.Args[0])
			if !ok {
				return true
			}
			switch {
			case goRegexpFuncs[sel.Sel.Name] && pkgIs(sel.X, "regexp"):
				if k, ok := regexAllowlist(lit); ok {
					obs = append(obs, observation{pos: int(n.Pos()), line: line(n), construct: cRegex, kinds: k})
				}
			case goLetterSetFuncs[sel.Sel.Name] && pkgIs(sel.X, "strings") && len(lit) >= 2:
				if k, ok := lettersOf(lit); ok {
					obs = append(obs, observation{pos: int(n.Pos()), line: line(n), construct: cLetterList, kinds: k})
				}
			}
		case *ast.CompositeLit:
			if k, ok := goLetterList(n); ok {
				obs = append(obs, observation{pos: int(n.Pos()), line: line(n), construct: cLetterList, kinds: k})
			}
		case *ast.SwitchStmt:
			o := observation{pos: int(n.Pos()), line: line(n), construct: cTypeSwitch}
			for _, stmt := range n.Body.List {
				cc := stmt.(*ast.CaseClause)
				if cc.List == nil {
					o.hasDefault = true
				}
				for _, e := range cc.List {
					name := selectorName(e)
					o.contractCase = o.contractCase || goContractCases[name]
					o.muxedContractCase = o.muxedContractCase || goMuxedContractCases[name]
				}
			}
			if o.contractCase {
				obs = append(obs, o)
			}
		}
		return true
	})
	goMarkWNearby(file, obs)
	return obs, nil
}

// goWMarkers are identifiers that show code handling W addresses.
var goWMarkers = map[string]bool{
	"IsValidMuxedContractAddress": true, "DecodeMuxedContract": true,
	"VersionByteMuxedContract": true, "ScAddressTypeScAddressTypeMuxedContract": true,
	"MuxedContract": true,
}

// goMarkWNearby sets wNearby on observations whose enclosing top-level
// function (or the file, outside functions) also handles W addresses.
func goMarkWNearby(file *ast.File, obs []observation) {
	handlesW := func(n ast.Node) bool {
		found := false
		ast.Inspect(n, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.Ident:
				found = found || goWMarkers[n.Name]
			case *ast.BasicLit:
				found = found || n.Value == `"W"` || n.Value == `'W'`
			}
			return !found
		})
		return found
	}
	var fileW *bool
	for i := range obs {
		if obs[i].construct == cTypeSwitch {
			continue
		}
		inFunc := false
		for _, d := range file.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok && int(fd.Pos()) <= obs[i].pos && obs[i].pos < int(fd.End()) {
				obs[i].wNearby = handlesW(fd)
				inFunc = true
				break
			}
		}
		if !inFunc {
			if fileW == nil {
				w := handlesW(file)
				fileW = &w
			}
			obs[i].wNearby = *fileW
		}
	}
}

// goLogical collects prefix tests and strkey predicates from one && / ||
// expression, grouped by the value they test.
func goLogical(root *ast.BinaryExpr, nested map[*ast.BinaryExpr]bool, pkgIs func(ast.Expr, ...string) bool, line func(ast.Node) int) []observation {
	prefixes := map[string]kindSet{}
	var order []string
	var preds kindSet
	predCount := 0
	var walk func(e ast.Expr)
	walk = func(e ast.Expr) {
		switch e := e.(type) {
		case *ast.ParenExpr:
			walk(e.X)
		case *ast.UnaryExpr:
			walk(e.X)
		case *ast.BinaryExpr:
			switch e.Op {
			case token.LOR, token.LAND:
				nested[e] = true
				walk(e.X)
				walk(e.Y)
			case token.EQL, token.NEQ:
				if recv, letter, ok := goFirstCharCompare(e); ok {
					if _, seen := prefixes[recv]; !seen {
						order = append(order, recv)
					}
					k := prefixes[recv]
					if k.add(letter) {
						prefixes[recv] = k
					}
				}
			}
		case *ast.CallExpr:
			sel, ok := e.Fun.(*ast.SelectorExpr)
			if !ok {
				return
			}
			if sel.Sel.Name == "HasPrefix" && pkgIs(sel.X, "strings") && len(e.Args) == 2 {
				if lit, ok := stringLit(e.Args[1]); ok && len(lit) == 1 {
					recv := exprString(e.Args[0])
					if _, seen := prefixes[recv]; !seen {
						order = append(order, recv)
					}
					k := prefixes[recv]
					if k.add(lit[0]) {
						prefixes[recv] = k
					}
				}
				return
			}
			if letter, ok := goStrkeyPredicates[sel.Sel.Name]; ok && pkgIs(sel.X, goStrkeyPaths...) {
				preds.add(letter)
				predCount++
			}
		}
	}
	walk(root)

	var obs []observation
	for _, recv := range order {
		if k := prefixes[recv]; k.size() >= 2 {
			obs = append(obs, observation{pos: int(root.Pos()), line: line(root), construct: cPrefixTest, kinds: k})
		}
	}
	if predCount >= 2 {
		obs = append(obs, observation{pos: int(root.Pos()), line: line(root), construct: cPredicates, kinds: preds})
	}
	return obs
}

// goFirstCharCompare matches s[0] == 'G', s[:1] == "G", string(s[0]) == "G"
// and their mirrored forms.
func goFirstCharCompare(e *ast.BinaryExpr) (recv string, letter byte, ok bool) {
	for _, pair := range [][2]ast.Expr{{e.X, e.Y}, {e.Y, e.X}} {
		lhs, rhs := pair[0], pair[1]
		var lit string
		if b, isLit := rhs.(*ast.BasicLit); isLit && b.Kind == token.CHAR {
			r, _, _, err := strconv.UnquoteChar(strings.Trim(b.Value, "'"), '\'')
			if err != nil || r > 127 {
				continue
			}
			lit = string(r)
		} else if s, isStr := stringLit(rhs); isStr {
			lit = s
		} else {
			continue
		}
		if len(lit) != 1 {
			continue
		}
		if call, isCall := lhs.(*ast.CallExpr); isCall && len(call.Args) == 1 {
			if id, isID := call.Fun.(*ast.Ident); isID && id.Name == "string" {
				lhs = call.Args[0]
			}
		}
		switch x := lhs.(type) {
		case *ast.IndexExpr:
			if isIntLit(x.Index, "0") {
				return exprString(x.X), lit[0], true
			}
		case *ast.SliceExpr:
			if (x.Low == nil || isIntLit(x.Low, "0")) && isIntLit(x.High, "1") && !x.Slice3 {
				return exprString(x.X), lit[0], true
			}
		}
	}
	return "", 0, false
}

// goLetterList matches []string{"G", "M", "C"}, []byte{'G', 'C'} and maps
// keyed by such letters.
func goLetterList(c *ast.CompositeLit) (kindSet, bool) {
	if len(c.Elts) < 2 {
		return 0, false
	}
	var k kindSet
	for _, el := range c.Elts {
		if kv, isKV := el.(*ast.KeyValueExpr); isKV {
			el = kv.Key
		}
		b, ok := el.(*ast.BasicLit)
		if !ok {
			return 0, false
		}
		var s string
		switch b.Kind {
		case token.STRING:
			s, _ = strconv.Unquote(b.Value)
		case token.CHAR:
			s = strings.Trim(b.Value, "'")
		default:
			return 0, false
		}
		if len(s) != 1 || !k.add(s[0]) {
			return 0, false
		}
	}
	return k, true
}

func stringLit(e ast.Expr) (string, bool) {
	b, ok := e.(*ast.BasicLit)
	if !ok || b.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(b.Value)
	return s, err == nil
}

func isIntLit(e ast.Expr, v string) bool {
	b, ok := e.(*ast.BasicLit)
	return ok && b.Kind == token.INT && b.Value == v
}

// selectorName returns the final name of x.Name or Name.
func selectorName(e ast.Expr) string {
	switch e := e.(type) {
	case *ast.SelectorExpr:
		return e.Sel.Name
	case *ast.Ident:
		return e.Name
	}
	return ""
}

// exprString renders simple expressions (identifiers, field selections,
// index expressions) as a grouping key.
func exprString(e ast.Expr) string {
	switch e := e.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return exprString(e.X) + "." + e.Sel.Name
	case *ast.ParenExpr:
		return exprString(e.X)
	case *ast.IndexExpr:
		return exprString(e.X) + "[]"
	case *ast.CallExpr:
		return exprString(e.Fun) + "()"
	}
	return "?"
}

// path0 is the default package name for an import path: its last element.
func path0(p string) string {
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return p[i+1:]
	}
	return p
}
