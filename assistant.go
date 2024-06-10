// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package assistant

import (
	"context"
	"fmt"
)

type (
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
	Assistant struct {
		ID           string
		Name         string
		Description  string
		Model        string
		Instructions string
		Tools        []Tool

		// It provides a different Executor than the default one set by SetDefaultExecutor.
		Executor Executor
		// It provides default options for all runs by this Assistant,
		// and can be overridden by options passed to Run.
		Options []Option
	}

	// Thread is a conversation session between an Assistant and a user.
	// Threads store Messages and automatically handle truncation to fit content into a model’s context.
	// Due to [Thread Locks], the same thread could not be ran by multiple goroutines simultaneously.
	//
	// If ID is empty or the thread with the given ID does not exist,
	// a new thread will be created on the server with the given information.
	// Otherwise, the thread with the given ID will be used and other fields are ignored.
	//
	// It's suggested that save the thread ID in the users' session and reuse it for the following runs
	// in the short duration, or save it into the database for the long duration.
	//
	// [Thread Locks]: https://platform.openai.com/docs/assistants/how-it-works/thread-locks
	Thread struct {
		ID       string
		Messages []Message
		Tools    []Tool
	}
)

// Run runs the given thread with the given message(s) on the given assistant.
// It returns the result according to the last message returned by the assistant.
// Both the assistant and the thread will be created on server if they do not exist,
// and can be reused for the following runs.
func Run[M string | Message | []Message, R any]( //nolint:ireturn
	ctx context.Context,
	asst *Assistant,
	thread *Thread,
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

	if err := asst.executor().Run(ctx, asst, thread, messages, opts); err != nil {
		return *new(R), fmt.Errorf("run thread with executor: %w", err)
	}

	return fromMessage[R](thread.Messages[len(thread.Messages)-1])
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

	return *defaultExecutor.Load()
}
