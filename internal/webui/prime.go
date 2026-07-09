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
		return
	}
	o, ok1 := atoiTrim(nums[0])
	m, ok2 := atoiTrim(nums[1])
	p, ok3 := atoiTrim(nums[2])
	if !ok1 || !ok2 || !ok3 {
		return
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
	label, rest, found := strings.Cut(entry, ":")
	if !found {
		return
	}
	n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(label), "task")))
	if err != nil {
		return
	}
	return n, strings.TrimSpace(rest), true
}

func atoiTrim(s string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	return n, err == nil
}

// mutate4go-manifest-begin
// {"version":1,"tested_at":"2026-07-09T12:56:54+05:30","module_hash":"0167bedff3369650b30c4a139214fb0123777348611ee103212dad52d3ebb055","functions":[{"id":"func/ParseTaskList","name":"ParseTaskList","line":11,"end_line":11,"hash":"a132ebc198e14ae4d5f8f82f8ec5644adb463f652033af381a65c4cde55b4907"},{"id":"func/ParseRiskSeeds","name":"ParseRiskSeeds","line":16,"end_line":26,"hash":"8df55181118f2d99d482a8a1173435459003be9ba0470efd1151481a9dcf6fa7"},{"id":"func/ParseEstimateSeeds","name":"ParseEstimateSeeds","line":31,"end_line":45,"hash":"b3d245b24ada1154bfcb67f83eeb71043279bbfe2238c28064eb878573a37c72"},{"id":"func/parseTriple","name":"parseTriple","line":49,"end_line":61,"hash":"7b71ee1994be854029043aa34eafc4a042fe77893917f0f419a6d5550a86e1a3"},{"id":"func/splitEntries","name":"splitEntries","line":63,"end_line":72,"hash":"51513a0ec483fac29f94b56b778f705e00933d314a7c41b0c5a6f3a7ce9addec"},{"id":"func/splitTaskEntry","name":"splitTaskEntry","line":76,"end_line":86,"hash":"04b28cfa543b0c254ddcd8a84d7e4ea30fb946322792e66ed135407ba9ec3c2f"},{"id":"func/atoiTrim","name":"atoiTrim","line":88,"end_line":91,"hash":"8f4bae2af3a69ba7234b5760feb5dc692ac35f5271469776567c8ed610b8ed82"}]}
// mutate4go-manifest-end
