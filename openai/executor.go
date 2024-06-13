// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package openai

import (
	"context"
	"fmt"

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
	message assistant.Message,
	opts []assistant.Option,
) error {
	if asst.ID == "" {
		//TODO: Fixing race condition since assistant can be created by another goroutine.
		if err := e.client.CreateAssistant(ctx, asst); err != nil {
			return err
		}
	}

	if thread.ID == "" {
		if err := e.client.CreateThread(ctx, thread); err != nil {
			return err
		}
	} else {
		metadata, err := e.client.GetThreadMetadata(ctx, thread.ID)
		if err != nil {
			return fmt.Errorf("get existing thread[%s]: %w", thread.ID, err)
		}
		// Load thread metadata from server.
		thread.Metadata = metadata
	}

	if err := e.client.CreateMessage(ctx, thread.ID, &message); err != nil {
		return err
	}
	thread.Messages = append(thread.Messages, message)

	return e.client.Run(ctx, asst, thread, opts)
}

func (e Executor) ShutdownAssistant(ctx context.Context, assistant *assistant.Assistant) error {
	return e.client.DeleteAssistant(ctx, assistant)
}
