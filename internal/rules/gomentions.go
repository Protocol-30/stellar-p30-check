package rules

import (
	"go/ast"
	"go/token"
	"sort"
)

// goMentions answers "does the top-level function around this position (or
// the file, outside functions) contain a node matching pred?". Each scope is
// inspected at most once, so asking for every finding in a file stays
// linear in the file size.
type goMentions struct {
	file *ast.File
	pred func(ast.Node) bool
	memo map[ast.Node]bool
}

func newGoMentions(file *ast.File, pred func(ast.Node) bool) *goMentions {
	return &goMentions{file: file, pred: pred, memo: map[ast.Node]bool{}}
}

// identIn returns a predicate matching identifiers in names.
func identIn(names map[string]bool) func(ast.Node) bool {
	return func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		return ok && names[id.Name]
	}
}

func (m *goMentions) at(pos token.Pos) bool {
	var scope ast.Node = m.file
	decls := m.file.Decls // in source order
	if i := sort.Search(len(decls), func(i int) bool { return decls[i].End() > pos }); i < len(decls) {
		if fd, ok := decls[i].(*ast.FuncDecl); ok && fd.Pos() <= pos {
			scope = fd
		}
	}
	if v, ok := m.memo[scope]; ok {
		return v
	}
	found := false
	ast.Inspect(scope, func(n ast.Node) bool {
		if n != nil && m.pred(n) {
			found = true
		}
		return !found
	})
	m.memo[scope] = found
	return found
}
