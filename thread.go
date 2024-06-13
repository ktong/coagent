// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package assistant

// Thread is a conversation session between an Assistant and a user.
// Threads store Messages and automatically handle truncation to fit content into a model’s context.
// Due to [Thread Locks], the same thread could not be ran by multiple goroutines simultaneously.
//
// If ID is empty or the thread with the given ID does not exist,
// a new thread will be created on the server with the given information.
// Otherwise, the thread with the given ID will be used and other fields are ignored.
//
// It's suggested that save the thread ID in the users' session and reuse it for the following runs
// in the short duration, or save it into the database for the long duration.
//
// [Thread Locks]: https://platform.openai.com/docs/assistants/how-it-works/thread-locks
type Thread struct {
	ID       string
	Messages []Message
	Tools    []Tool
}

// Following are the helpers to append messages to the thread.

func (t *Thread) AppendText(texts ...string) {
	message := Message{Role: RoleUser}
	for _, text := range texts {
		message.Content = append(message.Content, Text{Text: text})
	}
	t.Messages = append(t.Messages, message)
}

func (t *Thread) AppendImage(detail Detail, images ...string) {
	message := Message{Role: RoleUser}
	for _, image := range images {
		message.Content = append(message.Content, Image[string]{URL: image, Detail: detail})
	}
	t.Messages = append(t.Messages, message)
}

func (t *Thread) AppendImageContent(detail Detail, images ...[]byte) {
	message := Message{Role: RoleUser}
	for _, image := range images {
		message.Content = append(message.Content, Image[[]byte]{URL: image, Detail: detail})
	}
	t.Messages = append(t.Messages, message)
}
