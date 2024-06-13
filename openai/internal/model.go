// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package internal

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ktong/assistant"
	"github.com/ktong/assistant/internal/schema"
)

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
		ImageURL  *imageURL  `json:"image_url,omitempty"`
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

func toTool(t assistant.Tool) tool {
	switch t.(type) {
	case CodeInterpreter:
		return tool{Type: "code_interpreter"}
	case FileSearch:
		return tool{Type: "file_search"}
	default:
		return tool{}
	}
}

func toToolResources(tools []assistant.Tool) map[string]any {
	resources := map[string]any{}
	for _, t := range tools {
		switch tool := t.(type) {
		case CodeInterpreter:
			fileIDs := make([]string, 0, len(tool.Files))
			for _, file := range tool.Files {
				fileIDs = append(fileIDs, file.ID)
			}
			resources["code_interpreter"] = map[string][]string{"file_ids": fileIDs}
		case FileSearch:
			if tool.Store.ID != "" {
				resources["file_search"] = map[string][]string{"vector_store_ids": {tool.Store.ID}}
			} else {
				fileIDs := make([]string, 0, len(tool.Store.Files))
				for _, file := range tool.Store.Files {
					fileIDs = append(fileIDs, file.ID)
				}
				resources["file_search"] = map[string]map[string][]string{"vector_stores": {"file_ids": fileIDs}}
			}
		}
	}

	return resources
}

func toMessage(m assistant.Message) message {
	msg := message{
		Role: string(m.Role),
	}
	for _, c := range m.Content {
		switch cont := c.(type) {
		case assistant.Text:
			msg.Content = append(msg.Content, content{Type: "text", Text: cont.Text})
		case assistant.Image[string]:
			parsedURL, _ := url.Parse(cont.URL)
			switch parsedURL.Scheme {
			case "", "file":
				msg.Content = append(msg.Content,
					content{Type: "image_file", ImageFile: &imageFile{FileID: parsedURL.Path, Detail: string(cont.Detail)}},
				)
			default:
				msg.Content = append(msg.Content,
					content{Type: "image_url", ImageURL: &imageURL{URL: cont.URL, Detail: string(cont.Detail)}})
			}
		case assistant.Image[[]byte]:
			mime := http.DetectContentType(cont.URL)
			switch mime {
			case "image/gif", "image/jpeg", "image/pjpeg":
				maxEncLen := base64.StdEncoding.EncodedLen(len(cont.URL))
				buf := make([]byte, maxEncLen) //nolint:makezero
				base64.StdEncoding.Encode(buf, cont.URL)
				u := fmt.Sprintf("Data:%s;base64,%s", mime, buf)

				msg.Content = append(msg.Content,
					content{Type: "image_url", ImageURL: &imageURL{URL: u, Detail: string(cont.Detail)}},
				)
			}
		}

		appendFiles := func(tol assistant.Tool, files []assistant.File) {
		fileLoop:
			for _, file := range files {
				for _, attachment := range msg.Attachments {
					if attachment.FileID == file.ID {
						attachment.Tools = append(attachment.Tools, toTool(tol))

						continue fileLoop
					}
				}
				msg.Attachments = append(msg.Attachments, attachment{FileID: file.ID, Tools: []tool{toTool(tol)}})
			}
		}
		for _, t := range m.Tools {
			switch tol := t.(type) {
			case CodeInterpreter:
				appendFiles(tol, tol.Files)
			case FileSearch:
				appendFiles(tol, tol.Store.Files)
			}
		}
	}

	return msg
}
