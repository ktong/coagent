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

type functional[A, R any] interface {
	func(context.Context, A) (R, error) | func(context.Context, *Thread, A) (R, error)
}

// Function calling allows you to describe functions to the Assistants API
// and have it intelligently return the functions that need to be called along with their arguments.
type Function[A, R any, FN functional[A, R]] struct {
	embedded.Tool

	// The name of the function to be called.
	// Must be a-z, A-Z, 0-9, or contain underscores and dashes, with a maximum length of 64.
	Name string
	// A description of what the function does, used by the model to choose when and how to call the function.
	Description string
	// The real function attached to the tool.
	Function FN
}

// FunctionFor creates a function tool for either a function or a assistant.
func FunctionFor[A, R any, S functional[A, R] | Assistant](s S,
) Function[A, R, func(context.Context, *Thread, A) (R, error)] {
	switch from := any(s).(type) {
	case Assistant:
		return Function[A, R, func(context.Context, *Thread, A) (R, error)]{
			Name:        from.Name,
			Description: from.Description,
			Function: func(ctx context.Context, thread *Thread, argument A) (R, error) {
				message, err := toMessage(argument)
				if err != nil {
					return *new(R), fmt.Errorf("convert argument to content: %w", err)
				}
				thread.Messages = append(thread.Messages, message)

				err = from.Run(ctx, thread, message)
				if err != nil {
					return *new(R), fmt.Errorf("run assistant: %w", err)
				}

				return fromMessage[R](thread.Messages[len(thread.Messages)-1])
			},
		}
	case func(context.Context, A) (R, error):
		name := runtime.FuncForPC(reflect.ValueOf(from).Pointer()).Name()
		name = name[strings.LastIndex(name, ".")+1:]

		return Function[A, R, func(context.Context, *Thread, A) (R, error)]{
			Name: name,
			Function: func(ctx context.Context, _ *Thread, argument A) (R, error) {
				return from(ctx, argument)
			},
		}
	case func(context.Context, *Thread, A) (R, error):
		name := runtime.FuncForPC(reflect.ValueOf(from).Pointer()).Name()
		name = name[strings.LastIndex(name, ".")+1:]

		return Function[A, R, func(context.Context, *Thread, A) (R, error)]{
			Name:     name,
			Function: from,
		}
	default:
		return Function[A, R, func(context.Context, *Thread, A) (R, error)]{} // Should not happen.
	}
}

// Below are workarounds for allowing the generic type to be used in the function call.
// TODO: revise the workaround.

type FunctionSchema struct {
	Name        string
	Description string
	Parameter   *schema.Schema
}

func (f Function[A, R, FN]) Schema() (FunctionSchema, error) {
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

func (f Function[A, R, FN]) ID() string {
	return f.Name
}

func (f Function[A, R, FN]) Call(ctx context.Context, thread *Thread, a string) (Message, error) {
	var argument A
	if err := json.Unmarshal([]byte(a), &argument); err != nil {
		return Message{}, fmt.Errorf("unmarshal function call arguments: %w", err)
	}

	var (
		result R
		err    error
	)
	switch fn := any(f.Function).(type) {
	case func(context.Context, A) (R, error):
		result, err = fn(ctx, argument)
	case func(context.Context, *Thread, A) (R, error):
		result, err = fn(ctx, thread, argument)
	}
	if err != nil {
		return Message{}, fmt.Errorf("call function: %w", err)
	}

	return toMessage(result)
}
