// Package aiprovider contains the environmentally-unsuitable adapters that call
// external AI services. It is isolated from the pure wbs domain so the domain
// stays testable and dependency-free.
package aiprovider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"estimation/internal/wbs"
)

// systemPrompt instructs the model to act as the estimation assistant and break
// a requirement into a WBS following the Agile INVEST framework. It performs no
// math and makes no approvals — a Tech Lead reviews and approves the output.
const systemPrompt = `You are an Estimation Assistant that breaks a software requirement document into a Work Breakdown Structure (WBS): an ordered list of concrete technical tasks.

Follow the Agile INVEST framework for every task:
- Independent: minimise overlapping dependencies between tasks.
- Negotiable: capture the essence of the work, leaving room for implementation detail.
- Valuable: each task delivers value to the client or the system.
- Estimable: each task is clear enough to be estimated.
- Small: each task is completable within a few days or a single sprint.
- Testable: completion of each task is verifiable.

Do not perform any estimation, calculation, or pricing. Return only the task list by calling the submit_wbs tool.`

const toolName = "submit_wbs"

// defaultModel is the Claude model used for generation.
const defaultModel = anthropic.ModelClaudeOpus4_8

// ErrNoTasks is returned when the model produced no usable tasks.
var ErrNoTasks = errors.New("AI returned no tasks")

// AnthropicProvider generates a WBS by calling the Anthropic Messages API. It
// implements wbs.Provider.
type AnthropicProvider struct {
	client anthropic.Client
	model  anthropic.Model
}

// New builds a provider using the given model id (empty selects the default)
// and SDK request options. With no options the client resolves credentials the
// standard way (ANTHROPIC_API_KEY, then an authenticated profile).
func New(model string, opts ...option.RequestOption) *AnthropicProvider {
	selected := anthropic.Model(model)
	if selected == "" {
		selected = defaultModel
	}
	return &AnthropicProvider{
		client: anthropic.NewClient(opts...),
		model:  selected,
	}
}

// Generate asks Claude to break the requirement into an ordered task list.
func (p *AnthropicProvider) Generate(requirement string) ([]string, error) {
	tool := anthropic.ToolParam{
		Name:        toolName,
		Description: anthropic.String("Submit the work breakdown structure as an ordered list of task descriptions."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{
				"tasks": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "string"},
					"description": "Ordered list of task descriptions following the INVEST framework.",
				},
			},
			Required: []string{"tasks"},
		},
	}

	resp, err := p.client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     p.model,
		MaxTokens: 4096,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Tools: []anthropic.ToolUnionParam{{OfTool: &tool}},
		ToolChoice: anthropic.ToolChoiceUnionParam{
			OfTool: &anthropic.ToolChoiceToolParam{Name: toolName},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(requirement)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("anthropic generate: %w", err)
	}
	return tasksFromMessage(resp)
}

// tasksFromMessage extracts the submitted task list from the model response.
func tasksFromMessage(resp *anthropic.Message) ([]string, error) {
	for _, block := range resp.Content {
		if use, ok := block.AsAny().(anthropic.ToolUseBlock); ok && use.Name == toolName {
			return parseTasks(use.JSON.Input.Raw())
		}
	}
	return nil, ErrNoTasks
}

// parseTasks decodes the {"tasks": [...]} tool input and cleans the result.
func parseTasks(raw string) ([]string, error) {
	var payload struct {
		Tasks []string `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, fmt.Errorf("anthropic: decode tasks: %w", err)
	}
	tasks := make([]string, 0, len(payload.Tasks))
	for _, t := range payload.Tasks {
		if trimmed := strings.TrimSpace(t); trimmed != "" {
			tasks = append(tasks, trimmed)
		}
	}
	if len(tasks) == 0 {
		return nil, ErrNoTasks
	}
	return tasks, nil
}

// Ensure AnthropicProvider satisfies the domain port.
var _ wbs.Provider = (*AnthropicProvider)(nil)

// mutate4go-manifest-begin
// {"version":1,"tested_at":"2026-07-07T23:30:40+05:30","module_hash":"f76ac064350e5d3804428820a671bdbafa9523b921870463b3aff03867165aa4","functions":[{"id":"func/New","name":"New","line":52,"end_line":61,"hash":"6d1dd2b619f8885ec11affda37bd1c084ac2e908ca2947f3d10dae10f03c0ea4"},{"id":"func/AnthropicProvider.Generate","name":"AnthropicProvider.Generate","line":64,"end_line":98,"hash":"5dc9150e27cdd8833a001b6fd6d6d20773f8a10ea9da42805132798b4a841d04"},{"id":"func/tasksFromMessage","name":"tasksFromMessage","line":101,"end_line":108,"hash":"0992ea5b98b83d8223eaeee56bf5d947ff3893f1417ddb77e84ad9432550af6b"},{"id":"func/parseTasks","name":"parseTasks","line":111,"end_line":128,"hash":"0d382a7b074296f6cf8849a137396dab3380273bcf52a1ac41b2ced718706b9a"}]}
// mutate4go-manifest-end
