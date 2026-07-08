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

// riskSystemPrompt instructs the model to review an approved WBS and flag the
// risks of each task. It only identifies risks — it makes no estimates and no
// approvals; a Tech Lead reviews and edits the flagged notes.
const riskSystemPrompt = `You are a Risk Analysis Assistant. You review an approved Work Breakdown Structure (an ordered list of technical tasks) and flag the notable risks of each task: technical, security, delivery, and dependency risks that a Tech Lead should be aware of.

For each risk you identify, report the one-based number of the task it applies to and a short description of the risk. A task may have several risks or none. Do not estimate, calculate, or approve anything. Return only the risks by calling the submit_risks tool.`

const riskToolName = "submit_risks"

// ErrNoRiskSubmission is returned when the model produced no risk submission.
var ErrNoRiskSubmission = errors.New("AI returned no risk submission")

// FlagRisks asks Claude to flag the risks of each task in the approved WBS and
// returns one assignment per identified risk. A task with no risks simply
// yields no assignments; an empty submission is valid, not an error.
func (p *AnthropicProvider) FlagRisks(tasks []wbs.Task) ([]wbs.RiskAssignment, error) {
	tool := anthropic.ToolParam{
		Name:        riskToolName,
		Description: anthropic.String("Submit the flagged risks, each as the one-based task number it applies to and a short risk description."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{
				"risks": map[string]any{
					"type":        "array",
					"description": "The flagged risks across all tasks.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"taskNumber":  map[string]any{"type": "integer", "description": "One-based number of the task this risk applies to."},
							"description": map[string]any{"type": "string", "description": "Short description of the risk."},
						},
						"required": []string{"taskNumber", "description"},
					},
				},
			},
			Required: []string{"risks"},
		},
	}

	resp, err := p.client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     p.model,
		MaxTokens: 4096,
		System: []anthropic.TextBlockParam{
			{Text: riskSystemPrompt},
		},
		Tools: []anthropic.ToolUnionParam{{OfTool: &tool}},
		ToolChoice: anthropic.ToolChoiceUnionParam{
			OfTool: &anthropic.ToolChoiceToolParam{Name: riskToolName},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(taskListPrompt(tasks))),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("anthropic flag risks: %w", err)
	}
	return risksFromMessage(resp)
}

// taskListPrompt renders the approved WBS as a numbered task list for the model
// to flag risks against.
func taskListPrompt(tasks []wbs.Task) string {
	return numberedTaskList("Flag the risks of each task in this approved WBS by calling submit_risks.\n\n", tasks)
}

// risksFromMessage extracts the submitted risks from the model response.
func risksFromMessage(resp *anthropic.Message) ([]wbs.RiskAssignment, error) {
	if raw, ok := firstToolInput(resp, riskToolName); ok {
		return parseRisks(raw)
	}
	return nil, ErrNoRiskSubmission
}

// parseRisks decodes the {"risks": [...]} tool input, dropping risks with a
// blank description or a non-positive task number.
func parseRisks(raw string) ([]wbs.RiskAssignment, error) {
	var payload struct {
		Risks []struct {
			TaskNumber  int    `json:"taskNumber"`
			Description string `json:"description"`
		} `json:"risks"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, fmt.Errorf("anthropic: decode risks: %w", err)
	}
	assignments := make([]wbs.RiskAssignment, 0, len(payload.Risks))
	for _, r := range payload.Risks {
		description := strings.TrimSpace(r.Description)
		if description == "" || r.TaskNumber < 1 {
			continue
		}
		assignments = append(assignments, wbs.RiskAssignment{TaskNumber: r.TaskNumber, Description: description})
	}
	return assignments, nil
}

// mutate4go-manifest-begin
// {"version":1,"tested_at":"2026-07-08T23:59:02+05:30","module_hash":"54b705c76e738d072330ed484ec86650d9502aca13cd395e997590ade18a94d2","functions":[{"id":"func/AnthropicProvider.FlagRisks","name":"AnthropicProvider.FlagRisks","line":30,"end_line":71,"hash":"08fe7c3d3b6e5de76a7b036a84a49d2bbac0fb95120eb7739fa3e2a1f6b809f6"},{"id":"func/taskListPrompt","name":"taskListPrompt","line":75,"end_line":77,"hash":"ad4dce7aa4b7e2c52e15d99766eec17349022134f2c81e43fce5e433123c29e2"},{"id":"func/risksFromMessage","name":"risksFromMessage","line":80,"end_line":85,"hash":"8bf8d51758f2df5a34c8f5001f5da782d4511c6504fa07e7a248c591fc0d683e"},{"id":"func/parseRisks","name":"parseRisks","line":89,"end_line":108,"hash":"d477636a511fc36f1c0615ea20a1eb62b99c991f05c71b4a8168be9b21ae7997"}]}
// mutate4go-manifest-end
