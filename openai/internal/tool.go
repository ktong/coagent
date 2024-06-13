// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package internal

import (
	"github.com/ktong/assistant"
	"github.com/ktong/assistant/internal/embedded"
)

// CodeInterpreter allows Assistants to write and run Python code in a sandboxed execution environment.
// This tool can process files with diverse Data and formatting, and generate files with Data and images of graphs.
// Code Interpreter allows your Assistant to run code iteratively to solve challenging code and math problems.
// When your Assistant writes code that fails to run, it can iterate on this code by attempting to
// run different code until the code execution succeeds.
type CodeInterpreter struct {
	embedded.BuiltInTool

	// A list of files made available to the code_interpreter tool.
	// There can be a maximum of 20 files associated with the tool.
	Files []assistant.File
}

// FileSearch augments the Assistant with knowledge from outside its model,
// such as proprietary product information or documents provided by your users.
// OpenAI automatically parses and chunks your documents, creates and stores the embeddings,
// and use both vector and keyword search to retrieve relevant content to answer user queries.
type FileSearch struct {
	embedded.BuiltInTool

	// The vector store attached to this assistant.
	Store VectorStore
}

type VectorStore struct {
	ID    string
	Name  string
	Files []assistant.File
}
