// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package assistant_test

import (
	"encoding/base64"
	"testing"

	"github.com/ktong/assistant"
	"github.com/ktong/assistant/internal/assert"
)

func TestText_MarshalJSON(t *testing.T) {
	json, err := assistant.Text{Text: "What's the weather in San Francisco today and the likelihood it'll rain?"}.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, `{"type":"text","text":"What's the weather in San Francisco today and the likelihood it'll rain?"}`, string(json))
}

func TestImageFile_MarshalJSON(t *testing.T) {
	json, err := assistant.Image[string]{URL: "image.jpg", Detail: assistant.DetailAuto}.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, `{"type":"image_file","image_file":{"file_id":"image.jpg","detail":"auto"}}`, string(json))
}

func TestImageURL_MarshalJSON(t *testing.T) {
	json, err := assistant.Image[string]{URL: "https://sample.com/a.jpg", Detail: assistant.DetailAuto}.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, `{"type":"image_url","image_url":{"url":"https://sample.com/a.jpg","detail":"auto"}}`, string(json))
}

func TestImageContent_MarshalJSON(t *testing.T) {
	image, err := base64.StdEncoding.DecodeString("R0lGODlhAQABAIAAAP///wAAACH5BAEAAAAALAAAAAABAAEAAAICRAEAOw==")
	assert.NoError(t, err)

	json, err := assistant.Image[[]byte]{URL: image, Detail: assistant.DetailAuto}.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, `{"type":"image_url","image_url":{"url":"Data:image/gif;base64,R0lGODlhAQABAIAAAP///wAAACH5BAEAAAAALAAAAAABAAEAAAICRAEAOw==","detail":"auto"}}`, string(json))
}

func TestMessage_MarshalJSON(t *testing.T) {
	json, err := assistant.Message{
		Role:    assistant.RoleUser,
		Content: []assistant.Content{assistant.Text{Text: "What's the weather in San Francisco today and the likelihood it'll rain?"}},
		Tools:   []assistant.BuiltInTool{assistant.CodeInterpreter{Files: []assistant.File{{ID: "tool1"}, {ID: "tool3"}}}},
	}.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, `{"attachments":[{"file_id":"tool1","tools":[{"type":"code_interpreter"}]},{"file_id":"tool3","tools":[{"type":"code_interpreter"}]}],`+
		`"content":[{"type":"text","text":"What's the weather in San Francisco today and the likelihood it'll rain?"}],"role":"user"}`, string(json))
}
