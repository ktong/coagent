// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

type Client struct {
	host       string
	apiKey     string
	httpClient http.Client
}

func NewClient(apiKey string) Client {
	return Client{
		host:   "https://api.openai.com/v1",
		apiKey: apiKey,
		httpClient: http.Client{
			Transport: &http.Transport{
				DialContext:         (&net.Dialer{Timeout: time.Second}).DialContext,
				ForceAttemptHTTP2:   true,
				MaxIdleConns:        100, //nolint:mnd
				MaxIdleConnsPerHost: 100, //nolint:mnd
				TLSHandshakeTimeout: time.Second,
			},
		},
	}
}

func (c Client) Unary(ctx context.Context, path string, request, response any) error {
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+path, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create http request: %w", err)
	}
	c.setHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("handle http request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		body, _ = io.ReadAll(resp.Body)

		return fmt.Errorf("%d: %s", resp.StatusCode, body) //nolint:err113
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	return nil
}

func (c Client) Stream(
	ctx context.Context,
	path string,
	request any,
	eventHandler func(context.Context, Event) error,
) error {
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+path, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create http request: %w", err)
	}
	c.setHeader(req)
	req.Header.Set("Accept", "text/Type-Stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("handle http request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		body, _ = io.ReadAll(resp.Body)

		return fmt.Errorf("%d: %s", resp.StatusCode, body) //nolint:err113
	}

	reader := NewEventReader(resp.Body)
	for {
		event, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

			return fmt.Errorf("Read Type: %w", err)
		}
		if err := eventHandler(ctx, event); err != nil {
			return fmt.Errorf("handle Type: %w", err)
		}
	}
}

func (c Client) Delete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.host+path, nil)
	if err != nil {
		return fmt.Errorf("create http request: %w", err)
	}
	c.setHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("handle http request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	return nil
}

func (c Client) setHeader(req *http.Request) {
	req.Header.Set("Accept", "application/json")
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Add("Authorization", "Bearer "+c.apiKey)
	req.Header.Add("OpenAI-Beta", "assistants=v2") //nolint:canonicalheader
}
