// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package assistant

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/invopop/jsonschema"

	"github.com/ktong/assistant/embedded"
)

// Tool is a tool that can be used by an Assistant.
type Tool interface {
	embedded.Tool
}

// BuiltInTool is a OpenAI-hosted Tool like code_interpreter and file_search.
type BuiltInTool interface {
	embedded.BuiltInTool
}

// File is used to upload document.
type File struct {
	ID string
}

// CodeInterpreter allows Assistants to write and run Python code in a sandboxed execution environment.
// This tool can process files with diverse Data and formatting, and generate files with Data and images of graphs.
// Code Interpreter allows your Assistant to run code iteratively to solve challenging code and math problems.
// When your Assistant writes code that fails to run, it can iterate on this code by attempting to
// run different code until the code execution succeeds.
type CodeInterpreter struct {
	embedded.BuiltInTool

	// A list of files made available to the code_interpreter tool.
	// There can be a maximum of 20 files associated with the tool.
	Files []File
}

func (c CodeInterpreter) MarshalJSON() ([]byte, error) {
	return []byte(`{"type":"code_interpreter"}`), nil
}

// Function calling allows you to describe functions to the Assistants API
// and have it intelligently return the functions that need to be called along with their arguments.
type Function[A, R any] struct {
	embedded.Tool

	// The name of the function to be called.
	// Must be a-z, A-Z, 0-9, or contain underscores and dashes, with a maximum length of 64.
	Name string
	// A description of what the function does, used by the model to choose when and how to call the function.
	Description string
	// The real function attached to the tool.
	Function func(A) (R, error)
}

func (f Function[A, R]) name() string {
	return f.Name
}

func (f Function[A, R]) call(arguments []byte) string {
	var a A
	if err := json.Unmarshal(arguments, &a); err != nil {
		return fmt.Sprintf(`{"error": "unmarshal arguments: %s"}`, err)
	}
	r, err := f.Function(a)
	if err != nil {
		return fmt.Sprintf(`{"error": "call function: %s"}`, err)
	}
	b, err := json.Marshal(r)
	if err != nil {
		return fmt.Sprintf(`{"error": "marshal result: %s"}`, err)
	}

	return string(b)
}

func (f Function[A, R]) MarshalJSON() ([]byte, error) {
	locType := reflect.TypeFor[A]()
	reflector := jsonschema.Reflector{
		Anonymous:      true,
		DoNotReference: true,
	}
	schema := reflector.ReflectFromType(locType)
	parameters, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("marshal json schema: %w", err)
	}
	s := fmt.Sprintf(`{"type":"function","function":{"name":"%s","description":"%s","parameters":%s}}`,
		f.Name, f.Description, parameters)

	return []byte(s), nil
}
