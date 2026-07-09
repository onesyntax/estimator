package webui

import (
	"strconv"
	"strings"
)

// ParseTaskList parses a semicolon-separated task list ("Login API; Login UI")
// into its trimmed, non-empty descriptions. It backs the QA priming panel's
// Next-WBS field.
func ParseTaskList(s string) []string { return splitEntries(s) }

// ParseRiskSeeds parses "task N: description" entries separated by semicolons
// into risk seeds. It backs the QA priming panel's Next-risks field. Malformed
// entries are skipped.
func ParseRiskSeeds(s string) []RiskSeed {
	out := []RiskSeed{}
	for _, entry := range splitEntries(s) {
		number, body, ok := splitTaskEntry(entry)
		if !ok || body == "" {
			continue
		}
		out = append(out, RiskSeed{TaskNumber: number, Description: body})
	}
	return out
}

// ParseEstimateSeeds parses "task N: O/M/P" entries separated by semicolons into
// estimate seeds with a non-empty placeholder reasoning. It backs the QA priming
// panel's Next-estimates field. Malformed entries are skipped.
func ParseEstimateSeeds(s string) []EstimateSeed {
	out := []EstimateSeed{}
	for _, entry := range splitEntries(s) {
		number, body, ok := splitTaskEntry(entry)
		if !ok {
			continue
		}
		o, m, p, ok := parseTriple(body)
		if !ok {
			continue
		}
		out = append(out, EstimateSeed{TaskNumber: number, Optimistic: o, MostLikely: m, Pessimistic: p, Reasoning: "AI estimate"})
	}
	return out
}

// parseTriple parses a slash-separated "O/M/P" triple of integers. It reports
// ok only when there are exactly three parts and all parse as integers.
func parseTriple(body string) (o, m, p int, ok bool) {
	nums := strings.Split(body, "/")
	if len(nums) != 3 {
		return 0, 0, 0, false
	}
	o, ok1 := atoiTrim(nums[0])
	m, ok2 := atoiTrim(nums[1])
	p, ok3 := atoiTrim(nums[2])
	if !ok1 || !ok2 || !ok3 {
		return 0, 0, 0, false
	}
	return o, m, p, true
}

func splitEntries(s string) []string {
	parts := strings.Split(s, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// splitTaskEntry parses a "task N: body" entry into its one-based number and the
// trimmed body.
func splitTaskEntry(entry string) (number int, body string, ok bool) {
	colon := strings.IndexByte(entry, ':')
	if colon < 0 {
		return 0, "", false
	}
	label := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(entry[:colon]), "task"))
	n, err := strconv.Atoi(label)
	if err != nil {
		return 0, "", false
	}
	return n, strings.TrimSpace(entry[colon+1:]), true
}

func atoiTrim(s string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	return n, err == nil
}
