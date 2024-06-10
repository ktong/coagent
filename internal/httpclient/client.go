// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

//nolint:ireturn,wrapcheck
package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func Get[R any](ctx context.Context, path string, opts ...Option) (R, error) {
	var response R
	options := apply(opts)
	path, err := url.JoinPath(options.baseURL, path)
	if err != nil {
		return response, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return response, err
	}
	for k, v := range options.headers {
		req.Header.Set(k, v)
	}

	resp, err := options.client.Do(req)
	if err != nil {
		return response, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if err = checkStatus(resp); err != nil {
		return response, err
	}
	if err = unmarshalResponse(resp, &response); err != nil {
		return response, err
	}

	return response, nil
}

func Post[R any](ctx context.Context, path string, request any, opts ...Option) (R, error) {
	var response R
	options := apply(opts)
	path, err := url.JoinPath(options.baseURL, path)
	if err != nil {
		return response, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, path, nil)
	if err != nil {
		return response, err
	}
	if err = marshalRequest(req, request); err != nil {
		return response, err
	}
	for k, v := range options.headers {
		req.Header.Set(k, v)
	}

	resp, err := options.client.Do(req)
	if err != nil {
		return response, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if err = checkStatus(resp); err != nil {
		return response, err
	}
	if err = unmarshalResponse(resp, &response); err != nil {
		return response, err
	}

	return response, nil
}

func Delete(ctx context.Context, path string, opts ...Option) error {
	options := apply(opts)
	path, err := url.JoinPath(options.baseURL, path)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
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

	return nil
}

func marshalRequest(req *http.Request, request any) error {
	switch value := request.(type) {
	case io.Reader:
		req.Body = io.NopCloser(value)
	case string:
		req.Body = io.NopCloser(strings.NewReader(value))
	case []byte:
		req.Body = io.NopCloser(bytes.NewReader(value))
	default:
		buf := new(bytes.Buffer)
		if err := json.NewEncoder(buf).Encode(value); err != nil {
			return err
		}
		req.Body = io.NopCloser(buf)
		req.Header.Set("Content-Type", "application/json")
	}

	return nil
}

func unmarshalResponse(resp *http.Response, response any) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	switch value := response.(type) {
	case *io.Reader:
		*value = bytes.NewReader(body)
	case *[]byte:
		*value = body
	case *string:
		*value = string(body)
	default:
		if err := json.Unmarshal(body, value); err != nil {
			return err
		}
	}

	return nil
}

type StatusError struct {
	Code    int
	Message string
}

func (s *StatusError) Error() string {
	if s.Message == "" {
		s.Message = http.StatusText(s.Code)
	}

	return fmt.Sprintf("[%d] %s", s.Code, s.Message)
}

func checkStatus(resp *http.Response) error {
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusBadRequest {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)

	return &StatusError{Code: resp.StatusCode, Message: string(body)}
}
