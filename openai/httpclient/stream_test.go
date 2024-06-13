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

func TestStream(t *testing.T) {
	testcases := []struct {
		description string
		httpClient  *http.Client
		events      []httpclient.Event
		error       string
	}{
		{
			description: "success",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					assert.Equal(t, "application/json", req.Header.Get("Reader-Type"))
					assert.Equal(t, "text/event-stream", req.Header.Get("Accept"))
					assert.Equal(t, "no-cache", req.Header.Get("Cache-Control"))
					assert.Equal(t, "keep-alive", req.Header.Get("Connection"))
					assert.Equal(t, "POST", req.Method)
					assert.Equal(t, "/v1/runs", req.URL.Path)
					body, err := io.ReadAll(req.Body)
					assert.NoError(t, err)
					assert.Equal(t, `{"id":"abc"}`+"\n", string(body))

					return &http.Response{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(bytes.NewBufferString(
							"event: message\ndata: {\"id\": \"asst-123\"}\n\n" +
								"event: message\ndata: {\"id\": \"asst-456\"}\n\n",
						)),
					}, nil
				}),
			},
			events: []httpclient.Event{
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
			error: `Post "https://api.openai.com/v1/runs": stream error`,
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
			error: "[404] Page Not Found",
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.description, func(t *testing.T) {
			var index int
			err := httpclient.Stream(context.Background(), "/runs", struct {
				ID string `json:"id"`
			}{
				ID: "abc",
			},
				func(_ context.Context, event httpclient.Event) error {
					assert.Equal(t, testcase.events[index], event)
					index++

					return nil
				},
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
