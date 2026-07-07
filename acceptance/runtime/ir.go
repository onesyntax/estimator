// Package runtime is the shared execution engine for generated acceptance
// tests. It loads the parser JSON IR, expands scenario examples, and routes
// each step to a project step handler.
package runtime

import (
	"encoding/json"
	"fmt"
)

// Step is a single Gherkin step in the JSON IR.
type Step struct {
	Keyword    string   `json:"keyword"`
	Text       string   `json:"text"`
	Parameters []string `json:"parameters"`
}

// Scenario is a scenario (or scenario outline) in the JSON IR.
type Scenario struct {
	Name     string              `json:"name"`
	Steps    []Step              `json:"steps"`
	Examples []map[string]string `json:"examples"`
}

// Feature is the top-level JSON IR object.
type Feature struct {
	Name       string     `json:"name"`
	Background []Step     `json:"background"`
	Scenarios  []Scenario `json:"scenarios"`
}

// ParseFeature decodes a JSON IR document into a Feature.
func ParseFeature(irJSON []byte) (Feature, error) {
	var f Feature
	if err := json.Unmarshal(irJSON, &f); err != nil {
		return Feature{}, fmt.Errorf("parse IR: %w", err)
	}
	return f, nil
}

var registered []Feature

// MustRegister decodes and registers a feature's JSON IR. Generated entry points
// call it from an init function; a malformed IR is a programming error.
func MustRegister(irJSON string) {
	f, err := ParseFeature([]byte(irJSON))
	if err != nil {
		panic(err)
	}
	registered = append(registered, f)
}

// Registered returns every feature registered by generated entry points.
func Registered() []Feature { return registered }
