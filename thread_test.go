// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

//go:build integration

package assistant_test

import (
	"context"
	"testing"

	"github.com/ktong/assistant"
	"github.com/ktong/assistant/internal/assert"
)

func TestRun(t *testing.T) {
	type (
		location struct {
			City  string `json:"city"            jsonschema:"description=The city name,example=San Francisco"`
			State string `json:"state,omitempty" jsonschema:"description=The state abbreviation,example=CA"`
		}
		temperature struct {
			Temperature float32
			Unit        string
		}
	)

	message, err := assistant.Run[string, string](context.Background(),
		"What's the weather in San Francisco today and the likelihood it'll rain?",
		assistant.WithInstructions("You are a weather bot. Use the provided functions to answer questions."),
		assistant.WithTool(
			assistant.Function[location, temperature]{
				Name:        "CurrentTemperature",
				Description: "Get the current temperature for a specific location",
				Function: func(location) (temperature, error) {
					return temperature{Temperature: 72, Unit: "Fahrenheit"}, nil
				},
			},
			assistant.Function[location, float32]{
				Name:        "RainProbability",
				Description: "Get the probability of rain for a specific location",
				Function: func(location) (float32, error) {
					return 0.2, nil
				},
			},
		),
	)

	assert.NoError(t, err)
	assert.Equal(t, "The current temperature in San Francisco, CA is 72°F. There is a 20% chance of rain today.", message)
}
