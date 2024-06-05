// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package assistant_test

import (
	"testing"

	"github.com/ktong/assistant"
	"github.com/ktong/assistant/internal/assert"
)

func TestCodeInterpreter_MarshalJSON(t *testing.T) {
	json, err := assistant.CodeInterpreter{}.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, `{"type":"code_interpreter"}`, string(json))
}

func TestFunction_MarshalJSON(t *testing.T) {
	type location struct {
		City  string `json:"city"            jsonschema:"description=The city name,example=San Francisco"`
		State string `json:"state,omitempty" jsonschema:"description=The state abbreviation,example=CA"`
	}
	json, err := assistant.Function[location, float32]{
		Name:        "RainProbability",
		Description: "Get the probability of rain for a specific location",
		Function: func(location) (float32, error) {
			return 0.2, nil
		},
	}.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t,
		`{"type":"function","function":{"name":"RainProbability","description":"Get the probability of rain for a specific location",`+
			`"parameters":{"$schema":"https://json-schema.org/draft/2020-12/schema","properties":{`+
			`"city":{"type":"string","description":"The city name","examples":["San Francisco"]},`+
			`"state":{"type":"string","description":"The state abbreviation","examples":["CA"]}`+
			`},"additionalProperties":false,"type":"object","required":["city"]}}}`,
		string(json),
	)
}
