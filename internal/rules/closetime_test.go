package rules

import "testing"

func TestCloseTimesFixtures(t *testing.T) {
	runCases(t, CloseTimes{}, "../../testdata/rules/ts-001", []ruleCase{
		{"go-rate", []string{
			"rate.go:7 medium/medium",
			"rate.go:17 medium/medium",
			"rate.go:22 info/low",
		}, nil, "divisor can be 0"},
		{"go-safe", nil, nil, ""},
		{"js-rate", []string{"rate.js:2 medium/medium", "rate.js:6 info/low"}, nil, "divisor can be 0"},
		{"js-safe", nil, nil, ""},
	})
}
