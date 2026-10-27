package rules

import "strings"

// jsKind is a JavaScript/TypeScript token class.
type jsKind int

const (
	jsIdent jsKind = iota
	jsPunct
	jsString   // '...' or "..."; Text is the decoded value
	jsTemplate // `...`; Text is the raw body
	jsRegex    // /.../flags; Text is the pattern, Flags the flags
	jsNumber
)

type jsToken struct {
	Kind  jsKind
	Text  string
	Flags string
	Line  int
}

var jsRegexKeywords = map[string]bool{
	"return": true, "typeof": true, "instanceof": true, "in": true, "of": true,
	"new": true, "delete": true, "void": true, "throw": true, "case": true,
	"do": true, "else": true, "yield": true, "await": true,
}

var jsMultiPunct = []string{"===", "!==", "...", "==", "!=", "||", "&&", "??", "?.", "=>", "<=", ">=", "++", "--"}

// jsTokenize splits JS/TS/JSX source into tokens, dropping whitespace and
// comments. It is a lexer, not a parser: it never fails. Text it cannot
// classify (such as an apostrophe in JSX text) becomes a punctuation token,
// so one odd construct cannot hide the rest of the file.
func jsTokenize(src []byte) []jsToken {
	s := string(src)
	var toks []jsToken
	line := 1
	regexOK := func() bool {
		if len(toks) == 0 {
			return true
		}
		prev := toks[len(toks)-1]
		switch prev.Kind {
		case jsIdent:
			return jsRegexKeywords[prev.Text]
		case jsPunct:
			return prev.Text != ")" && prev.Text != "]" && prev.Text != "}"
		}
		return false
	}
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '\n':
			line++
			i++
		case c == ' ' || c == '\t' || c == '\r' || c == '\f' || c == '\v':
			i++
		case strings.HasPrefix(s[i:], "//"):
			for i < len(s) && s[i] != '\n' {
				i++
			}
		case strings.HasPrefix(s[i:], "/*"):
			end := strings.Index(s[i+2:], "*/")
			if end < 0 {
				end = len(s) - i - 2
			}
			line += strings.Count(s[i:i+2+end], "\n")
			i += end + 4
		case c == '\'' || c == '"':
			val, n, ok := jsStringLit(s[i:], c)
			if !ok {
				toks = append(toks, jsToken{Kind: jsPunct, Text: string(c), Line: line})
				i++
				continue
			}
			toks = append(toks, jsToken{Kind: jsString, Text: val, Line: line})
			line += strings.Count(s[i:i+n], "\n")
			i += n
		case c == '`':
			n := jsTemplateLen(s[i:])
			body := s[i+1 : i+n]
			if n >= 2 && s[i+n-1] == '`' {
				body = s[i+1 : i+n-1] // drop the closing backtick
			}
			toks = append(toks, jsToken{Kind: jsTemplate, Text: body, Line: line})
			line += strings.Count(s[i:i+n], "\n")
			i += n
		case c == '/' && regexOK():
			body, flags, n, ok := jsRegexLit(s[i:])
			if !ok {
				toks = append(toks, jsToken{Kind: jsPunct, Text: "/", Line: line})
				i++
				continue
			}
			toks = append(toks, jsToken{Kind: jsRegex, Text: body, Flags: flags, Line: line})
			i += n
		case isIdentStart(c):
			j := i + 1
			for j < len(s) && isIdentPart(s[j]) {
				j++
			}
			toks = append(toks, jsToken{Kind: jsIdent, Text: s[i:j], Line: line})
			i = j
		case c >= '0' && c <= '9':
			j := i + 1
			for j < len(s) && (isIdentPart(s[j]) || s[j] == '.') {
				j++
			}
			toks = append(toks, jsToken{Kind: jsNumber, Text: s[i:j], Line: line})
			i = j
		default:
			p := string(c)
			for _, m := range jsMultiPunct {
				if strings.HasPrefix(s[i:], m) {
					p = m
					break
				}
			}
			toks = append(toks, jsToken{Kind: jsPunct, Text: p, Line: line})
			i += len(p)
		}
	}
	return toks
}

func isIdentStart(c byte) bool {
	return c == '_' || c == '$' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= 0x80
}

func isIdentPart(c byte) bool { return isIdentStart(c) || c >= '0' && c <= '9' }

// jsStringLit decodes a quoted string starting at s[0]. A string may not
// contain an unescaped newline; ok is false if it does or never closes.
func jsStringLit(s string, quote byte) (val string, n int, ok bool) {
	var b strings.Builder
	for i := 1; i < len(s); i++ {
		switch c := s[i]; c {
		case quote:
			return b.String(), i + 1, true
		case '\n':
			return "", 0, false
		case '\\':
			if i+1 >= len(s) {
				return "", 0, false
			}
			i++
			switch e := s[i]; e {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case '\n': // line continuation
			default:
				b.WriteByte(e)
			}
		default:
			b.WriteByte(c)
		}
	}
	return "", 0, false
}

// jsTemplateLen returns the length of a template literal starting at s[0],
// skipping over ${...} substitutions. An unterminated template runs to the
// end of the input.
func jsTemplateLen(s string) int {
	depth := 0
	for i := 1; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '$':
			if depth == 0 && i+1 < len(s) && s[i+1] == '{' {
				depth = 1
				i++
			}
		case '{':
			if depth > 0 {
				depth++
			}
		case '}':
			if depth > 0 {
				depth--
			}
		case '`':
			if depth == 0 {
				return i + 1
			}
		}
	}
	return len(s)
}

// jsRegexLit reads a regex literal starting at s[0] == '/'.
func jsRegexLit(s string) (body, flags string, n int, ok bool) {
	inClass := false
	for i := 1; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '\n':
			return "", "", 0, false
		case '[':
			inClass = true
		case ']':
			inClass = false
		case '/':
			if inClass {
				continue
			}
			j := i + 1
			for j < len(s) && (s[j] >= 'a' && s[j] <= 'z') {
				j++
			}
			if i == 1 {
				return "", "", 0, false // "//" is a comment, handled earlier
			}
			return s[1:i], s[i+1 : j], j, true
		}
	}
	return "", "", 0, false
}
