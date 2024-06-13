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
	Function func(ctx context.Context, argument A) (R, error)
}

type functional[A, R any] interface {
	func(context.Context, A) (R, error) | func(context.Context, string, A) (R, error) | Assistant
}

// FunctionFor creates a function tool for either a function or a assistant.
func FunctionFor[A, R any, S functional[A, R]](s S) Function[A, R] {
	switch from := any(s).(type) {
	case Assistant:
		return Function[A, R]{
			Name:        from.Name,
			Description: from.Description,
			Function: func(ctx context.Context, argument A) (R, error) {
				message, err := toMessage(argument)
				if err != nil {
					return *new(R), fmt.Errorf("convert argument to content: %w", err)
				}
				thread := &Thread{ID: fmt.Sprintf("%v", ctx.Value(contextKeyThreadID{})), Messages: []Message{message}}
				message, err = from.Run(ctx, thread)
				if err != nil {
					return *new(R), fmt.Errorf("run assistant: %w", err)
				}

				return fromMessage[R](message)
			},
		}
	case func(context.Context, A) (R, error):
		name := runtime.FuncForPC(reflect.ValueOf(from).Pointer()).Name()
		name = name[strings.LastIndex(name, ".")+1:]

		return Function[A, R]{
			Name:     name,
			Function: from,
		}
	case func(context.Context, string, A) (R, error):
		name := runtime.FuncForPC(reflect.ValueOf(from).Pointer()).Name()
		name = name[strings.LastIndex(name, ".")+1:]

		return Function[A, R]{
			Name: name,
			Function: func(ctx context.Context, argument A) (R, error) {
				return from(ctx, fmt.Sprintf("%v", ctx.Value(contextKeyThreadID{})), argument)
			},
		}
	default:
		return Function[A, R]{} // Should not happen.
	}
}

// Below are workarounds for allowing the generic type to be used in the function call.
// TODO: revise the workaround.

type FunctionSchema struct {
	Name        string
	Description string
	Parameter   *schema.Schema
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

func (f Function[A, R]) Call(ctx context.Context, threadID string, argument string) (Message, error) {
	if threadID != "" {
		ctx = context.WithValue(ctx, contextKeyThreadID{}, threadID)
	}

	var a A
	if err := json.Unmarshal([]byte(argument), &a); err != nil {
		return Message{}, fmt.Errorf("unmarshal function call arguments: %w", err)
	}
	r, err := f.Function(ctx, a)
	if err != nil {
		return Message{}, fmt.Errorf("call function: %w", err)
	}

	return toMessage(r)
}

type contextKeyThreadID struct{}
