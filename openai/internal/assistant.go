// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package internal

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/ktong/assistant"
	"github.com/ktong/assistant/openai/httpclient"
)

func (c Client) CreateAssistant(ctx context.Context, asst *assistant.Assistant) error {
	subject := struct {
		Name          string         `json:"name,omitempty"`
		Description   string         `json:"description,omitempty"`
		Model         string         `json:"model"`
		Instructions  string         `json:"instructions,omitempty"`
		Tools         []tool         `json:"tools,omitempty"`
		ToolResources map[string]any `json:"tool_resources,omitempty"`
		Metadata      map[string]any `json:"metadata,omitempty"`
	}{
		Name:          asst.Name,
		Description:   asst.Description,
		Model:         asst.Model,
		Instructions:  asst.Instructions,
		Tools:         make([]tool, 0, len(asst.Tools)),
		ToolResources: toToolResources(asst.Tools),
		Metadata:      asst.Metadata,
	}
	if subject.Model == "" {
		subject.Model = "gpt-4o"
	}
	for _, t := range asst.Tools {
		switch t.(type) {
		case CodeInterpreter, FileSearch:
			subject.Tools = append(subject.Tools, toTool(t))
		default:
			if tol, ok := t.(interface {
				Schema() (assistant.FunctionSchema, error)
			}); ok {
				schema, err := tol.Schema()
				if err != nil {
					return fmt.Errorf("get schema: %w", err)
				}
				subject.Tools = append(subject.Tools,
					tool{Type: "function", Function: &function{
						Name:        schema.Name,
						Description: schema.Description,
						Parameters:  schema.Parameter,
					}},
				)
			}
		}
	}
	// TODO: upload files and vector stores in tools

	type id struct {
		ID string `json:"id"`
	}
	resp, err := httpclient.Post[id](ctx, "/assistants", subject, c...)
	if err != nil {
		return fmt.Errorf("create assistant: %w", err)
	}
	asst.ID = resp.ID

	return nil
}

func (c Client) DeleteAssistant(ctx context.Context, asst *assistant.Assistant) error {
	if err := httpclient.Delete(ctx, "/assistants/"+asst.ID, c...); err != nil {
		// Ignore 404 for deleting.
		var status *httpclient.StatusError
		if !errors.As(err, &status) || status.Code == http.StatusNotFound {
			return fmt.Errorf("delete subject: %w", err)
		}
	}
	asst.ID = ""

	// TODO: delete files and vector stores in tools

	return nil
}
