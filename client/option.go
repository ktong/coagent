// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package client

import "net/http"

// WithAPIKey provides the [OpenAI API key].
//
// By default, the key is read from environment variable OPENAI_API_KEY.
//
// [OpenAI API key]: https://platform.openai.com/account/api-keys
func WithAPIKey(apiKey string) Option {
	return func(options *options) {
		options.apiKey = apiKey
	}
}

// WithHTTPClient provides a http.client for OpenAI REST API.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(options *options) {
		options.httpClient = httpClient
	}
}

type (
	// Option configures a Client.
	Option  func(*options)
	options Client
)
