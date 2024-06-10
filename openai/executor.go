// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package openai

import (
	"context"
	"os"

	"github.com/ktong/assistant"
	"github.com/ktong/assistant/internal/httpclient"
)

var _ assistant.Executor = (*Executor)(nil)

type Executor struct {
	clientOptions []httpclient.Option
}

func NewExecutor(opts ...httpclient.Option) Executor {
	return Executor{clientOptions: append([]httpclient.Option{
		httpclient.WithBaseURL("https://api.openai.com/v1"),
		httpclient.WithHeader("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY")),
		httpclient.WithHeader("OpenAI-Beta", "assistants=v2"),
	}, opts...)}
}

func (e Executor) Run(
	ctx context.Context,
	assistant *assistant.Assistant,
	thread *assistant.Thread,
	messages []assistant.Message,
	opts []assistant.Option,
) error {
	if assistant.ID == "" {
		//TODO: Fixing race condition since assistant can be created by another goroutine.
		if err := e.createAssistant(ctx, assistant); err != nil {
			return err
		}
	}

	thread.Messages = append(thread.Messages, messages...)
	if thread.ID == "" {
		if err := e.createThread(ctx, thread); err != nil {
			return err
		}
	} else {
		// TODO: Check if thread is running by other goroutine.
		for i := range messages {
			if err := e.createMessage(ctx, thread.ID, &messages[i]); err != nil {
				return err
			}
		}
	}

	return e.run(ctx, assistant, thread, opts)
}

func (e Executor) ShutdownAssistant(ctx context.Context, assistant *assistant.Assistant) error {
	return e.deleteAssistant(ctx, assistant)
}
