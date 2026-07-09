package aiprovider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"

	"estimation/internal/wbs"
)

// estimateSystemPrompt instructs the model to produce a 3-point estimate for
// each task of an approved WBS. It suggests values only — a Tech Lead reviews,
// may override, and approves the set.
const estimateSystemPrompt = `You are an Estimation Assistant. You review an approved Work Breakdown Structure (an ordered list of technical tasks) and produce a 3-point estimate for each task: an optimistic, a most-likely, and a pessimistic value, plus one reasoning that justifies all three together.

Use the planning-poker Fibonacci scale {0, 1, 2, 3, 5, 8, 13, 20, 40, 100} for every value, and keep each triple strictly increasing (optimistic < most likely < pessimistic). Do not approve anything. Return only the estimates by calling the submit_estimates tool.`

const estimateToolName = "submit_estimates"

// ErrNoEstimateSubmission is returned when the model produced no estimate
// submission.
var ErrNoEstimateSubmission = errors.New("AI returned no estimate submission")

// Estimate asks Claude to produce a 3-point estimate for each task of the
// approved WBS and returns one assignment per task it estimates. Assignments
// for non-positive task numbers are dropped.
func (p *AnthropicProvider) Estimate(tasks []wbs.Task) ([]wbs.EstimateAssignment, error) {
	tool := anthropic.ToolParam{
		Name:        estimateToolName,
		Description: anthropic.String("Submit the 3-point estimates, one per task, each with the task number, optimistic/mostLikely/pessimistic values, and a reasoning."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{
				"estimates": map[string]any{
					"type":        "array",
					"description": "The 3-point estimates across all tasks.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"taskNumber":  map[string]any{"type": "integer", "description": "One-based number of the task this estimate applies to."},
							"optimistic":  map[string]any{"type": "integer", "description": "Optimistic value on the Fibonacci scale."},
							"mostLikely":  map[string]any{"type": "integer", "description": "Most-likely value on the Fibonacci scale."},
							"pessimistic": map[string]any{"type": "integer", "description": "Pessimistic value on the Fibonacci scale."},
							"reasoning":   map[string]any{"type": "string", "description": "One reasoning justifying all three points."},
						},
						"required": []string{"taskNumber", "optimistic", "mostLikely", "pessimistic", "reasoning"},
					},
				},
			},
			Required: []string{"estimates"},
		},
	}

	resp, err := p.client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     p.model,
		MaxTokens: 4096,
		System: []anthropic.TextBlockParam{
			{Text: estimateSystemPrompt},
		},
		Tools: []anthropic.ToolUnionParam{{OfTool: &tool}},
		ToolChoice: anthropic.ToolChoiceUnionParam{
			OfTool: &anthropic.ToolChoiceToolParam{Name: estimateToolName},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(estimateTaskListPrompt(tasks))),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("anthropic estimate: %w", err)
	}
	return estimatesFromMessage(resp)
}

// estimateTaskListPrompt renders the approved WBS as a numbered task list for
// the model to estimate.
func estimateTaskListPrompt(tasks []wbs.Task) string {
	return numberedTaskList("Produce a 3-point estimate for each task in this approved WBS by calling submit_estimates.\n\n", tasks)
}

// estimatesFromMessage extracts the submitted estimates from the model response.
func estimatesFromMessage(resp *anthropic.Message) ([]wbs.EstimateAssignment, error) {
	if raw, ok := firstToolInput(resp, estimateToolName); ok {
		return parseEstimates(raw)
	}
	return nil, ErrNoEstimateSubmission
}

// parseEstimates decodes the {"estimates": [...]} tool input, dropping estimates
// with a non-positive task number. Value and ordering validation is the domain's
// responsibility.
func parseEstimates(raw string) ([]wbs.EstimateAssignment, error) {
	var payload struct {
		Estimates []struct {
			TaskNumber  int    `json:"taskNumber"`
			Optimistic  int    `json:"optimistic"`
			MostLikely  int    `json:"mostLikely"`
			Pessimistic int    `json:"pessimistic"`
			Reasoning   string `json:"reasoning"`
		} `json:"estimates"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, fmt.Errorf("anthropic: decode estimates: %w", err)
	}
	assignments := make([]wbs.EstimateAssignment, 0, len(payload.Estimates))
	for _, e := range payload.Estimates {
		if e.TaskNumber < 1 {
			continue
		}
		assignments = append(assignments, wbs.EstimateAssignment{
			TaskNumber:  e.TaskNumber,
			Optimistic:  e.Optimistic,
			MostLikely:  e.MostLikely,
			Pessimistic: e.Pessimistic,
			Reasoning:   strings.TrimSpace(e.Reasoning),
		})
	}
	return assignments, nil
}

// mutate4go-manifest-begin
// {"version":1,"tested_at":"2026-07-09T12:56:43+05:30","module_hash":"bd0fdb560a3130e9ee5ddeb10232dc9393da5d64f59b360ab698328af38ec611","functions":[{"id":"func/AnthropicProvider.Estimate","name":"AnthropicProvider.Estimate","line":31,"end_line":75,"hash":"b3dd9b3e9b6e309ebd39b599c0cf1aa812cee54e93981d1f8a04ba29ae89a597"},{"id":"func/estimateTaskListPrompt","name":"estimateTaskListPrompt","line":79,"end_line":81,"hash":"06626a074e17f86684afd94ff25ca962d22493f95ee3c00fa831fcf33b057fb1"},{"id":"func/estimatesFromMessage","name":"estimatesFromMessage","line":84,"end_line":89,"hash":"ba910783927df70e78ef35e653455da7fc643024692858f8de073f6b9f6a40e3"},{"id":"func/parseEstimates","name":"parseEstimates","line":94,"end_line":121,"hash":"b4759564d5f5ede383e6c33917b76dec6edceb0f2f2db228cbec2a51793a27ba"}]}
// mutate4go-manifest-end
