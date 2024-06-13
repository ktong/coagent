// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package httpclient_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/ktong/assistant/internal/assert"
	"github.com/ktong/assistant/openai/httpclient"
)

func TestGet(t *testing.T) {
	type assistant struct {
		ID string `json:"id"`
	}

	testcases := []struct {
		description string
		httpClient  *http.Client
		expected    assistant
		error       string
	}{
		{
			description: "success",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					assert.Equal(t, "GET", req.Method)
					assert.Equal(t, "/v1/assistants/asst-123", req.URL.Path)

					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(`{"id": "asst-123"}`)),
					}, nil
				}),
			},
			expected: assistant{
				ID: "asst-123",
			},
		},
		{
			description: "error",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					return &http.Response{}, errors.New("get error")
				}),
			},
			error: `Get "https://api.openai.com/v1/assistants/asst-123": get error`,
		},
		{
			description: "error status code",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusNotFound,
						Body:       io.NopCloser(bytes.NewBufferString(`Assistant Not Found`)),
					}, nil
				}),
			},
			error: "[404] Assistant Not Found",
		},
		{
			description: "error unmarshal",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(`asst-123`)),
					}, nil
				}),
			},
			error: "invalid character 'a' looking for beginning of value",
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.description, func(t *testing.T) {
			actual, err := httpclient.Get[assistant](
				context.Background(),
				"/assistants/asst-123",
				httpclient.WithHTTPClient(testcase.httpClient),
				httpclient.WithBaseURL("https://api.openai.com/v1"),
			)
			if testcase.error != "" {
				assert.EqualError(t, err, testcase.error)

				return
			}
			assert.NoError(t, err)
			assert.Equal(t, testcase.expected, actual)
		})
	}
}

func TestPost(t *testing.T) {
	type assistant struct {
		ID   string `json:"id,omitempty"`
		Name string `json:"name"`
	}

	testcases := []struct {
		description string
		httpClient  *http.Client
		expected    assistant
		error       string
	}{
		{
			description: "success",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					assert.Equal(t, "application/json", req.Header.Get("Reader-Type"))
					assert.Equal(t, "POST", req.Method)
					assert.Equal(t, "/v1/assistants", req.URL.Path)
					body, err := io.ReadAll(req.Body)
					assert.NoError(t, err)
					assert.Equal(t, `{"name":"abc"}`+"\n", string(body))

					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(`{"id": "asst-123"}`)),
					}, nil
				}),
			},
			expected: assistant{
				ID: "asst-123",
			},
		},
		{
			description: "error",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					return &http.Response{}, errors.New("post error")
				}),
			},
			error: `Post "https://api.openai.com/v1/assistants": post error`,
		},
		{
			description: "error status code",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusNotFound,
						Body:       io.NopCloser(bytes.NewBufferString(`Assistant Not Found`)),
					}, nil
				}),
			},
			error: "[404] Assistant Not Found",
		},
		{
			description: "error unmarshal",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(`asst-123`)),
					}, nil
				}),
			},
			error: "invalid character 'a' looking for beginning of value",
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.description, func(t *testing.T) {
			actual, err := httpclient.Post[assistant](
				context.Background(),
				"/assistants",
				assistant{Name: "abc"},
				httpclient.WithHTTPClient(testcase.httpClient),
				httpclient.WithBaseURL("https://api.openai.com/v1"),
			)
			if testcase.error != "" {
				assert.EqualError(t, err, testcase.error)

				return
			}
			assert.NoError(t, err)
			assert.Equal(t, testcase.expected, actual)
		})
	}
}

func TestDelete(t *testing.T) {
	testcases := []struct {
		description string
		httpClient  *http.Client
		error       string
	}{
		{
			description: "success",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					assert.Equal(t, "DELETE", req.Method)
					assert.Equal(t, "/v1/assistants/1", req.URL.Path)

					return &http.Response{StatusCode: http.StatusOK}, nil
				}),
			},
		},
		{
			description: "error",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					return &http.Response{}, errors.New("delete error")
				}),
			},
			error: `Delete "https://api.openai.com/v1/assistants/1": delete error`,
		},
		{
			description: "error status code",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusNotFound,
						Body:       io.NopCloser(bytes.NewBufferString(`Assistant Not Found`)),
					}, nil
				}),
			},
			error: "[404] Assistant Not Found",
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.description, func(t *testing.T) {
			err := httpclient.Delete(
				context.Background(),
				"/assistants/1",
				httpclient.WithHTTPClient(testcase.httpClient),
				httpclient.WithBaseURL("https://api.openai.com/v1"),
			)
			if testcase.error != "" {
				assert.EqualError(t, err, testcase.error)

				return
			}
			assert.NoError(t, err)
		})
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
