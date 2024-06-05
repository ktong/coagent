// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package assistant

import "github.com/ktong/assistant/internal"

func WithModel(model string) Option {
	return func(options *options) {
		options.model = model
	}
}

func WithInstructions(instructions string) Option {
	return func(options *options) {
		options.instructions = instructions
	}
}

func WithTool(tools ...Tool) Option {
	return func(options *options) {
		options.tools = append(options.tools, tools...)
	}
}

type (
	Option  func(*options)
	options struct {
		model        string
		instructions string
		tools        []Tool
		client       internal.Client
	}
)
