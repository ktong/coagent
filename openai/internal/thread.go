// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package internal

import (
	"context"
	"fmt"

	"github.com/ktong/assistant"
	"github.com/ktong/assistant/openai/httpclient"
)

func (c Client) CreateThread(ctx context.Context, thread *assistant.Thread) error {
	subject := struct {
		Messages      []message      `json:"messages,omitempty"`
		ToolResources map[string]any `json:"tool_resources,omitempty"`
	}{
		Messages:      make([]message, 0, len(thread.Messages)),
		ToolResources: toToolResources(thread.Tools),
	}
	for _, msg := range thread.Messages {
		subject.Messages = append(subject.Messages, toMessage(msg))
	}
	// TODO: upload files and vector store in tools
	// TODO: upload files in messages

	type id struct {
		ID string `json:"id"`
	}
	resp, err := httpclient.Post[id](ctx, "/threads", subject, c...)
	if err != nil {
		return fmt.Errorf("create thread: %w", err)
	}
	thread.ID = resp.ID
	// Clear messages to avoid double message creation on reused thread.
	thread.Messages = nil

	return nil
}

func (c Client) CreateMessage(ctx context.Context, threadID string, msg *assistant.Message) error {
	// TODO: upload files in message

	type id struct {
		ID string `json:"id"`
	}
	resp, err := httpclient.Post[id](ctx, "/threads/"+threadID+"/messages", toMessage(*msg), c...)
	if err != nil {
		return fmt.Errorf("create message: %w", err)
	}
	msg.ID = resp.ID

	return nil
}
