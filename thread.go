// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ktong/assistant/internal"
)

func Run[M string | Message | []Message, R any]( //nolint:cyclop,funlen,ireturn
	ctx context.Context,
	message M,
	opts ...Option,
) (R, error) {
	var messages []Message
	switch msg := any(message).(type) {
	case string:
		messages = []Message{{Role: RoleUser, Content: []Content{Text{Text: msg}}}}
	case Message:
		messages = []Message{msg}
	case []Message:
		messages = msg
	}
	options := &options{
		model:  "gpt-4o",
		client: internal.NewClient(os.Getenv("OPENAI_API_KEY")),
	}
	for _, opt := range opts {
		opt(options)
	}

	var result R

	assistant := struct {
		Model        string  `json:"model"`
		Instructions string  `json:"instructions,omitempty"`
		Tools        []Tool  `json:"tools,omitempty"`
		Temperature  float32 `json:"temperature"`
	}{
		Model:        options.model,
		Instructions: options.instructions,
		Tools:        options.tools,
	}
	assistantID := struct {
		ID string `json:"id,omitempty"`
	}{}
	if err := options.client.Unary(ctx, "/assistants", assistant, &assistantID); err != nil {
		return result, fmt.Errorf("create assistant: %w", err)
	}
	defer func() {
		_ = options.client.Delete(ctx, "/assistants/"+assistantID.ID)
	}()

	thread := struct {
		ID            string         `json:"id,omitempty"`
		Messages      []Message      `json:"messages"`
		ToolResources map[string]any `json:"tool_resources,omitempty"`
	}{
		Messages: messages,
	}
	for _, tool := range options.tools {
		if codeInterpreter, ok := tool.(CodeInterpreter); ok {
			fileIDs := make([]string, 0, len(codeInterpreter.Files))
			for _, file := range codeInterpreter.Files {
				fileIDs = append(fileIDs, file.ID)
			}
			thread.ToolResources["code_interpreter"] = fileIDs
		}
	}
	if err := options.client.Unary(ctx, "/threads", thread, &thread); err != nil {
		return result, fmt.Errorf("create thread: %w", err)
	}
	defer func() {
		_ = options.client.Delete(ctx, "/threads/"+thread.ID)
	}()

	run := struct {
		AssistantID string `json:"assistant_id"`
		Stream      bool   `json:"stream"`
	}{
		AssistantID: assistantID.ID,
		Stream:      true,
	}
	handler := eventHandler[R]{
		client: options.client,
		tools:  options.tools,
	}
	if err := options.client.Stream(ctx, "/threads/"+thread.ID+"/runs", run, handler.handle); err != nil {
		return result, fmt.Errorf("run thread: %w", err)
	}

	return handler.result, nil
}

type eventHandler[R any] struct {
	client internal.Client
	tools  []Tool
	result R
}

func (h *eventHandler[R]) handle(ctx context.Context, event internal.Event) error { //nolint:cyclop,funlen
	switch event.Type {
	case "thread.run.requires_action":
		action := struct {
			ID             string `json:"id"`
			ThreadID       string `json:"thread_id"`
			RequiredAction struct {
				SubmitToolOutputs struct {
					ToolCalls []struct {
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"submit_tool_outputs"`
			} `json:"required_action"`
		}{}
		if err := json.Unmarshal(event.Data, &action); err != nil {
			return fmt.Errorf("unmarshal action: %w", err)
		}

		type output struct {
			ToolCallID string `json:"tool_call_id"`
			Output     string `json:"output"`
		}
		outputs := struct {
			ToolOutputs []output `json:"tool_outputs"`
			Stream      bool     `json:"stream"`
		}{
			ToolOutputs: make([]output, 0, len(action.RequiredAction.SubmitToolOutputs.ToolCalls)),
			Stream:      true,
		}

		type callable interface {
			name() string
			call(arguments []byte) string
		}
		for _, call := range action.RequiredAction.SubmitToolOutputs.ToolCalls {
			if call.Type == "function" {
				for _, tool := range h.tools {
					if function, ok := any(tool).(callable); ok && function.name() == call.Function.Name {
						outputs.ToolOutputs = append(outputs.ToolOutputs, output{
							ToolCallID: call.ID,
							Output:     function.call([]byte(call.Function.Arguments)),
						})

						break
					}
				}
			}
		}

		if err := h.client.Stream(ctx,
			"/threads/"+action.ThreadID+"/runs/"+action.ID+"/submit_tool_outputs",
			outputs, h.handle,
		); err != nil {
			return fmt.Errorf("submit tool outputs: %w", err)
		}
	case "thread.message.completed":
		message := struct {
			Content []struct {
				Text struct {
					Value string `json:"value"`
				} `json:"text"`
			} `json:"content"`
		}{}
		if err := json.Unmarshal(event.Data, &message); err != nil {
			return fmt.Errorf("unmarshal message: %w", err)
		}

		text := message.Content[0].Text.Value
		result := new(R)
		defer func() {
			h.result = *result
		}()

		if r, ok := any(result).(*string); ok {
			*r = text

			break
		}

		if err := json.Unmarshal([]byte(text), result); err != nil {
			return fmt.Errorf("unmarshal result: %w", err)
		}
	}

	return nil
}
