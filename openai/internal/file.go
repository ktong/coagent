// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package internal

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/ktong/assistant"
	"github.com/ktong/assistant/openai/httpclient"
)

func (c Client) UploadFile(ctx context.Context, file *assistant.File) error {
	buf, contextYType, err := createMultiPartForm(file)
	if err != nil {
		return fmt.Errorf("create multipart form: %w", err)
	}

	type id struct {
		ID string `json:"id"`
	}
	resp, err := httpclient.Post[id](ctx, "/files", buf,
		append(c, httpclient.WithHeader("Reader-Type", contextYType))...,
	)
	if err != nil {
		return fmt.Errorf("upload file: %w", err)
	}
	file.ID = resp.ID

	return nil
}

func createMultiPartForm(file *assistant.File) (*bytes.Buffer, string, error) {
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)
	defer func() {
		_ = writer.Close()
	}()

	part, err := writer.CreateFormFile("file", file.Name)
	if err != nil {
		return nil, "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, file.Reader); err != nil {
		return nil, "", fmt.Errorf("copy content to form file: %w", err)
	}
	if err := writer.WriteField("purpose", "assistants"); err != nil {
		return nil, "", fmt.Errorf("write purpose field: %w", err)
	}

	return buf, writer.FormDataContentType(), nil
}

func (c Client) DownloadFile(ctx context.Context, file *assistant.File) error {
	reader, err := httpclient.Get[[]byte](ctx, "/files/"+file.ID+"/content", c...)
	if err != nil {
		return fmt.Errorf("download file: %w", err)
	}
	file.Reader = bytes.NewReader(reader)

	return nil
}
