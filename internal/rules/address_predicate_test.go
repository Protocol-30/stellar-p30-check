package rules

import "testing"

func TestRegexWPrefixProof(t *testing.T) {
	for _, tc := range []struct {
		pattern, flags    string
		reject, supported bool
	}{
		{`^[GMC][A-Z2-7]{55}$`, "", true, true},
		{`^(G|M|C)[A-Z2-7]+$`, "", true, true},
		{`^(?:G|M|C)`, "", true, true},
		{`^[^W]`, "", true, true},
		{`^[gmc]`, "i", true, true},
		{`^[GMCW][A-Z2-7]+$`, "", false, true},
		{`^[w]`, "i", false, true},
		{`^W`, "", false, true},
		{`[GMC]`, "", false, true},
		{`^G|W`, "", false, true},
		{`^(G|W)`, "", false, true},
		{`^(G)?W`, "", false, true},
		{`^G*W`, "", false, true},
		{`^G*$`, "", false, true},
		{`^(W|G)?`, "", false, true},
		{`^.*G`, "", false, true},
		{`^\x57`, "", false, true},
		{`^$`, "", false, true},
		{`^(?!W)`, "", false, false},
		{`(?i)^G`, "", false, false},
		{`\A[GC]`, "", false, false},
		{`\A.*`, "", false, false},
		{`^\Q[W]\E`, "", false, false},
		{`^\zG`, "", false, false},
		{`^[GMC]`, "m", false, false},
		{`^[GMC]`, "u", false, false},
		{`[`, "", false, false},
	} {
		t.Run(tc.pattern+tc.flags, func(t *testing.T) {
			reject, supported := regexRejectsW(tc.pattern, tc.flags)
			if reject != tc.reject || supported != tc.supported {
				t.Fatalf("got(%t,%t),want(%t,%t)", reject, supported, tc.reject, tc.supported)
			}
		})
	}
}

func TestWPrefixPredicates(t *testing.T) {
	for _, tc := range []struct {
		prefix string
		want   wTruth
	}{{"", wTrue}, {"W", wTrue}, {"WA", wUnknown}, {"G", wFalse}, {"C", wFalse}, {"M", wFalse}, {"w", wFalse}} {
		if got := prefixWTruth(tc.prefix); got != tc.want {
			t.Fatalf("prefix %q: %v", tc.prefix, got)
		}
	}
	if wAnd(wUnknown, wFalse) != wFalse || wAnd(wTrue, wTrue) != wTrue || wAnd(wTrue, wUnknown) != wUnknown || wOr(wUnknown, wTrue) != wTrue || wOr(wFalse, wFalse) != wFalse || wOr(wFalse, wUnknown) != wUnknown || wNot(wUnknown) != wUnknown || wNot(wTrue) != wFalse || wNot(wFalse) != wTrue {
		t.Fatal("invalid three-valued predicate proof")
	}
}
