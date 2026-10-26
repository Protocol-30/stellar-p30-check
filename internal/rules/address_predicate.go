package rules

import (
	"regexp/syntax"
	"strings"
	"unicode"
)

// wTruth describes a predicate for every W-prefixed string, not a guessed
// sample address. Unknown includes predicates true for only some W strings.
// Analyzers must reject unsupported/side-effectful predicates before combining
// these values; an unknown call might mutate the parameter under analysis.
type wTruth uint8

const (
	wUnknown wTruth = iota
	wFalse
	wTrue
)

func wNot(v wTruth) wTruth {
	if v == wTrue {
		return wFalse
	}
	if v == wFalse {
		return wTrue
	}
	return wUnknown
}
func wAnd(a, b wTruth) wTruth {
	if a == wFalse || b == wFalse {
		return wFalse
	}
	if a == wTrue && b == wTrue {
		return wTrue
	}
	return wUnknown
}
func wOr(a, b wTruth) wTruth {
	if a == wTrue || b == wTrue {
		return wTrue
	}
	if a == wFalse && b == wFalse {
		return wFalse
	}
	return wUnknown
}
func prefixWTruth(prefix string) wTruth {
	if prefix == "" || prefix == "W" {
		return wTrue
	}
	if !strings.HasPrefix(prefix, "W") {
		return wFalse
	}
	return wUnknown
}

// regexRejectsW proves that an anchored literal regex cannot match any string
// beginning W. This is a conservative shared JS/Go subset; it never executes a
// target regex or assumes a fabricated W payload/checksum. Unsupported syntax
// or flags cannot support a proof.
func regexRejectsW(pattern, flags string) (reject, supported bool) {
	if flags != "" && flags != "i" {
		return false, false
	}
	// Go-only escapes must not acquire Go's anchoring/quoting semantics when
	// analyzing JavaScript legacy regexes, where they can be identity escapes.
	for _, escape := range []string{`\A`, `\z`, `\Z`, `\Q`, `\E`, `\a`} {
		if strings.Contains(pattern, escape) {
			return false, false
		}
	}
	// JS does not have Go's inline flag syntax. Noncapturing groups are shared;
	// lookarounds/backreferences and other extensions remain unsupported.
	for rest := pattern; ; {
		i := strings.Index(rest, "(?")
		if i < 0 {
			break
		}
		if !strings.HasPrefix(rest[i:], "(?:") {
			return false, false
		}
		rest = rest[i+3:]
	}
	parseFlags := syntax.Perl
	if flags == "i" {
		parseFlags |= syntax.FoldCase
	}
	re, err := syntax.Parse(pattern, parseFlags)
	if err != nil {
		return false, false
	}
	// The supported subset requires a literal leading caret in the source,
	// rather than inferring a JS anchor from a Go-specific parsed assertion.
	if !strings.HasPrefix(pattern, "^") {
		return false, true
	}
	if !regexAnchored(re) {
		return false, true
	}
	possible, nullable := regexFirstW(re)
	return !possible && !nullable, true
}

func regexAnchored(re *syntax.Regexp) bool {
	switch re.Op {
	case syntax.OpBeginText:
		return true
	case syntax.OpCapture:
		return regexAnchored(re.Sub[0])
	case syntax.OpConcat:
		for _, sub := range re.Sub {
			if sub.Op == syntax.OpEmptyMatch {
				continue
			}
			return regexAnchored(sub)
		}
	case syntax.OpAlternate:
		for _, sub := range re.Sub {
			if !regexAnchored(sub) {
				return false
			}
		}
		return len(re.Sub) > 0
	}
	return false
}

// regexFirstW returns an over-approximation of possible first W consumption,
// and whether the expression can consume zero characters. Over-approximation
// can suppress a finding, but cannot turn a W-accepting regex into a proof.
func regexFirstW(re *syntax.Regexp) (possible, nullable bool) {
	switch re.Op {
	case syntax.OpNoMatch:
		return false, false
	case syntax.OpEmptyMatch, syntax.OpBeginLine, syntax.OpEndLine, syntax.OpBeginText, syntax.OpEndText, syntax.OpWordBoundary, syntax.OpNoWordBoundary:
		return false, true
	case syntax.OpLiteral:
		if len(re.Rune) == 0 {
			return false, true
		}
		return runeCanBeW(re.Rune[0], re.Flags&syntax.FoldCase != 0), false
	case syntax.OpCharClass:
		for i := 0; i+1 < len(re.Rune); i += 2 {
			if re.Rune[i] <= 'W' && 'W' <= re.Rune[i+1] {
				return true, false
			}
			if re.Flags&syntax.FoldCase != 0 && re.Rune[i] <= 'w' && 'w' <= re.Rune[i+1] {
				return true, false
			}
		}
		return false, false
	case syntax.OpAnyChar, syntax.OpAnyCharNotNL:
		return true, false
	case syntax.OpCapture:
		return regexFirstW(re.Sub[0])
	case syntax.OpConcat:
		nullable = true
		for _, sub := range re.Sub {
			p, n := regexFirstW(sub)
			possible = possible || p
			if !n {
				nullable = false
				break
			}
		}
		return possible, nullable
	case syntax.OpAlternate:
		for _, sub := range re.Sub {
			p, n := regexFirstW(sub)
			possible = possible || p
			nullable = nullable || n
		}
		return possible, nullable
	case syntax.OpStar, syntax.OpQuest:
		p, _ := regexFirstW(re.Sub[0])
		return p, true
	case syntax.OpPlus:
		return regexFirstW(re.Sub[0])
	case syntax.OpRepeat:
		p, n := regexFirstW(re.Sub[0])
		return p, n || re.Min == 0
	default:
		return true, true
	}
}

func runeCanBeW(r rune, fold bool) bool {
	if r == 'W' {
		return true
	}
	if fold {
		for next := unicode.SimpleFold(r); next != r; next = unicode.SimpleFold(next) {
			if next == 'W' {
				return true
			}
		}
	}
	return false
}
