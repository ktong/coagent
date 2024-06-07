// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package client_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ktong/assistant/client"
	"github.com/ktong/assistant/internal/assert"
)

func TestClient_Post(t *testing.T) {
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
					assert.Equal(t, "application/json", req.Header.Get("Accept"))
					assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
					assert.True(t, strings.HasPrefix(req.Header.Get("Authorization"), "Bearer "))
					assert.Equal(t, "assistants=v2", req.Header.Get("OpenAI-Beta")) //nolint:canonicalheader
					assert.Equal(t, "POST", req.Method)
					assert.Equal(t, "/v1/assistants", req.URL.Path)
					body, err := io.ReadAll(req.Body)
					assert.NoError(t, err)
					assert.Equal(t, "abc", string(body))

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
			error: `handle post request: Post "https://api.openai.com/v1/assistants": post error`,
		},
		{
			description: "error status code",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusNotFound,
						Body:       io.NopCloser(bytes.NewBufferString(`Page Not Found`)),
					}, nil
				}),
			},
			error: "post request response 404: Page Not Found",
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
			error: "unmarshal post response: invalid character 'a' looking for beginning of value",
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.description, func(t *testing.T) {
			subject := client.New(client.WithHTTPClient(testcase.httpClient))
			actual := assistant{}
			err := subject.Post(context.Background(), "/assistants", "abc", &actual)
			if testcase.error != "" {
				assert.EqualError(t, err, testcase.error)

				return
			}
			assert.NoError(t, err)
			assert.Equal(t, testcase.expected, actual)
		})
	}
}

func TestClient_Delete(t *testing.T) {
	testcases := []struct {
		description string
		httpClient  *http.Client
		error       string
	}{
		{
			description: "success",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					assert.Equal(t, "application/json", req.Header.Get("Accept"))
					assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
					assert.True(t, strings.HasPrefix(req.Header.Get("Authorization"), "Bearer "))
					assert.Equal(t, "assistants=v2", req.Header.Get("OpenAI-Beta")) //nolint:canonicalheader
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
			error: `handle delete request: Delete "https://api.openai.com/v1/assistants/1": delete error`,
		},
		{
			description: "error status code",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusNotFound,
						Body:       io.NopCloser(bytes.NewBufferString(`Page Not Found`)),
					}, nil
				}),
			},
			error: "delete request response 404: Page Not Found",
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.description, func(t *testing.T) {
			subject := client.New(client.WithHTTPClient(testcase.httpClient))
			err := subject.Delete(context.Background(), "/assistants/1")
			if testcase.error != "" {
				assert.EqualError(t, err, testcase.error)

				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestClient_Stream(t *testing.T) {
	testcases := []struct {
		description string
		httpClient  *http.Client
		events      []client.Event
		error       string
	}{
		{
			description: "success",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					assert.Equal(t, "text/event-stream", req.Header.Get("Accept"))
					assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
					assert.True(t, strings.HasPrefix(req.Header.Get("Authorization"), "Bearer "))
					assert.Equal(t, "assistants=v2", req.Header.Get("OpenAI-Beta")) //nolint:canonicalheader
					assert.Equal(t, "no-cache", req.Header.Get("Cache-Control"))
					assert.Equal(t, "keep-alive", req.Header.Get("Connection"))
					assert.Equal(t, "POST", req.Method)
					assert.Equal(t, "/v1/assistants", req.URL.Path)
					body, err := io.ReadAll(req.Body)
					assert.NoError(t, err)
					assert.Equal(t, "abc", string(body))

					return &http.Response{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(bytes.NewBufferString(
							"event: message\ndata: {\"id\": \"asst-123\"}\n\n" +
								"event: message\ndata: {\"id\": \"asst-456\"}\n\n",
						)),
					}, nil
				}),
			},
			events: []client.Event{
				{Type: "message", Data: []byte(`{"id": "asst-123"}`)},
				{Type: "message", Data: []byte(`{"id": "asst-456"}`)},
			},
		},
		{
			description: "error",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					return &http.Response{}, errors.New("stream error")
				}),
			},
			error: `handle stream request: Post "https://api.openai.com/v1/assistants": stream error`,
		},
		{
			description: "error status code",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusNotFound,
						Body:       io.NopCloser(bytes.NewBufferString(`Page Not Found`)),
					}, nil
				}),
			},
			error: "stream request response 404: Page Not Found",
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.description, func(t *testing.T) {
			subject := client.New(client.WithHTTPClient(testcase.httpClient))
			var index int
			err := subject.Stream(context.Background(), "/assistants", "abc",
				func(_ context.Context, event client.Event) error {
					assert.Equal(t, testcase.events[index], event)
					index++

					return nil
				},
			)
			if testcase.error != "" {
				assert.EqualError(t, err, testcase.error)

				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestClient_File(t *testing.T) {
	testcases := []struct {
		description string
		httpClient  *http.Client
		expected    string
		error       string
	}{
		{
			description: "success",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					assert.Equal(t, "application/json", req.Header.Get("Accept"))
					assert.True(t, strings.HasPrefix(req.Header.Get("Content-Type"), "multipart/form-data; boundary="))
					assert.True(t, strings.HasPrefix(req.Header.Get("Authorization"), "Bearer "))
					assert.Equal(t, "assistants=v2", req.Header.Get("OpenAI-Beta")) //nolint:canonicalheader
					assert.Equal(t, "POST", req.Method)
					assert.Equal(t, "/v1/files", req.URL.Path)
					assert.NoError(t, req.ParseMultipartForm(0))
					assert.Equal(t, "assistants", req.FormValue("purpose"))
					file, header, err := req.FormFile("file")
					assert.NoError(t, err)
					assert.Equal(t, "a.html", header.Filename)
					body, err := io.ReadAll(file)
					assert.NoError(t, err)
					assert.Equal(t, "<html></html>", string(body))

					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(`{"id": "file-123"}`)),
					}, nil
				}),
			},
			expected: "file-123",
		},
		{
			description: "error",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					return &http.Response{}, errors.New("file error")
				}),
			},
			error: `handle file request: Post "https://api.openai.com/v1/files": file error`,
		},
		{
			description: "error status code",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusNotFound,
						Body:       io.NopCloser(bytes.NewBufferString(`Page Not Found`)),
					}, nil
				}),
			},
			error: "file request response 404: Page Not Found",
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
			error: "unmarshal file response: invalid character 'a' looking for beginning of value",
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.description, func(t *testing.T) {
			subject := client.New(client.WithHTTPClient(testcase.httpClient))
			fileID, err := subject.File(context.Background(), "a.html", strings.NewReader("<html></html>"))
			if testcase.error != "" {
				assert.EqualError(t, err, testcase.error)

				return
			}
			assert.NoError(t, err)
			assert.Equal(t, testcase.expected, fileID)
		})
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
