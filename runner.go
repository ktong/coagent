// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package coagent

import (
	"context"
	"sync/atomic"
)

// Runner Loader is the interface that wraps the Run method.
//
// Run executes the provided messages using the provided agent and options.
type Runner interface {
	Run(ctx context.Context, agent Agent, messages []Message, opts []RunOption) (Message, error)
}

// SetDefaultRunner sets the default runner to be used by the Agent.
// If the provided Runner is nil, the default runner is not changed.
func SetDefaultRunner(runner Runner) {
	if runner != nil {
		defaultRunner.Store(&runner)
	}
}

var defaultRunner atomic.Pointer[Runner] //nolint:gochecknoglobals

func init() { //nolint:gochecknoinits
	SetDefaultRunner(&noopRunner{})
}

type noopRunner struct{}

func (n *noopRunner) Run(context.Context, Agent, []Message, []RunOption) (Message, error) {
	// No operation performed
	return Message{}, nil
}
