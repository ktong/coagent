// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package client provides convenient access to the OpenAI REST API.
package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type (
	// Client provides convenient access to the [OpenAI REST API].
	//
	// To create a new Client, call [New].
	//
	// [OpenAI REST API]: https://platform.openai.com/docs/api-reference/assistants
	Client struct {
		host       string
		apiKey     string
		httpClient *http.Client
	}

	// Event represents the [stream event] returned by Client.Stream.
	//
	// [stream event]: https://platform.openai.com/docs/api-reference/assistants-streaming/events
	Event struct {
		Type string
		Data []byte
	}
)

// New creates a new Client with the given Option(s).
func New(opts ...Option) Client {
	options := &options{
		host:   "https://api.openai.com/v1",
		apiKey: os.Getenv("OPENAI_API_KEY"),
		httpClient: &http.Client{
			Transport: &http.Transport{
				DialContext:         (&net.Dialer{Timeout: time.Second}).DialContext,
				ForceAttemptHTTP2:   true,
				MaxIdleConns:        100, //nolint:mnd
				MaxIdleConnsPerHost: 100, //nolint:mnd
				TLSHandshakeTimeout: time.Second,
			},
		},
	}
	for _, opt := range opts {
		opt(options)
	}

	return Client(*options)
}

// Post sends a POST request to the given path with request body and populars response.
func (c Client) Post(ctx context.Context, path string, request, response any) error {
	buf, err := marshalBody(request)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+path, buf)
	if err != nil {
		return fmt.Errorf("create post request: %w", err)
	}
	c.setHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("handle post request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("post request response %d: %s", resp.StatusCode, body) //nolint:err113
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("unmarshal post response: %w", err)
	}

	return nil
}

func marshalBody(request any) (io.Reader, error) {
	switch v := request.(type) {
	case string:
		return strings.NewReader(v), nil
	case []byte:
		return bytes.NewReader(v), nil
	default:
		buf := new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(request)
		if err != nil {
			return nil, fmt.Errorf("marshal post request: %w", err)
		}

		return buf, nil
	}
}

// Delete sends a DELETE request to the given path.
func (c Client) Delete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.host+path, nil)
	if err != nil {
		return fmt.Errorf("create delete request: %w", err)
	}
	c.setHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("handle delete request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("delete request response %d: %s", resp.StatusCode, body) //nolint:err113
	}

	return nil
}

// UploadFile uploads a file with the given name and content to OpenAI [files] storage.
//
// [files]: https://platform.openai.com/docs/api-reference/files
func (c Client) UploadFile(ctx context.Context, name string, content io.Reader) (string, error) {
	buf := new(bytes.Buffer)

	writer := multipart.NewWriter(buf)
	if err := writer.WriteField("purpose", "assistants"); err != nil {
		return "", fmt.Errorf("write purpose field: %w", err)
	}
	fileWriter, err := writer.CreateFormFile("file", name)
	if err != nil {
		return "", fmt.Errorf("create file field: %w", err)
	}
	if _, err = io.Copy(fileWriter, content); err != nil {
		return "", fmt.Errorf("write file content: %w", err)
	}
	if err = writer.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+"/files", buf)
	if err != nil {
		return "", fmt.Errorf("create upload file request: %w", err)
	}
	req.Header.Add("Content-Type", writer.FormDataContentType())
	c.setHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("handle upload file request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)

		return "", fmt.Errorf("upload file response %d: %s", resp.StatusCode, body) //nolint:err113
	}

	var file struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&file); err != nil {
		return "", fmt.Errorf("unmarshal upload file response: %w", err)
	}

	return file.ID, nil
}

func (c Client) DownloadFile(ctx context.Context, id string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.host+"/files/"+id+"/content", nil)
	if err != nil {
		return nil, fmt.Errorf("create download file request: %w", err)
	}
	c.setHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("handle download file request: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		return nil, fmt.Errorf("download file response %d: %s", resp.StatusCode, body) //nolint:err113
	}

	return resp.Body, nil
}

// Stream sends a [streaming] request to the given path with request body.
// It calls eventHandler to process the Server-sent events.
//
// [streaming]: https://platform.openai.com/docs/api-reference/assistants-streaming
func (c Client) Stream(
	ctx context.Context,
	path string,
	request any,
	eventHandler func(context.Context, Event) error,
) error {
	buf, err := marshalBody(request)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+path, buf)
	if err != nil {
		return fmt.Errorf("create stream request: %w", err)
	}
	c.setHeader(req)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("handle stream request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("stream request response %d: %s", resp.StatusCode, body) //nolint:err113
	}

	reader := eventReader{reader: bufio.NewReader(resp.Body)}
	for {
		event, err := reader.read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

			return fmt.Errorf("read stream event: %w", err)
		}
		if err := eventHandler(ctx, event); err != nil {
			return fmt.Errorf("handle stream event: %w", err)
		}
	}
}

type eventReader struct {
	reader *bufio.Reader
}

func (stream eventReader) read() (Event, error) {
	var event Event

	for {
		rawLine, err := stream.reader.ReadBytes('\n')
		if err != nil {
			return event, fmt.Errorf("read line: %w", err)
		}
		rawLine = bytes.TrimRight(rawLine, "\r\n")
		rawLine = bytes.TrimSpace(rawLine)
		if len(rawLine) == 0 { // Delimiter of events.
			return event, nil
		}

		key, value, _ := bytes.Cut(rawLine, []byte(":"))
		value = bytes.TrimSpace(value)
		switch string(key) {
		case "event":
			event.Type = string(value)
		case "data":
			if event.Data != nil {
				event.Data = append(event.Data, '\n')
			}
			event.Data = append(event.Data, value...)
		default:
			// Ignore unknown fields and comments.
		}
	}
}

func (c Client) setHeader(req *http.Request) {
	req.Header.Set("Accept", "application/json")
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Add("Authorization", "Bearer "+c.apiKey)
	req.Header.Add("OpenAI-Beta", "assistants=v2") //nolint:canonicalheader
}
