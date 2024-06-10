// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

//nolint:ireturn
package openai

import (
	"github.com/ktong/assistant"
	"github.com/ktong/assistant/internal/embedded"
)

func WithModel(model string) assistant.Option {
	return funcOption{
		fn: func(r *run) {
			r.Model = model
		},
	}
}

func WithInstructions(instructions string) assistant.Option {
	return funcOption{
		fn: func(r *run) {
			r.Instructions = instructions
		},
	}
}

func WithTemperature(temperature float32) assistant.Option {
	return funcOption{
		fn: func(r *run) {
			r.Temperature = temperature
		},
	}
}

func WithTools(tools ...assistant.Tool) assistant.Option {
	return funcOption{
		fn: func(r *run) {
			// TODO: what if tools has files?
			r.Tools = append(r.Tools, toTools(tools)...)
		},
	}
}

func WithMaxPromptTokens(maxPromptTokens int) assistant.Option {
	return funcOption{
		fn: func(r *run) {
			r.MaxPromptTokens = maxPromptTokens
		},
	}
}

func WithMaxCompletionTokens(maxCompletionTokens int) assistant.Option {
	return funcOption{
		fn: func(r *run) {
			r.MaxCompletionTokens = maxCompletionTokens
		},
	}
}

func WithParallelToolCallS(parallelToolCallS bool) assistant.Option {
	return funcOption{
		fn: func(r *run) {
			r.ParallelToolCallS = parallelToolCallS
		},
	}
}

type (
	run struct {
		AssistantID         string  `json:"assistant_id"`
		Stream              bool    `json:"stream"`
		Model               string  `json:"model,omitempty"`
		Instructions        string  `json:"instructions,omitempty"`
		Tools               []tool  `json:"tools,omitempty"`
		Temperature         float32 `json:"temperature"`
		MaxPromptTokens     int     `json:"max_prompt_tokens,omitempty"`
		MaxCompletionTokens int     `json:"max_completion_tokens,omitempty"`
		ParallelToolCallS   bool    `json:"parallel_tool_calls"`
	}
)

type funcOption struct {
	embedded.Option

	fn func(*run)
}

func (f funcOption) Apply(r *run) {
	f.fn(r)
}
