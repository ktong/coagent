// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package httpclient

import (
	"net"
	"net/http"
	"time"
)

func WithHTTPClient(client *http.Client) Option {
	return func(o *options) {
		o.client = client
	}
}

func WithBaseURL(baseURL string) Option {
	return func(o *options) {
		o.baseURL = baseURL
	}
}

func WithHeader(key, value string) Option {
	return func(o *options) {
		o.headers[key] = value
	}
}

type (
	// Option configures the httpclient request.
	Option  func(*options)
	options struct {
		client  *http.Client
		baseURL string
		headers map[string]string
	}
)

func apply(opts []Option) options {
	option := options{
		client:  defaultClient,
		headers: map[string]string{},
	}
	for _, opt := range opts {
		opt(&option)
	}

	return option
}

const timeout = 5 * time.Second

var defaultClient = &http.Client{ //nolint:gochecknoglobals
	Transport: &http.Transport{
		DialContext:           (&net.Dialer{Timeout: timeout}).DialContext,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100, //nolint:mnd
		MaxIdleConnsPerHost:   100, //nolint:mnd
	},
}
