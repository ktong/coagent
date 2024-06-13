// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

//nolint:ireturn
package openai

import (
	"github.com/ktong/assistant/openai/httpclient"
	"github.com/ktong/assistant/openai/internal"
)

type (
	Option          = httpclient.Option
	CodeInterpreter = internal.CodeInterpreter
	FileSearch      = internal.FileSearch
)

//nolint:gochecknoglobals
var (
	WithHTTPClient = httpclient.WithHTTPClient
	WithBaseURL    = httpclient.WithBaseURL
	WithHeader     = httpclient.WithHeader
)

//nolint:gochecknoglobals
var (
	WithModel               = internal.WithModel
	WithInstructions        = internal.WithInstructions
	WithTemperature         = internal.WithTemperature
	WithMaxPromptTokens     = internal.WithMaxPromptTokens
	WithMaxCompletionTokens = internal.WithMaxCompletionTokens
	WithParallelToolCallS   = internal.WithParallelToolCallS
)
