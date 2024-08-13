// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package assistant

import (
	"context"
	"fmt"
)

// Assistant is a purpose-built AI that uses models and calls tools.
// It can be used to run multiple threads on different goroutines simultaneously.
//
// If ID is empty or the assistant with the given ID does not exist,
// a new assistant will be created on the server with the given information.
// Otherwise, the assistant with the given ID will be used and other fields are ignored.
//
// It's suggested that each instance has a dedicated life-time assistant,
// and should be shutdown while the instance shutdown. So that different instances
// could run with different version of the assistance.
type Assistant struct {
	ID           string
	Name         string
	Description  string
	Model        string
	Instructions string
	Tools        []Tool
	Metadata     map[string]any

	// It provides a different Executor than the default one set by SetDefaultExecutor.
	Executor Executor
	// It provides default options for all runs by this Assistant,
	// and can be overridden by options passed to Run.
	Options []Option
}

// Run runs the given thread with the given message(s) on the given assistant.
// It returns the result according to the last message returned by the assistant.
// Both the assistant and the thread will be created on server if they do not exist,
// and can be reused for the following runs.
//
// The options passed to Run will override the options on the assistant,
// but not sub assistants (as tools) of the assistant.
func (a *Assistant) Run(ctx context.Context, thread *Thread, message Message, opts ...Option) error {
	if err := a.executor().Run(ctx, a, thread, message, append(a.Options, opts...)); err != nil {
		return fmt.Errorf("run assistant with executor: %w", err)
	}

	return nil
}

func (a *Assistant) Shutdown(ctx context.Context) error {
	if err := a.executor().ShutdownAssistant(ctx, a); err != nil {
		return fmt.Errorf("shutdown assistant with executor: $%w", err)
	}

	return nil
}

func (a *Assistant) executor() Executor { //nolint:ireturn
	if a.Executor != nil {
		return a.Executor
	}

	// TODO: what if the defaultExecutor is nil?
	return *defaultExecutor.Load()
}
