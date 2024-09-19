// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package coagent

// Agent is a purpose-built AI that uses models and calls tools.
//
// It's suggested that each instance has a dedicated life-time agent,
// and should be shutdown while the instance shutdown. So that different instances
// could run with different version of the assistance.
type Agent struct {
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
