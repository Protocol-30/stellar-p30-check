package rules

// jsFile is a tokenized JS/TS file with bracket structure computed once, so
// matchers can look up matching brackets, enclosing groups and token counts
// in constant time. Scanned files are untrusted: every matcher must stay
// linear in the file size, whatever the nesting.
type jsFile struct {
	toks       []jsToken
	match      []int // index of the matching bracket for ( ) [ ] { }, -1 if none
	encl       []int // index of the innermost open bracket around each token, -1 at top level
	firstMinus []int // for "(": the first "-" directly inside the group, -1 if none
	prefix     map[string][]int
}

func isOpen(t jsToken) bool {
	return t.Kind == jsPunct && (t.Text == "(" || t.Text == "[" || t.Text == "{")
}

func isClose(t jsToken) bool {
	return t.Kind == jsPunct && (t.Text == ")" || t.Text == "]" || t.Text == "}")
}

func newJSFile(src []byte) *jsFile {
	toks := jsTokenize(src)
	n := len(toks)
	f := &jsFile{
		toks: toks, match: make([]int, n), encl: make([]int, n),
		firstMinus: make([]int, n), prefix: map[string][]int{},
	}
	var stack []int
	for i, t := range toks {
		f.match[i], f.firstMinus[i], f.encl[i] = -1, -1, -1
		if len(stack) > 0 {
			f.encl[i] = stack[len(stack)-1]
		}
		switch {
		case isOpen(t):
			stack = append(stack, i)
		case isClose(t) && len(stack) > 0:
			o := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			f.match[o], f.match[i] = i, o
		case t.Kind == jsPunct && t.Text == "-" && len(stack) > 0:
			if o := stack[len(stack)-1]; toks[o].Text == "(" && f.firstMinus[o] < 0 {
				f.firstMinus[o] = i
			}
		}
	}
	return f
}

func (f *jsFile) at(i int, kind jsKind, text string) bool {
	return i >= 0 && i < len(f.toks) && f.toks[i].Kind == kind && (text == "" || f.toks[i].Text == text)
}

// count returns how many tokens in [lo, hi) satisfy pred. The prefix sums
// for each named predicate are built once per file.
func (f *jsFile) count(name string, pred func(i int) bool, lo, hi int) int {
	p, ok := f.prefix[name]
	if !ok {
		p = make([]int, len(f.toks)+1)
		for i := range f.toks {
			p[i+1] = p[i]
			if pred(i) {
				p[i+1]++
			}
		}
		f.prefix[name] = p
	}
	if lo < 0 {
		lo = 0
	}
	if hi > len(f.toks) {
		hi = len(f.toks)
	}
	if lo >= hi {
		return 0
	}
	return p[hi] - p[lo]
}

// jsSwitch describes "switch (...) { ... }" starting at token i.
type jsSwitch struct {
	bodyOpen, bodyClose int
	hasDefault          bool
}

// switchAt returns the switch statement starting at token i, if any.
func (f *jsFile) switchAt(i int) (jsSwitch, bool) {
	if !f.at(i, jsIdent, "switch") || !f.at(i+1, jsPunct, "(") {
		return jsSwitch{}, false
	}
	closeParen := f.match[i+1]
	if closeParen < 0 || !f.at(closeParen+1, jsPunct, "{") {
		return jsSwitch{}, false
	}
	open := closeParen + 1
	if f.match[open] < 0 {
		return jsSwitch{}, false
	}
	return jsSwitch{bodyOpen: open, bodyClose: f.match[open], hasDefault: f.hasDefault(open)}, true
}

// hasDefault reports whether "default:" appears directly inside the brace
// group opened at open (not inside a nested block).
func (f *jsFile) hasDefault(open int) bool {
	if f.prefix["default-in"] == nil {
		// Mark each brace group that directly contains "default:".
		marks := make([]int, len(f.toks)+1)
		for j := range f.toks {
			if f.at(j, jsIdent, "default") && f.at(j+1, jsPunct, ":") && f.encl[j] >= 0 {
				marks[f.encl[j]] = 1
			}
		}
		f.prefix["default-in"] = marks
	}
	return f.prefix["default-in"][open] == 1
}
