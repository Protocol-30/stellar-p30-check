package rules

import "strings"

// JavaScript SDK names that identify address kinds (@stellar/stellar-sdk 17.3.0).
var jsStrKeyPredicates = map[string]byte{
	"isValidEd25519PublicKey":  'G',
	"isValidMed25519PublicKey": 'M',
	"isValidContract":          'C',
	"isValidMuxedContract":     'W',
}

// jsAddressObservations finds address shapes in JS/TS/JSX/TSX source using
// tokens only. Expression grouping is approximate: tests on the same value
// within one statement (between ;, { and }) are treated as one check.
func jsAddressObservations(src []byte) []observation {
	return jsAddressObs(newJSFile(src))
}

func jsAddressObs(f *jsFile) []observation {
	toks := f.toks
	at := f.at
	var obs []observation

	// Regex literals and RegExp("...") constructions.
	for i, t := range toks {
		pattern := ""
		switch {
		case t.Kind == jsRegex:
			pattern = t.Text
		case at(i, jsIdent, "RegExp") && at(i+1, jsPunct, "(") && at(i+2, jsString, ""):
			pattern = toks[i+2].Text
		default:
			continue
		}
		if k, ok := regexAllowlist(pattern); ok {
			obs = append(obs, observation{pos: i, line: t.Line, construct: cRegex, kinds: k})
		}
	}

	// Letter lists: ['G', 'M', 'C'] and 'GMC'.includes(...).
	for i, t := range toks {
		switch {
		case at(i, jsPunct, "["):
			var k kindSet
			j, n := i+1, 0
			for at(j, jsString, "") && len(toks[j].Text) == 1 && k.add(toks[j].Text[0]) {
				n++
				j++
				if !at(j, jsPunct, ",") {
					break
				}
				j++
			}
			if n >= 2 && at(j, jsPunct, "]") {
				obs = append(obs, observation{pos: i, line: t.Line, construct: cLetterList, kinds: k})
			}
		case t.Kind == jsString && len(t.Text) >= 2 && at(i+1, jsPunct, ".") &&
			(at(i+2, jsIdent, "includes") || at(i+2, jsIdent, "indexOf")) && at(i+3, jsPunct, "("):
			if k, ok := lettersOf(t.Text); ok {
				obs = append(obs, observation{pos: i, line: t.Line, construct: cLetterList, kinds: k})
			}
		}
	}

	// Prefix tests and StrKey predicates, per statement.
	start := 0
	for i := 0; i <= len(toks); i++ {
		if i < len(toks) && !(toks[i].Kind == jsPunct && (toks[i].Text == ";" || toks[i].Text == "{" || toks[i].Text == "}")) {
			continue
		}
		obs = append(obs, jsStatement(toks[start:i], start)...)
		start = i + 1
	}

	// switch (...) { case ...scAddressTypeContract...: }
	named := func(name string) func(int) bool {
		return func(j int) bool {
			return toks[j].Text == name && (toks[j].Kind == jsIdent || toks[j].Kind == jsString)
		}
	}
	for i := range toks {
		sw, ok := f.switchAt(i)
		if !ok {
			continue
		}
		lo, hi := sw.bodyOpen+1, sw.bodyClose
		o := observation{
			pos: i, line: toks[i].Line, construct: cTypeSwitch, hasDefault: sw.hasDefault,
			contractCase:      f.count("scAddressTypeContract", named("scAddressTypeContract"), lo, hi) > 0,
			muxedContractCase: f.count("scAddressTypeMuxedContract", named("scAddressTypeMuxedContract"), lo, hi) > 0,
		}
		if o.contractCase {
			obs = append(obs, o)
		}
	}
	jsMarkWNearby(toks, obs)
	return obs
}

// jsWMarkers are identifiers and strings that show code handling W
// addresses.
var jsWMarkers = map[string]bool{
	"isValidMuxedContract": true, "decodeMuxedContract": true, "encodeMuxedContract": true,
	"scAddressTypeMuxedContract": true, "W": true,
}

// jsMarkWNearby sets wNearby on allowlist observations whose enclosing
// function also handles W. The enclosing function is approximated as the
// outermost brace block around the observation, skipping class bodies so a
// method counts as its own scope; at top level the whole file is the scope.
func jsMarkWNearby(toks []jsToken, obs []observation) {
	n := len(toks)
	isMarker := func(t jsToken) bool {
		return (t.Kind == jsIdent && t.Text != "W" || t.Kind == jsString) && jsWMarkers[t.Text]
	}
	markers := make([]int, n+1) // markers[i] = marker tokens before i
	for i, t := range toks {
		markers[i+1] = markers[i]
		if isMarker(t) {
			markers[i+1]++
		}
	}
	isBrace := func(i int, b string) bool { return toks[i].Kind == jsPunct && toks[i].Text == b }

	match := make([]int, n)
	var stack []int
	for i := range toks {
		match[i] = n - 1
		switch {
		case isBrace(i, "{"):
			stack = append(stack, i)
		case isBrace(i, "}") && len(stack) > 0:
			match[stack[len(stack)-1]] = i
			stack = stack[:len(stack)-1]
		}
	}
	isClassBody := func(open int) bool {
		for k := open - 1; k >= 0 && k >= open-8; k-- {
			t := toks[k]
			if t.Kind == jsIdent && t.Text == "class" {
				return true
			}
			if t.Kind == jsPunct && strings.ContainsAny(t.Text, ";{}()=") {
				return false
			}
		}
		return false
	}

	// scope[i] is the outermost non-class brace around token i, or -1 for
	// the file. firstOpen tracks the stack index of that brace, so each
	// token costs O(1) however deeply classes nest.
	scope := make([]int, n)
	stack = stack[:0]
	firstOpen := -1
	for i := range toks {
		if isBrace(i, "}") && len(stack) > 0 {
			stack = stack[:len(stack)-1]
			if firstOpen >= len(stack) {
				firstOpen = -1
			}
		}
		scope[i] = -1
		if firstOpen >= 0 {
			scope[i] = stack[firstOpen]
		}
		if isBrace(i, "{") {
			if firstOpen < 0 && !isClassBody(i) {
				firstOpen = len(stack)
			}
			stack = append(stack, i)
		}
	}

	for i := range obs {
		o := &obs[i]
		if o.construct == cTypeSwitch || o.pos < 0 || o.pos >= n {
			continue
		}
		lo, hi := 0, n-1
		if s := scope[o.pos]; s >= 0 {
			lo, hi = s, match[s]
		}
		o.wNearby = markers[hi+1]-markers[lo] > 0
	}
}

// jsStatement groups prefix tests by the value they test, and StrKey
// predicates by statement.
func jsStatement(toks []jsToken, base int) []observation {
	at := func(i int, kind jsKind, text string) bool {
		return i >= 0 && i < len(toks) && toks[i].Kind == kind && (text == "" || toks[i].Text == text)
	}
	isEq := func(i int) bool {
		return at(i, jsPunct, "===") || at(i, jsPunct, "==") || at(i, jsPunct, "!==") || at(i, jsPunct, "!=")
	}
	letterAt := func(i int) (byte, bool) {
		if at(i, jsString, "") && len(toks[i].Text) == 1 {
			return toks[i].Text[0], true
		}
		return 0, false
	}

	prefixes := map[string]kindSet{}
	firstLine := map[string]int{}
	firstPos := map[string]int{}
	var order []string
	note := func(recv string, letter byte, line, pos int) {
		k := prefixes[recv]
		if !k.add(letter) {
			return
		}
		if _, seen := firstLine[recv]; !seen {
			firstLine[recv] = line
			firstPos[recv] = base + pos
			order = append(order, recv)
		}
		prefixes[recv] = k
	}
	var preds kindSet
	predCount, predLine, predPos := 0, 0, 0

	for i := range toks {
		switch {
		// recv.startsWith('G')
		case at(i, jsPunct, ".") && at(i+1, jsIdent, "startsWith") && at(i+2, jsPunct, "(") && at(i+4, jsPunct, ")"):
			if l, ok := letterAt(i + 3); ok {
				note(jsReceiver(toks, i), l, toks[i].Line, i)
			}
		// recv[0] === 'G'
		case at(i, jsPunct, "[") && at(i+1, jsNumber, "0") && at(i+2, jsPunct, "]") && isEq(i+3):
			if l, ok := letterAt(i + 4); ok {
				note(jsReceiver(toks, i), l, toks[i].Line, i)
			}
		// recv.charAt(0) === 'G'
		case at(i, jsPunct, ".") && at(i+1, jsIdent, "charAt") && at(i+2, jsPunct, "(") && at(i+3, jsNumber, "0") && at(i+4, jsPunct, ")") && isEq(i+5):
			if l, ok := letterAt(i + 6); ok {
				note(jsReceiver(toks, i), l, toks[i].Line, i)
			}
		// 'G' === recv[0] / 'G' === recv.charAt(0)
		case isEq(i+1) && at(i+2, jsIdent, ""):
			l, ok := letterAt(i)
			if !ok {
				continue
			}
			j := i + 3
			for at(j, jsPunct, ".") && at(j+1, jsIdent, "") && !at(j+1, jsIdent, "charAt") {
				j += 2
			}
			if (at(j, jsPunct, "[") && at(j+1, jsNumber, "0") && at(j+2, jsPunct, "]")) ||
				(at(j, jsPunct, ".") && at(j+1, jsIdent, "charAt") && at(j+3, jsNumber, "0")) {
				note(jsReceiver(toks, j), l, toks[i].Line, i)
			}
		// StrKey.isValidContract(...)
		case at(i, jsIdent, "StrKey") && at(i+1, jsPunct, "."):
			if !at(i+2, jsIdent, "") {
				continue
			}
			if l, ok := jsStrKeyPredicates[toks[i+2].Text]; ok {
				if !preds.has(l) {
					predCount++
				}
				preds.add(l)
				if predLine == 0 {
					predLine, predPos = toks[i].Line, base+i
				}
			}
		}
	}

	var obs []observation
	for _, recv := range order {
		if k := prefixes[recv]; k.size() >= 2 {
			obs = append(obs, observation{pos: firstPos[recv], line: firstLine[recv], construct: cPrefixTest, kinds: k})
		}
	}
	if predCount >= 2 {
		obs = append(obs, observation{pos: predPos, line: predLine, construct: cPredicates, kinds: preds})
	}
	return obs
}

// jsReceiver renders the member chain ending just before toks[i]
// (a.b?.c), or "?" when it is not a simple chain.
func jsReceiver(toks []jsToken, i int) string {
	j := i - 1
	for j >= 0 && (toks[j].Kind == jsIdent || toks[j].Kind == jsPunct && (toks[j].Text == "." || toks[j].Text == "?.")) {
		j--
	}
	if j == i-1 {
		return "?"
	}
	var parts []string
	for _, t := range toks[j+1 : i] {
		parts = append(parts, t.Text)
	}
	r := ""
	for _, p := range parts {
		if p == "?." {
			p = "."
		}
		r += p
	}
	return r
}
