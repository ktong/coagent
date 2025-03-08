// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package coagent

import (
	"context"
	"sync/atomic"

	"github.com/ktong/coagent/internal/embedded"
)

type (
	// Runner Loader is the interface that wraps the Run method.
	//
	// Run executes the provided messages using the provided agent and options.
	Runner interface {
		Run(ctx context.Context, agent Agent, messages []Message, opts []RunOption) (Message, error)
	}
	RunOption interface {
		embedded.RunOption
	}
)

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
