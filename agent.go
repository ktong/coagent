// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package coagent

import "github.com/ktong/coagent/internal/embedded"

// Agent is a purpose-built AI that uses models and calls tools.
type Agent struct {
	embedded.Tool // Agent can be used as a Tool in another agent.

	Name         string
	Description  string
	Model        string
	Instructions string
	Tools        []Tool

	// It provides a different Runner than the default one set by SetDefaultRunner.
	Runner Runner
	// It provides default options for all runs by this Agent,
	// and can be overridden by options passed to Run.
	Options []RunOption
}
