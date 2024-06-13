// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package internal

import (
	"context"
	"fmt"

	"github.com/ktong/assistant"
	"github.com/ktong/assistant/openai/httpclient"
)

type Thread struct {
	ID            string         `json:"id,omitempty"`
	Messages      []message      `json:"messages,omitempty"`
	ToolResources map[string]any `json:"tool_resources,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

func (c Client) CreateThread(ctx context.Context, thread *assistant.Thread) error {
	subject := Thread{
		Messages:      make([]message, 0, len(thread.Messages)),
		ToolResources: toToolResources(thread.Tools),
		Metadata:      thread.Metadata,
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

	if _, err := httpclient.Post[struct{}](ctx, "/threads/"+threadID+"/messages", toMessage(*msg), c...); err != nil {
		return fmt.Errorf("create message: %w", err)
	}

	return nil
}

func (c Client) GetThreadMetadata(ctx context.Context, id string) (map[string]any, error) {
	resp, err := httpclient.Get[Thread](ctx, "/threads/"+id, c...)
	if err != nil {
		return nil, fmt.Errorf("get thread: %w", err)
	}

	return resp.Metadata, nil
}

func (c Client) UpdateThreadMetadata(ctx context.Context, threadID string, metaData map[string]any) error {
	if _, err := httpclient.Post[struct{}](ctx, "/threads/"+threadID, Thread{Metadata: metaData}, c...); err != nil {
		return fmt.Errorf("update thread metadata: %w", err)
	}

	return nil
}
