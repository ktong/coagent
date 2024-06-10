// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package openai

import "github.com/ktong/assistant/internal/schema"

type (
	function struct {
		Name        string         `json:"name"`
		Description string         `json:"description,omitempty"`
		Parameters  *schema.Schema `json:"parameters,omitempty"`
	}
	tool struct {
		Type     string    `json:"type"`
		Function *function `json:"function,omitempty"`
	}
	attachment struct {
		FileID string `json:"file_id,omitempty"`
		Tools  []tool `json:"tools,omitempty"`
	}
	content struct {
		Type      string     `json:"type"`
		Text      string     `json:"text,omitempty"`
		ImageFile *imageFile `json:"image_file,omitempty"`
		ImageUrl  *imageURL  `json:"image_url,omitempty"`
	}
	imageFile struct {
		FileID string `json:"file_id"`
		Detail string `json:"detail,omitempty"`
	}
	imageURL struct {
		URL    string `json:"url"`
		Detail string `json:"detail,omitempty"`
	}
	message struct {
		Role        string       `json:"role"`
		Content     []content    `json:"content"`
		Attachments []attachment `json:"attachments,omitempty"`
	}
)
