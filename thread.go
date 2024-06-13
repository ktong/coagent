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
	Metadata map[string]any
}
