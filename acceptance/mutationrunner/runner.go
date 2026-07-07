// Package mutationrunner holds the testable behavior of the APS gherkin-mutator
// runner adapter: evaluating one mutated feature IR against the project step
// handlers and speaking the persistent-worker newline-delimited JSON protocol.
//
// The command (cmd/acceptance-mutation-runner) is a thin shell that wires Serve
// to the process stdin/stdout.
package mutationrunner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"estimation/acceptance/runtime"
	"estimation/acceptance/steps"
)

// Runner outcome strings, as defined by the APS mutator worker protocol.
const (
	OutcomeSuccess = "test_success"
	OutcomeFailure = "test_failure"
	OutcomeError   = "infrastructure_error"
)

// request is one mutation job from the mutator. Only ID and FeatureJSON are
// used; GeneratedDir and WorkDir are accepted for protocol completeness.
type request struct {
	ID           string `json:"id"`
	FeatureJSON  string `json:"feature_json"`
	GeneratedDir string `json:"generated_dir"`
	WorkDir      string `json:"work_dir"`
}

// response is one job result. Duration is nanoseconds.
type response struct {
	ID       string `json:"id"`
	Outcome  string `json:"outcome"`
	Output   string `json:"output"`
	Error    string `json:"error"`
	Duration int64  `json:"duration"`
}

// Evaluate runs the project acceptance steps against the feature IR at path and
// classifies the run: a failed execution is a detected (killed) mutation, a
// clean pass is a surviving mutation, and any inability to read or parse the IR
// is an infrastructure error.
func Evaluate(path string) (outcome, output, errText string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return OutcomeError, "", fmt.Sprintf("read feature IR %s: %v", path, err)
	}
	feature, err := runtime.ParseFeature(data)
	if err != nil {
		return OutcomeError, "", fmt.Sprintf("parse feature IR %s: %v", path, err)
	}
	var out strings.Builder
	_, fail := steps.NewRegistry().Run([]runtime.Feature{feature}, &out)
	if fail > 0 {
		return OutcomeFailure, out.String(), ""
	}
	return OutcomeSuccess, out.String(), ""
}

// handle decodes one job line and produces its JSON response line.
func handle(line string) ([]byte, error) {
	started := time.Now()
	var req request
	var resp response
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		resp = response{Outcome: OutcomeError, Error: fmt.Sprintf("decode request: %v", err)}
	} else {
		outcome, output, errText := Evaluate(req.FeatureJSON)
		resp = response{ID: req.ID, Outcome: outcome, Output: output, Error: errText}
	}
	resp.Duration = time.Since(started).Nanoseconds()
	return json.Marshal(resp)
}

// Serve is the persistent worker loop: one JSON job per input line, one JSON
// response per output line. Only protocol JSON goes to out; diagnostics belong
// on stderr.
func Serve(in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)
	writer := bufio.NewWriter(out)
	for {
		line, readErr := reader.ReadString('\n')
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			payload, err := handle(trimmed)
			if err != nil {
				return err
			}
			if _, err := writer.Write(append(payload, '\n')); err != nil {
				return err
			}
			if err := writer.Flush(); err != nil {
				return err
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				return nil
			}
			return readErr
		}
	}
}

// mutate4go-manifest-begin
// {"version":1,"tested_at":"2026-07-07T22:19:15+05:30","module_hash":"5a936ead3cdbe424b095fd5007cd22ed42162a1c292f60e85458552a7e216efe","functions":[{"id":"func/Evaluate","name":"Evaluate","line":51,"end_line":66,"hash":"a054f332d374e5dab60127081db18a915ec903b56534be974929242784d9721c"},{"id":"func/handle","name":"handle","line":69,"end_line":81,"hash":"8e872d9e400dda94e1c56aa409603f029ca494a6c6fa65be277f79204f5fbc5e"},{"id":"func/Serve","name":"Serve","line":86,"end_line":110,"hash":"61540b98888b07e1fd05e6ba7e6b78924318b8bb92b08508797ee0c4bcd88081"}]}
// mutate4go-manifest-end
