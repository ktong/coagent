// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package openai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ktong/assistant"
	"github.com/ktong/assistant/internal/httpclient"
)

func (e Executor) run(
	ctx context.Context,
	asst *assistant.Assistant,
	thread *assistant.Thread,
	opts []assistant.Option,
) error {
	run := &run{
		AssistantID: asst.ID,
		Stream:      true,
	}
	for _, opt := range asst.Options {
		if o, ok := opt.(funcOption); ok {
			o.Apply(run)
		}
	}
	for _, opt := range opts {
		if o, ok := opt.(funcOption); ok {
			o.Apply(run)
		}
	}
	// TODO: Add tools to run from assistant

	handler := eventHandler{
		executor:  e,
		thread:    thread,
		stream:    make(chan func() error, 1),
		functions: make(map[string]callable),
	}
	for _, tool := range asst.Tools {
		if call, ok := tool.(callable); ok {
			handler.functions[call.ID()] = call
		}
	}

	handler.stream <- func() error {
		return httpclient.Stream(ctx, "/threads/"+thread.ID+"/runs", run, handler.handle, e.clientOptions...)
	}

	return handler.run()
}

type (
	callable interface {
		ID() string
		Call(ctx context.Context, argument string) (assistant.Message, error)
	}
	eventHandler struct {
		executor  Executor
		thread    *assistant.Thread
		functions map[string]callable
		stream    chan func() error
	}
)

func (h *eventHandler) handle(ctx context.Context, event httpclient.Event) error {
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

		for _, call := range action.RequiredAction.SubmitToolOutputs.ToolCalls {
			if call.Type == "function" {
				if function := h.functions[call.Function.Name]; function != nil {
					var text string
					result, err := function.Call(ctx, call.Function.Arguments)
					if err != nil {
						text = fmt.Sprintf(`{"error": "%s"}`, err)
					} else {
						// TODO: handle non-text content
						switch content := result.Content[0].(type) {
						case assistant.Text:
							text = content.Text
						case assistant.Image[[]byte]:
						case assistant.Image[string]:
						default:
						}
					}
					outputs.ToolOutputs = append(outputs.ToolOutputs, output{
						ToolCallID: call.ID,
						Output:     text,
					})
				}
			}
		}

		h.stream <- func() error {
			return httpclient.Stream(ctx,
				"/threads/"+action.ThreadID+"/runs/"+action.ID+"/submit_tool_outputs",
				outputs, h.handle, h.executor.clientOptions...,
			)
		}
	case "thread.message.completed", "thread.message.incomplete":
		msg := struct {
			ID                string `json:"id"`
			Role              string `json:"role"`
			Status            string `json:"status"`
			IncompleteDetails struct {
				Reason string `json:"reason"`
			} `json:"incomplete_details"`
			Content []struct {
				Type string `json:"type"`
				Text struct {
					Value       string `json:"value"`
					Annotations []struct {
						Type       string `json:"type"`
						Text       string `json:"text"`
						StartIndex int    `json:"start_index"`
						EndIndex   int    `json:"end_index"`
						FilePath   struct {
							FileID string `json:"file_id"`
						} `json:"file_path"`
					} `json:"annotations"`
				} `json:"text"`
				ImageFile *imageFile `json:"image_file,omitempty"`
			} `json:"content"`
		}{}
		if err := json.Unmarshal(event.Data, &msg); err != nil {
			return fmt.Errorf("unmarshal message: %w", err)
		}

		switch msg.Status {
		case "completed":
			newMessage := assistant.Message{
				ID:   msg.ID,
				Role: assistant.Role(msg.Role),
			}
			for _, content := range msg.Content {
				switch content.Type {
				case "text":
					newMessage.Content = append(newMessage.Content, assistant.Text{
						Text: content.Text.Value,
					})
					// TODO: Handle annotations
				}
			}
			h.thread.Messages = append(h.thread.Messages, newMessage)
		case "incomplete":
			return fmt.Errorf("message incomplete: %s", msg.IncompleteDetails.Reason)
		}
	}

	return nil
}

func (h *eventHandler) run() error {
	for {
		select {
		case f := <-h.stream:
			if err := f(); err != nil {
				return err
			}
		default:
			return nil
		}
	}
}
