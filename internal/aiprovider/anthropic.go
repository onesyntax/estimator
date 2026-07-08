// Package aiprovider contains the environmentally-unsuitable adapters that call
// external AI services. It is isolated from the pure wbs domain so the domain
// stays testable and dependency-free.
package aiprovider

import (
	"context"
	"encoding/base64"
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

// Generate asks Claude to break the requirement into an ordered task list. A
// text requirement is sent as a text block; a PDF requirement is sent as a
// native document block so Claude reads the PDF directly — the domain does no
// PDF text extraction.
func (p *AnthropicProvider) Generate(req wbs.Requirement) ([]string, error) {
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
			anthropic.NewUserMessage(requirementBlocks(req)...),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("anthropic generate: %w", err)
	}
	return tasksFromMessage(resp)
}

// requirementBlocks renders a requirement as the content blocks of the user
// message: a base64 PDF document block (followed by a short instruction) for a
// PDF requirement, or a single text block otherwise. The document block precedes
// the text block, as the API expects.
func requirementBlocks(req wbs.Requirement) []anthropic.ContentBlockParamUnion {
	if len(req.PDF) > 0 {
		data := base64.StdEncoding.EncodeToString(req.PDF)
		return []anthropic.ContentBlockParamUnion{
			anthropic.NewDocumentBlock(anthropic.Base64PDFSourceParam{Data: data}),
			anthropic.NewTextBlock("Break this requirement document into a WBS by calling submit_wbs."),
		}
	}
	return []anthropic.ContentBlockParamUnion{anthropic.NewTextBlock(req.Text)}
}

// tasksFromMessage extracts the submitted task list from the model response.
func tasksFromMessage(resp *anthropic.Message) ([]string, error) {
	if raw, ok := firstToolInput(resp, toolName); ok {
		return parseTasks(raw)
	}
	return nil, ErrNoTasks
}

// numberedTaskList renders intro text followed by the tasks as a one-based
// numbered list — the prompt shape the risk and estimate requests share.
func numberedTaskList(intro string, tasks []wbs.Task) string {
	var b strings.Builder
	b.WriteString(intro)
	for i, t := range tasks {
		fmt.Fprintf(&b, "%d. %s\n", i+1, t.Description)
	}
	return b.String()
}

// firstToolInput returns the raw JSON input of the first tool_use block that
// calls toolName, and whether such a block was present. Each provider operation
// extracts its forced tool call this way.
func firstToolInput(resp *anthropic.Message, toolName string) (string, bool) {
	for _, block := range resp.Content {
		if use, ok := block.AsAny().(anthropic.ToolUseBlock); ok && use.Name == toolName {
			return use.JSON.Input.Raw(), true
		}
	}
	return "", false
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
// {"version":1,"tested_at":"2026-07-08T22:24:13+05:30","module_hash":"cde3f69147a9477c4f4e620f810c249656a0c1a7a9797e1976b6eb541011f152","functions":[{"id":"func/New","name":"New","line":53,"end_line":62,"hash":"6d1dd2b619f8885ec11affda37bd1c084ac2e908ca2947f3d10dae10f03c0ea4"},{"id":"func/AnthropicProvider.Generate","name":"AnthropicProvider.Generate","line":68,"end_line":102,"hash":"0196c2dcd7f66fc65a3612559443fbddc1bc1141a53744e1bc2d39f070b3f802"},{"id":"func/requirementBlocks","name":"requirementBlocks","line":108,"end_line":117,"hash":"33e3f028e93edb7c54cf438bc0e5737ec913aa9ebebc8e795e9a32f847e44328"},{"id":"func/tasksFromMessage","name":"tasksFromMessage","line":120,"end_line":125,"hash":"0b3fddf4433479847dc17259929bfef72091d6532dc7bd0f85c669da9930bba8"},{"id":"func/numberedTaskList","name":"numberedTaskList","line":129,"end_line":136,"hash":"7eb40c18c8945af4dd3ade642c2f7355a04b61c1fe8c735bbe4beb1c6df04ddd"},{"id":"func/firstToolInput","name":"firstToolInput","line":141,"end_line":148,"hash":"9d43374fa1d5bf45aa5bc81ec0876adda426ebdf7c98dac9fe194ceab01fc0ec"},{"id":"func/parseTasks","name":"parseTasks","line":151,"end_line":168,"hash":"0d382a7b074296f6cf8849a137396dab3380273bcf52a1ac41b2ced718706b9a"}]}
// mutate4go-manifest-end
