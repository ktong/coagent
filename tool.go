// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package coagent

import "github.com/ktong/coagent/internal/embedded"

type Tool interface {
	embedded.Tool
}

// Function calling allows you to describe functions to LLM models
// that can call them along with their arguments.
type Function struct {
	embedded.Tool

	// The name of the function to be called.
	Name string
	// A description of what the function does, used by the model to choose when and how to call the function.
	Description string

	// The real function attached to the tool.
	Function any
}
