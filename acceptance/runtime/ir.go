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

// mutate4go-manifest-begin
// {"version":1,"tested_at":"2026-07-08T15:50:58+05:30","module_hash":"02b35b50ac501237f7559e859fe5a8b5eaa82f9f37fdf52c2658e674f5e62b62","functions":[{"id":"func/ParseFeature","name":"ParseFeature","line":33,"end_line":39,"hash":"0e074105b9a99933b6a049aa55983ab0c02d71582239ab41681af7dd36e6686f"},{"id":"func/MustRegister","name":"MustRegister","line":45,"end_line":51,"hash":"930b01d40869f93f1c3d59523a741b1a906fed546009a976b1e6fa7714315924"},{"id":"func/Registered","name":"Registered","line":54,"end_line":54,"hash":"2d455f28c42a5e075b72bbf22940961edbd208aa093830924ce95de2a85e0ab8"}]}
// mutate4go-manifest-end
