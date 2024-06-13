// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package openai

import (
	"context"

	"github.com/ktong/assistant"
	"github.com/ktong/assistant/openai/internal"
)

var _ assistant.Executor = (*Executor)(nil)

type Executor struct {
	client internal.Client
}

func NewExecutor(opts ...Option) Executor {
	return Executor{client: internal.NewClient(opts...)}
}

func (e Executor) Run(
	ctx context.Context,
	asst *assistant.Assistant,
	thread *assistant.Thread,
	opts []assistant.Option,
) (assistant.Message, error) {
	if asst.ID == "" {
		//TODO: Fixing race condition since assistant can be created by another goroutine.
		if err := e.client.CreateAssistant(ctx, asst); err != nil {
			return assistant.Message{}, err
		}
	}

	if thread.ID == "" {
		if err := e.client.CreateThread(ctx, thread); err != nil {
			return assistant.Message{}, err
		}
	} else {
		// TODO: Check if thread is running by other goroutine.
		for i := range thread.Messages {
			if err := e.client.CreateMessage(ctx, thread.ID, &thread.Messages[i]); err != nil {
				return assistant.Message{}, err
			}
		}
	}

	return e.client.Run(ctx, asst, thread, opts)
}

func (e Executor) ShutdownAssistant(ctx context.Context, assistant *assistant.Assistant) error {
	return e.client.DeleteAssistant(ctx, assistant)
}
