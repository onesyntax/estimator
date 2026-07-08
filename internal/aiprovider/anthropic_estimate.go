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
	var b strings.Builder
	b.WriteString("Produce a 3-point estimate for each task in this approved WBS by calling submit_estimates.\n\n")
	for i, t := range tasks {
		fmt.Fprintf(&b, "%d. %s\n", i+1, t.Description)
	}
	return b.String()
}

// estimatesFromMessage extracts the submitted estimates from the model response.
func estimatesFromMessage(resp *anthropic.Message) ([]wbs.EstimateAssignment, error) {
	for _, block := range resp.Content {
		if use, ok := block.AsAny().(anthropic.ToolUseBlock); ok && use.Name == estimateToolName {
			return parseEstimates(use.JSON.Input.Raw())
		}
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
