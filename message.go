// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package coagent

import (
	"io"

	"github.com/ktong/coagent/internal/embedded"
)

type (
	Message struct {
		Role    string
		Content []Content
		Tools   []Tool
	}
	Content interface {
		embedded.Content
	}

	// Text content that is part of a message.
	Text struct {
		embedded.Content

		Text string
	}

	// Image is a base64-encoded image in the content of a message.
	// It's only supported on Vision-compatible models.
	Image struct {
		embedded.Content

		Image io.Reader
	}
)
