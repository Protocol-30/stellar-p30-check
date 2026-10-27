package rules

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"github.com/Protocol-30/stellar-p30-check/internal/walker"
)

// sourceFile is one file read and parsed once for every source rule.
type sourceFile struct {
	walker.File
	data    []byte
	lines   []string
	readErr error
	fset    *token.FileSet
	goAST   *ast.File
	goErr   error   // Go parse error
	js      *jsFile // JS/TS tokens and bracket index
}

func loadSource(in Input, f walker.File) *sourceFile {
	sf := &sourceFile{File: f}
	sf.data, sf.readErr = in.ReadFile(f.Path)
	if sf.readErr != nil {
		return sf
	}
	sf.lines = strings.Split(string(sf.data), "\n")
	switch {
	case f.Kind == "source-go":
		sf.fset = token.NewFileSet()
		sf.goAST, sf.goErr = parser.ParseFile(sf.fset, f.Path, sf.data, parser.SkipObjectResolution)
	case isCodeKind(f.Kind):
		sf.js = newJSFile(sf.data)
	}
	return sf
}

// codeProblem returns the warning for a code file that cannot be checked,
// or nil when it can.
func (sf *sourceFile) codeProblem() error {
	if sf.readErr != nil {
		return sf.readErr
	}
	if sf.Kind == "source-go" {
		return sf.goErr
	}
	return nil
}

// sourceRule is a rule that looks at source files one at a time. CheckAll
// runs all source rules in a single pass so each file is read and parsed
// once, and only one parsed file is held in memory at a time.
type sourceRule interface {
	Rule
	// prepare does the whole-input work (such as SDK version resolution)
	// and returns the per-file check.
	prepare(ctx context.Context, in Input) (func(*sourceFile) Output, error)
}

// isSourceInput reports whether a discovered file is read by source rules.
func isSourceInput(f walker.File) bool {
	return isCodeKind(f.Kind) || f.Kind == "xdr-definition"
}

// checkSource runs a single source rule on its own.
func checkSource(ctx context.Context, in Input, r sourceRule) (Output, error) {
	outs, err := CheckAll(ctx, in, []Rule{r})
	if err != nil {
		return Output{}, err
	}
	return outs[0], nil
}

// CheckAll runs rules over the input and returns one Output per rule, in
// order. Errors name the failing rule.
func CheckAll(ctx context.Context, in Input, rs []Rule) ([]Output, error) {
	outs := make([]Output, len(rs))
	type check struct {
		idx int
		fn  func(*sourceFile) Output
	}
	var checks []check
	for i, r := range rs {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("scan cancelled: %w", err)
		}
		if sr, ok := r.(sourceRule); ok {
			fn, err := sr.prepare(ctx, in)
			if err != nil {
				return nil, fmt.Errorf("rule %s: %w", r.ID(), err)
			}
			checks = append(checks, check{i, fn})
			continue
		}
		out, err := r.Check(ctx, in)
		if err != nil {
			return nil, fmt.Errorf("rule %s: %w", r.ID(), err)
		}
		outs[i] = out
	}
	if len(checks) == 0 {
		return outs, nil
	}
	for _, f := range in.Files {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("scan cancelled: %w", err)
		}
		if !isSourceInput(f) {
			continue
		}
		sf := loadSource(in, f)
		for _, c := range checks {
			o := c.fn(sf)
			outs[c.idx].Findings = append(outs[c.idx].Findings, o.Findings...)
			outs[c.idx].Warnings = append(outs[c.idx].Warnings, o.Warnings...)
		}
	}
	return outs, nil
}
