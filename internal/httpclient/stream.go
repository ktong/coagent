// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

//nolint:wrapcheck
package httpclient

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Event struct {
	Type string
	Data []byte
}

func Stream( //nolint:cyclop
	ctx context.Context,
	path string,
	request any,
	handler func(context.Context, Event) error,
	opts ...Option,
) error {
	options := apply(opts)
	path, err := url.JoinPath(options.baseURL, path)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}
	if err = marshalRequest(req, request); err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")
	for k, v := range options.headers {
		req.Header.Set(k, v)
	}

	resp, err := options.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if err = checkStatus(resp); err != nil {
		return err
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
		if err := handler(ctx, event); err != nil {
			return fmt.Errorf("handle stream event: %w", err)
		}
	}
}

type eventReader struct {
	reader *bufio.Reader
}

func (s eventReader) read() (Event, error) {
	var event Event

	for {
		rawLine, err := s.reader.ReadBytes('\n')
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
