//go:build race

package rules

// raceEnabled is true when tests run under the race detector, which slows
// execution several times; timing budgets are scaled accordingly.
const raceEnabled = true
