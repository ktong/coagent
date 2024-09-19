// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/ktong/assistant/internal/embedded"
	"github.com/ktong/assistant/internal/schema"
)

type Tool interface {
	embedded.Tool
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
	Function func(context.Context, *Thread, A) (R, error)
}

// FunctionFor creates a function tool for either a function or an assistant.
func FunctionFor[A, R any, S func(context.Context, A) (R, error) | func(A) (R, error) |
	func(context.Context, *Thread, A) (R, error) | func(*Thread, A) (R, error)](s S) Function[A, R] {
	name := runtime.FuncForPC(reflect.ValueOf(s).Pointer()).Name()
	name = name[strings.LastIndex(name, ".")+1:]

	var function func(context.Context, *Thread, A) (R, error)
	switch from := any(s).(type) {
	case func(context.Context, *Thread, A) (R, error):
		function = from
	case func(*Thread, A) (R, error):
		function = func(_ context.Context, thread *Thread, argument A) (R, error) {
			return from(thread, argument)
		}
	case func(context.Context, A) (R, error):
		function = func(ctx context.Context, _ *Thread, argument A) (R, error) {
			return from(ctx, argument)
		}
	case func(A) (R, error):
		function = func(_ context.Context, _ *Thread, argument A) (R, error) {
			return from(argument)
		}
	default:
	}

	return Function[A, R]{Name: name, Function: function}
}

// Below are workarounds for allowing the generic type to be used in the function call.
// TODO: revise the workaround.

type FunctionSchema struct {
	Name        string
	Description string
	Parameter   schema.Schema
}

func (f Function[A, R]) Schema() (FunctionSchema, error) {
	parameterSchema, err := schema.For[A]()
	if err != nil {
		return FunctionSchema{}, fmt.Errorf("generate function schema: %w", err)
	}

	return FunctionSchema{
		Name:        f.Name,
		Description: f.Description,
		Parameter:   parameterSchema,
	}, nil
}

func (f Function[A, R]) ID() string {
	return f.Name
}

func (f Function[A, R]) Call(ctx context.Context, thread *Thread, a string) (Message, error) {
	var argument A
	if err := json.Unmarshal([]byte(a), &argument); err != nil {
		return Message{}, fmt.Errorf("unmarshal function call arguments: %w", err)
	}

	var (
		result R
		err    error
	)
	result, err = f.Function(ctx, thread, argument)
	if err != nil {
		return Message{}, fmt.Errorf("call function: %w", err)
	}

	return toMessage(result)
}
