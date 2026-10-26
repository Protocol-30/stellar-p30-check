package rules

import (
	"fmt"
	"path"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Protocol-30/stellar-p30-check/internal/walker"
	treesitter "github.com/tree-sitter/go-tree-sitter"
	javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

const jsAddressNodeLimit = 200000

// This is a deliberately small, syntax-based flow check. Only top-level simple
// functions are considered; it is not an interprocedural JavaScript analyzer.
func analyzeJSAddress(file walker.File) ([]addressCandidate, []string, error) {
	result := []addressCandidate{}
	notes := []string{}
	parser := treesitter.NewParser()
	defer parser.Close()
	language := treesitter.NewLanguage(javascript.Language())
	switch path.Ext(file.Path) {
	case ".ts":
		language = treesitter.NewLanguage(typescript.LanguageTypescript())
	case ".tsx":
		language = treesitter.NewLanguage(typescript.LanguageTSX())
	}
	if err := parser.SetLanguage(language); err != nil {
		return result, notes, fmt.Errorf("load source grammar: %w", err)
	}
	parser.SetTimeoutMicros(2_000_000)
	tree := parser.Parse(file.Data, nil)
	if tree == nil {
		return result, notes, fmt.Errorf("source parsing exhausted its two-second budget")
	}
	defer tree.Close()
	root := tree.RootNode()
	// Count all nodes, not just named nodes. The iterative traversal also avoids
	// consuming a Go call stack proportional to target-controlled nesting.
	all := []*treesitter.Node{}
	stack := []*treesitter.Node{root}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		all = append(all, n)
		if len(all) > jsAddressNodeLimit {
			return result, notes, fmt.Errorf("source AST exceeds %d nodes", jsAddressNodeLimit)
		}
		for i := n.ChildCount(); i > 0; i-- {
			stack = append(stack, n.Child(i-1))
		}
	}
	if root.HasError() {
		return result, []string{fmt.Sprintf("Address analysis skipped %q because its source syntax is unsupported or malformed.", file.Path)}, nil
	}
	a := jsAddressAnalyzer{data: file.Data, bindings: map[string]string{}}
	a.readImports(root)
	if len(a.bindings) == 0 {
		return result, notes, nil
	}
	a.rejectUnsafeBindings(root, all)
	for _, statement := range jsNamed(root) {
		if statement.Kind() == "export_statement" {
			if child := statement.ChildByFieldName("declaration"); child != nil {
				statement = child
			} else {
				continue
			}
		}
		var functions []*treesitter.Node
		switch statement.Kind() {
		case "function_declaration":
			functions = append(functions, statement)
		case "lexical_declaration", "variable_declaration":
			for _, declaration := range jsNamed(statement) {
				if declaration.Kind() != "variable_declarator" || !jsIsIdentifier(declaration.ChildByFieldName("name")) {
					continue
				}
				value := declaration.ChildByFieldName("value")
				if value != nil && (value.Kind() == "arrow_function" || value.Kind() == "function_expression") {
					functions = append(functions, value)
				}
			}
		}
		for _, function := range functions {
			if candidate, ok := a.function(file.Path, function); ok {
				result = append(result, candidate)
			}
		}
	}
	return result, notes, nil
}

type jsAddressAnalyzer struct {
	data              []byte
	bindings          map[string]string // local name -> StrKey, TransactionBuilder, namespace
	undefinedShadowed bool
}

func (a *jsAddressAnalyzer) text(n *treesitter.Node) string {
	if n == nil {
		return ""
	}
	return string(a.data[n.StartByte():n.EndByte()])
}

func jsNamed(n *treesitter.Node) []*treesitter.Node {
	if n == nil {
		return nil
	}
	children := []*treesitter.Node{}
	for i := uint(0); i < n.NamedChildCount(); i++ {
		child := n.NamedChild(i)
		if child.Kind() != "comment" {
			children = append(children, child)
		}
	}
	return children
}

func jsIsIdentifier(n *treesitter.Node) bool { return n != nil && n.Kind() == "identifier" }

func jsUnwrap(n *treesitter.Node) *treesitter.Node {
	for n != nil && n.Kind() == "parenthesized_expression" {
		children := jsNamed(n)
		if len(children) != 1 {
			return nil
		}
		n = children[0]
	}
	return n
}

func jsHasToken(n *treesitter.Node, kind string) bool {
	for i := uint(0); i < n.ChildCount(); i++ {
		if n.Child(i).Kind() == kind {
			return true
		}
	}
	return false
}

func (a *jsAddressAnalyzer) readImports(root *treesitter.Node) {
	counts := map[string]int{}
	for _, statement := range jsNamed(root) {
		if statement.Kind() != "import_statement" || jsHasToken(statement, "type") {
			continue
		}
		source, valid := a.stringLiteral(statement.ChildByFieldName("source"))
		isSDK := valid && (source == "@stellar/stellar-sdk" || source == "@stellar/stellar-sdk/base")
		for _, clause := range jsNamed(statement) {
			if clause.Kind() != "import_clause" {
				continue
			}
			for _, imported := range jsNamed(clause) {
				switch imported.Kind() {
				case "identifier":
					counts[a.text(imported)]++ // default imports are not supported SDK bindings
				case "namespace_import":
					children := jsNamed(imported)
					if len(children) == 1 && jsIsIdentifier(children[0]) {
						local := a.text(children[0])
						counts[local]++
						if isSDK {
							a.bindings[local] = "namespace"
						}
					}
				case "named_imports":
					for _, specifier := range jsNamed(imported) {
						if specifier.Kind() != "import_specifier" || jsHasToken(specifier, "type") {
							continue
						}
						name := specifier.ChildByFieldName("name")
						alias := specifier.ChildByFieldName("alias")
						if alias == nil {
							alias = name
						}
						if !jsIsIdentifier(alias) {
							continue
						}
						local, canonical := a.text(alias), a.text(name)
						counts[local]++
						if isSDK && (canonical == "StrKey" || canonical == "TransactionBuilder") {
							a.bindings[local] = canonical
						}
					}
				}
			}
		}
	}
	for local := range a.bindings {
		if counts[local] != 1 {
			delete(a.bindings, local)
		}
	}
	a.undefinedShadowed = counts["undefined"] > 0
}

func (a *jsAddressAnalyzer) expressionRoot(n *treesitter.Node) string {
	n = jsUnwrap(n)
	for n != nil {
		switch n.Kind() {
		case "identifier":
			return a.text(n)
		case "member_expression", "subscript_expression":
			n = jsUnwrap(n.ChildByFieldName("object"))
		case "as_expression", "satisfies_expression", "non_null_expression":
			children := jsNamed(n)
			if len(children) == 0 {
				return ""
			}
			n = jsUnwrap(children[0])
		default:
			return ""
		}
	}
	return ""
}

// Reject obvious module-level rebinding, member writes, escapes to unknown
// callbacks, and built-in prototype modification. This is conservative: an
// unrelated local write may suppress a match rather than inventing certainty.
func (a *jsAddressAnalyzer) rejectUnsafeBindings(root *treesitter.Node, nodes []*treesitter.Node) {
	unsafe := map[string]bool{}
	for _, statement := range jsNamed(root) {
		if statement.Kind() == "export_statement" {
			statement = statement.ChildByFieldName("declaration")
			if statement == nil {
				continue
			}
		}
		switch statement.Kind() {
		case "function_declaration", "class_declaration":
			unsafe[a.text(statement.ChildByFieldName("name"))] = true
		case "lexical_declaration", "variable_declaration":
			for _, declaration := range jsNamed(statement) {
				if declaration.Kind() == "variable_declarator" {
					unsafe[a.text(declaration.ChildByFieldName("name"))] = true
				}
			}
		}
	}
	builtinsChanged := false
	for _, node := range nodes {
		switch node.Kind() {
		case "identifier", "shorthand_property_identifier":
			name := a.text(node)
			if a.bindings[name] != "" && !a.safeSDKUse(node) {
				unsafe[name] = true
			}
			if jsSensitiveGlobal(name) && !a.safeSDKUse(node) {
				builtinsChanged = true
			}
		case "assignment_expression", "augmented_assignment_expression":
			name := a.expressionRoot(node.ChildByFieldName("left"))
			unsafe[name] = true
			if jsSensitiveGlobal(name) {
				builtinsChanged = true
			}
			if name = a.expressionRoot(node.ChildByFieldName("right")); a.bindings[name] != "" {
				unsafe[name] = true
			}
		case "update_expression":
			name := a.expressionRoot(node.ChildByFieldName("argument"))
			unsafe[name] = true
			if jsSensitiveGlobal(name) {
				builtinsChanged = true
			}
		case "variable_declarator":
			name := a.expressionRoot(node.ChildByFieldName("value"))
			if a.bindings[name] != "" {
				unsafe[name] = true
			}
			if jsSensitiveGlobal(name) {
				builtinsChanged = true
			}
		case "unary_expression":
			if a.text(node.ChildByFieldName("operator")) == "delete" {
				name := a.expressionRoot(node.ChildByFieldName("argument"))
				unsafe[name] = true
				if jsSensitiveGlobal(name) {
					builtinsChanged = true
				}
			}
		case "call_expression":
			callee := node.ChildByFieldName("function")
			if jsIsIdentifier(callee) && (a.text(callee) == "eval" || a.text(callee) == "Function") {
				builtinsChanged = true // opaque dynamic code can alter SDK bindings
			}
			if callee != nil && callee.Kind() == "member_expression" {
				property := a.text(callee.ChildByFieldName("property"))
				if property == "eval" || property == "Function" {
					builtinsChanged = true
				}
			}
			for _, argument := range jsNamed(node.ChildByFieldName("arguments")) {
				name := a.expressionRoot(argument)
				if a.bindings[name] != "" {
					unsafe[name] = true
				}
				if jsSensitiveGlobal(name) {
					builtinsChanged = true
				}
			}
		case "new_expression":
			if constructor := node.ChildByFieldName("constructor"); jsIsIdentifier(constructor) && a.text(constructor) == "Function" {
				builtinsChanged = true
			} else if constructor != nil && constructor.Kind() == "member_expression" && a.text(constructor.ChildByFieldName("property")) == "Function" {
				builtinsChanged = true
			}
			for _, argument := range jsNamed(node.ChildByFieldName("arguments")) {
				if jsSensitiveGlobal(a.expressionRoot(argument)) {
					builtinsChanged = true
				}
			}
		}
	}
	for name := range a.bindings {
		if unsafe[name] || builtinsChanged {
			delete(a.bindings, name)
		}
	}
	a.undefinedShadowed = a.undefinedShadowed || unsafe["undefined"]
}

func jsSensitiveGlobal(name string) bool {
	switch name {
	case "RegExp", "String", "Array", "Function", "eval", "globalThis", "window", "self", "global":
		return true
	}
	return false
}

// SDK objects must not escape in a container, cast, callback argument, or
// extracted alias. Walk only their immediate member chain, so inspection stays
// linear in the source tree rather than recursively rescanning call arguments.
func (a *jsAddressAnalyzer) safeSDKUse(node *treesitter.Node) bool {
	current := node
	for depth := 0; depth < 64; depth++ {
		parent := current.Parent()
		if parent == nil {
			return false
		}
		switch parent.Kind() {
		case "import_specifier", "namespace_import", "import_clause", "type_query", "typeof_type":
			return true
		case "member_expression", "subscript_expression":
			object := parent.ChildByFieldName("object")
			if object == nil || object.Id() != current.Id() {
				return false
			}
			current = parent
		case "parenthesized_expression", "as_expression", "satisfies_expression", "non_null_expression":
			current = parent
		case "call_expression":
			callee := parent.ChildByFieldName("function")
			return callee != nil && callee.Id() == current.Id() && current.Kind() == "member_expression"
		case "new_expression":
			constructor := parent.ChildByFieldName("constructor")
			return constructor != nil && constructor.Id() == current.Id() && a.sdkSymbol(current, "TransactionBuilder")
		default:
			return false
		}
	}
	return false
}

func (a *jsAddressAnalyzer) function(file string, function *treesitter.Node) (addressCandidate, bool) {
	body := function.ChildByFieldName("body")
	if body == nil || body.Kind() != "statement_block" {
		return addressCandidate{}, false
	}
	parameters, valid := a.parameters(function)
	if !valid {
		return addressCandidate{}, false
	}
	for _, name := range parameters {
		if a.bindings[name] != "" || name == "undefined" {
			return addressCandidate{}, false
		}
	}
	if name := a.text(function.ChildByFieldName("name")); a.bindings[name] != "" || name == "undefined" {
		return addressCandidate{}, false
	}
	statements := jsNamed(body)
	if len(statements) != 2 && len(statements) != 3 {
		return addressCandidate{}, false
	}
	var builder string
	if len(statements) == 3 {
		builder = a.builderDeclaration(statements[0])
		if builder == "" || a.bindings[builder] != "" || jsStringContains(parameters, builder) {
			return addressCandidate{}, false
		}
		statements = statements[1:]
	}
	guard, last := statements[0], statements[1]
	if guard.Kind() != "if_statement" || guard.ChildByFieldName("alternative") != nil || !a.rejects(guard.ChildByFieldName("consequence")) || last.Kind() != "return_statement" {
		return addressCandidate{}, false
	}
	returned := jsNamed(last)
	if len(returned) != 1 {
		return addressCandidate{}, false
	}
	call := jsUnwrap(returned[0])
	if call == nil || call.Kind() != "call_expression" || call.ChildByFieldName("optional_chain") != nil {
		return addressCandidate{}, false
	}
	callee := call.ChildByFieldName("function")
	arguments := jsNamed(call.ChildByFieldName("arguments"))
	if len(arguments) == 0 || !jsIsIdentifier(arguments[0]) {
		return addressCandidate{}, false
	}
	parameter := a.text(arguments[0])
	if !jsStringContains(parameters, parameter) {
		return addressCandidate{}, false
	}
	kind, api := "", ""
	if callee != nil && callee.Kind() == "member_expression" && callee.ChildByFieldName("optional_chain") == nil {
		method := a.text(callee.ChildByFieldName("property"))
		object := jsUnwrap(callee.ChildByFieldName("object"))
		if a.sdkSymbol(object, "StrKey") && len(arguments) == 1 && (method == "decodeMuxedContract" || method == "isValidMuxedContract") && builder == "" {
			kind, api = addressWOnly, "StrKey."+method
		} else if method == "addSacTransferOperation" && len(arguments) >= 3 && len(arguments) <= 4 && ((builder != "" && jsIsIdentifier(object) && a.text(object) == builder) || a.newBuilder(object)) {
			// Arguments other than destination must not contain assignments or
			// callbacks that could overwrite the selected input before the call.
			if !a.simpleSACArguments(arguments[1:]) {
				return addressCandidate{}, false
			}
			kind, api = addressSACDestination, "TransactionBuilder.addSacTransferOperation(destination, asset, amount)"
		}
	}
	if kind == "" {
		return addressCandidate{}, false
	}
	truth, safe := a.guardTruth(guard.ChildByFieldName("condition"), parameter, 0)
	if !safe || truth != wTrue {
		return addressCandidate{}, false
	}
	return addressCandidate{
		path: file, line: int(guard.StartPosition().Row) + 1,
		evidence: a.text(guard), parameter: parameter, api: api, ecosystem: "npm", kind: kind,
	}, true
}

func (a *jsAddressAnalyzer) parameters(function *treesitter.Node) ([]string, bool) {
	parameters := function.ChildByFieldName("parameters")
	if parameters == nil {
		parameter := function.ChildByFieldName("parameter")
		if jsIsIdentifier(parameter) {
			return []string{a.text(parameter)}, true
		}
		return nil, false
	}
	result := []string{}
	seen := map[string]bool{}
	for _, parameter := range jsNamed(parameters) {
		if parameter.Kind() == "required_parameter" {
			if parameter.ChildByFieldName("value") != nil || jsHasToken(parameter, "=") {
				return nil, false
			}
			parameter = parameter.ChildByFieldName("pattern")
		}
		if !jsIsIdentifier(parameter) || seen[a.text(parameter)] {
			return nil, false
		}
		seen[a.text(parameter)] = true
		result = append(result, a.text(parameter))
	}
	return result, len(result) > 0
}

func (a *jsAddressAnalyzer) sdkSymbol(n *treesitter.Node, symbol string) bool {
	n = jsUnwrap(n)
	if jsIsIdentifier(n) {
		return a.bindings[a.text(n)] == symbol
	}
	return n != nil && n.Kind() == "member_expression" && n.ChildByFieldName("optional_chain") == nil && jsIsIdentifier(n.ChildByFieldName("object")) && a.bindings[a.text(n.ChildByFieldName("object"))] == "namespace" && a.text(n.ChildByFieldName("property")) == symbol
}

func (a *jsAddressAnalyzer) newBuilder(n *treesitter.Node) bool {
	n = jsUnwrap(n)
	if n == nil || n.Kind() != "new_expression" || !a.sdkSymbol(n.ChildByFieldName("constructor"), "TransactionBuilder") {
		return false
	}
	return a.simpleSACArguments(jsNamed(n.ChildByFieldName("arguments")))
}

func (a *jsAddressAnalyzer) builderDeclaration(n *treesitter.Node) string {
	if n.Kind() != "lexical_declaration" || !jsHasToken(n, "const") {
		return ""
	}
	children := jsNamed(n)
	if len(children) != 1 || children[0].Kind() != "variable_declarator" || !jsIsIdentifier(children[0].ChildByFieldName("name")) || !a.newBuilder(children[0].ChildByFieldName("value")) {
		return ""
	}
	return a.text(children[0].ChildByFieldName("name"))
}

func (a *jsAddressAnalyzer) simpleSACArguments(arguments []*treesitter.Node) bool {
	stack := append([]*treesitter.Node(nil), arguments...)
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		switch n.Kind() {
		case "identifier", "property_identifier", "shorthand_property_identifier", "string", "string_fragment", "escape_sequence", "number", "true", "false", "null", "undefined", "object", "pair", "array":
		default:
			return false
		}
		stack = append(stack, jsNamed(n)...)
	}
	return true
}

func (a *jsAddressAnalyzer) rejects(n *treesitter.Node) bool {
	if n == nil {
		return false
	}
	if n.Kind() == "statement_block" {
		children := jsNamed(n)
		if len(children) != 1 {
			return false
		}
		n = children[0]
	}
	if n.Kind() == "throw_statement" {
		return true
	}
	if n.Kind() == "return_statement" {
		children := jsNamed(n)
		if len(children) != 1 {
			return false
		}
		value := jsUnwrap(children[0])
		return value != nil && (value.Kind() == "false" || value.Kind() == "null" || (!a.undefinedShadowed && (value.Kind() == "undefined" || (jsIsIdentifier(value) && a.text(value) == "undefined"))))
	}
	return false
}

func (a *jsAddressAnalyzer) guardTruth(n *treesitter.Node, parameter string, depth int) (wTruth, bool) {
	if depth > 64 {
		return wUnknown, false
	}
	n = jsUnwrap(n)
	if n == nil {
		return wUnknown, false
	}
	switch n.Kind() {
	case "unary_expression":
		if a.text(n.ChildByFieldName("operator")) == "!" {
			value, safe := a.guardTruth(n.ChildByFieldName("argument"), parameter, depth+1)
			return wNot(value), safe
		}
	case "binary_expression":
		op := a.text(n.ChildByFieldName("operator"))
		if op == "&&" || op == "||" {
			left, leftSafe := a.guardTruth(n.ChildByFieldName("left"), parameter, depth+1)
			right, rightSafe := a.guardTruth(n.ChildByFieldName("right"), parameter, depth+1)
			if op == "&&" {
				return wAnd(left, right), leftSafe && rightSafe
			}
			return wOr(left, right), leftSafe && rightSafe
		}
		if op == "===" || op == "!==" {
			left, right := n.ChildByFieldName("left"), n.ChildByFieldName("right")
			literal, valid := a.stringLiteral(right)
			if !a.firstCharacter(left, parameter) || !valid {
				literal, valid = a.stringLiteral(left)
				if !a.firstCharacter(right, parameter) || !valid {
					return wUnknown, false
				}
			}
			value := wFalse
			if literal == "W" {
				value = wTrue
			}
			if op == "!==" {
				value = wNot(value)
			}
			return value, true
		}
	case "call_expression":
		if n.ChildByFieldName("optional_chain") != nil {
			return wUnknown, false
		}
		callee := n.ChildByFieldName("function")
		arguments := jsNamed(n.ChildByFieldName("arguments"))
		if callee == nil || callee.Kind() != "member_expression" || callee.ChildByFieldName("optional_chain") != nil || len(arguments) != 1 {
			return wUnknown, false
		}
		object, method := jsUnwrap(callee.ChildByFieldName("object")), a.text(callee.ChildByFieldName("property"))
		if method == "startsWith" && jsIsIdentifier(object) && a.text(object) == parameter {
			if prefix, valid := a.stringLiteral(arguments[0]); valid {
				return prefixWTruth(prefix), true
			}
		}
		if method == "test" && object != nil && object.Kind() == "regex" && jsIsIdentifier(arguments[0]) && a.text(arguments[0]) == parameter {
			pattern, flags := a.text(object.ChildByFieldName("pattern")), a.text(object.ChildByFieldName("flags"))
			// These Go regexp/syntax extensions are not equivalent JavaScript
			// legacy-regex constructs. In particular, a negated Go Unicode/POSIX
			// class can exclude W while the same JS source accepts it.
			if strings.Contains(pattern, "[:") || strings.Contains(pattern, `\p`) || strings.Contains(pattern, `\P`) || strings.Contains(pattern, `\x{`) {
				return wUnknown, false
			}
			reject, supported := regexRejectsW(pattern, flags)
			if reject {
				return wFalse, supported
			}
			return wUnknown, supported
		}
		if method == "includes" && object != nil && object.Kind() == "array" && a.firstCharacter(arguments[0], parameter) {
			value := wFalse
			for _, element := range jsNamed(object) {
				literal, valid := a.stringLiteral(element)
				if !valid {
					return wUnknown, false
				}
				if literal == "W" {
					value = wTrue
				}
			}
			return value, true
		}
	}
	return wUnknown, false
}

func (a *jsAddressAnalyzer) firstCharacter(n *treesitter.Node, parameter string) bool {
	n = jsUnwrap(n)
	if n == nil {
		return false
	}
	if n.Kind() == "subscript_expression" {
		return jsIsIdentifier(n.ChildByFieldName("object")) && a.text(n.ChildByFieldName("object")) == parameter && a.text(n.ChildByFieldName("index")) == "0"
	}
	if n.Kind() == "call_expression" {
		if n.ChildByFieldName("optional_chain") != nil {
			return false
		}
		function := n.ChildByFieldName("function")
		arguments := jsNamed(n.ChildByFieldName("arguments"))
		return function != nil && function.Kind() == "member_expression" && function.ChildByFieldName("optional_chain") == nil && jsIsIdentifier(function.ChildByFieldName("object")) && a.text(function.ChildByFieldName("object")) == parameter && a.text(function.ChildByFieldName("property")) == "charAt" && len(arguments) == 1 && a.text(arguments[0]) == "0"
	}
	return false
}

func jsStringContains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

// Decode only a documented, unambiguous subset of JS string escapes. In
// particular, escaped W must never be mistaken for a C/G/M-only prefix.
func (a *jsAddressAnalyzer) stringLiteral(n *treesitter.Node) (string, bool) {
	if n == nil || n.Kind() != "string" {
		return "", false
	}
	raw := a.text(n)
	if len(raw) < 2 || (raw[0] != '\'' && raw[0] != '"') || raw[len(raw)-1] != raw[0] {
		return "", false
	}
	var out strings.Builder
	for i := 1; i < len(raw)-1; i++ {
		if raw[i] != '\\' {
			out.WriteByte(raw[i])
			continue
		}
		i++
		if i >= len(raw)-1 {
			return "", false
		}
		switch raw[i] {
		case '\'', '"', '\\', '/':
			out.WriteByte(raw[i])
		case 'b':
			out.WriteByte('\b')
		case 'f':
			out.WriteByte('\f')
		case 'n':
			out.WriteByte('\n')
		case 'r':
			out.WriteByte('\r')
		case 't':
			out.WriteByte('\t')
		case 'v':
			out.WriteByte('\v')
		case 'x', 'u':
			width := 2
			if raw[i] == 'u' {
				width = 4
			}
			if i+width >= len(raw)-1 {
				return "", false
			}
			value, err := strconv.ParseUint(raw[i+1:i+width+1], 16, 32)
			if err != nil || !utf8.ValidRune(rune(value)) {
				return "", false
			}
			out.WriteRune(rune(value))
			i += width
		default:
			// Legacy octal, identity escapes, braced Unicode, and continuation
			// forms are intentionally outside this proof subset.
			return "", false
		}
	}
	return out.String(), true
}
