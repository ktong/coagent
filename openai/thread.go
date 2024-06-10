// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package openai

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ktong/assistant"
	"github.com/ktong/assistant/internal/httpclient"
)

func (e Executor) createThread(ctx context.Context, thread *assistant.Thread) error {
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
	resp, err := httpclient.Post[id](ctx, "/threads", subject, e.clientOptions...)
	if err != nil {
		return fmt.Errorf("create thread: %w", err)
	}
	thread.ID = resp.ID

	messages, err := e.listMessages(ctx, thread.ID)
	if err != nil {
		return err
	}
	for i, msg := range messages {
		thread.Messages[i].ID = msg
	}

	return nil
}

func (e Executor) createMessage(ctx context.Context, threadID string, msg *assistant.Message) error {
	// TODO: upload files in message

	type id struct {
		ID string `json:"id"`
	}
	resp, err := httpclient.Post[id](ctx, "/threads/"+threadID+"/messages", toMessage(*msg), e.clientOptions...)
	if err != nil {
		return fmt.Errorf("create message: %w", err)
	}
	msg.ID = resp.ID

	return nil
}

func (e Executor) listMessages(ctx context.Context, threadID string) ([]string, error) {
	type messages struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	resp, err := httpclient.Get[messages](ctx, "/threads/"+threadID+"/messages", e.clientOptions...)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}

	ids := make([]string, 0, len(resp.Data))
	for _, m := range resp.Data {
		ids = append(ids, m.ID)
	}

	return ids, nil
}

func toMessage(m assistant.Message) message {
	msg := message{
		Role: string(m.Role),
	}
	for _, c := range m.Content {
		switch cont := c.(type) {
		case assistant.Text:
			msg.Content = append(msg.Content, content{Type: "text", Text: cont.Text})
		case assistant.Image[string]:
			parsedURL, _ := url.Parse(cont.URL)
			switch parsedURL.Scheme {
			case "", "file":
				msg.Content = append(msg.Content,
					content{Type: "image_file", ImageFile: &imageFile{FileID: parsedURL.Path, Detail: string(cont.Detail)}},
				)
			default:
				msg.Content = append(msg.Content,
					content{Type: "image_url", ImageUrl: &imageURL{URL: cont.URL, Detail: string(cont.Detail)}})
			}
		case assistant.Image[[]byte]:
			mime := http.DetectContentType(cont.URL)
			switch mime {
			case "image/gif", "image/jpeg", "image/pjpeg":
				maxEncLen := base64.StdEncoding.EncodedLen(len(cont.URL))
				buf := make([]byte, maxEncLen) //nolint:makezero
				base64.StdEncoding.Encode(buf, cont.URL)
				u := fmt.Sprintf("Data:%s;base64,%s", mime, buf)

				msg.Content = append(msg.Content,
					content{Type: "image_url", ImageUrl: &imageURL{URL: u, Detail: string(cont.Detail)}},
				)
			}
		case assistant.Attachment:
			msg.Attachments = append(msg.Attachments, attachment{FileID: cont.File.ID, Tools: toTools(cont.For)})
		}
	}

	return msg
}
