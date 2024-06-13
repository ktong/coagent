// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

//go:build integration

package assistant_test

import (
	"context"
	"testing"

	"github.com/ktong/assistant"
	"github.com/ktong/assistant/internal/assert"
	"github.com/ktong/assistant/openai"
)

func TestAssistant_Run(t *testing.T) {
	assistant.SetDefaultExecutor(openai.NewExecutor())
	ctx := context.Background()

	asst := assistant.Assistant{
		Name:         "Weather Bot",
		Instructions: "You are a weather bot.",
		Tools: []assistant.Tool{
			assistant.FunctionFor[location, temperature](getCurrentTemperature),
			assistant.FunctionFor[location, float32](getRainProbability),
		},
	}
	defer func() {
		_ = asst.Shutdown(ctx)
	}()

	var thread assistant.Thread
	thread.AppendText("What's the weather in San Francisco today and the likelihood it'll rain?")
	message, err := asst.Run(ctx, &thread)
	assert.NoError(t, err)
	assert.True(t, thread.ID != "")
	assert.Equal(t,
		"The current temperature in San Francisco, CA is 72°F. The likelihood of rain today is 20%.",
		message.Content[0].(assistant.Text).Text,
	)

	thread.AppendText("What's the weather in New York City?")
	message, err = asst.Run(ctx, &thread)
	assert.NoError(t, err)
	assert.Equal(t,
		"The current temperature in New York City, NY is 72°F. The likelihood of rain today is 20%.",
		message.Content[0].(assistant.Text).Text,
	)
}

type (
	location struct {
		City  string `json:"city"            description:"The city name"          example:"San Francisco"`
		State string `json:"state,omitempty" description:"The state abbreviation" example:"CA"`
	}
	temperature struct {
		Temperature float32
		Unit        string
	}
)

func getCurrentTemperature(context.Context, location) (temperature, error) {
	return temperature{Temperature: 72, Unit: "Fahrenheit"}, nil
}

func getRainProbability(context.Context, location) (float32, error) {
	return 0.2, nil
}
