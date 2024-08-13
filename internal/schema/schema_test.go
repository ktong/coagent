// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package schema_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/ktong/assistant/internal/assert"
	"github.com/ktong/assistant/internal/schema"
)

type RecursiveChildKey struct {
	Key  string             `json:"key"`
	Self *RecursiveChildKey `json:"self,omitempty"`
}

type RecursiveChild struct {
	RecursiveChildLoop
}

type RecursiveChildLoop struct {
	*RecursiveChild
	Slice   []*RecursiveChildLoop  `json:"slice"`
	Array   [1]*RecursiveChildLoop `json:"array"`
	ByValue RecursiveChildKey      `json:"byValue"`
	ByRef   *RecursiveChildKey     `json:"byRef"`
}

type EmbeddedChild struct {
	// This one should be ignored as it is overridden by `Embedded`.
	Value string `json:"value" description:"old doc"`
}

type Embedded struct {
	EmbeddedChild
	Value string `json:"value" description:"new doc"`
}

func TestSchema(t *testing.T) { //nolint:maintidx
	testcases := []struct {
		name     string
		input    func() (schema.Schema, error)
		expected string
		error    string
	}{
		{
			name:     "bool",
			input:    schema.For[bool],
			expected: `{"type": "boolean"}`,
		},
		{
			name:     "bool-pointer",
			input:    schema.For[*bool],
			expected: `{"type": "boolean"}`,
		},
		{
			name:     "int",
			input:    schema.For[int],
			expected: `{"type": "integer"}`,
		},
		{
			name:     "int32",
			input:    schema.For[int32],
			expected: `{"type": "integer"}`,
		},
		{
			name:     "int64",
			input:    schema.For[int64],
			expected: `{"type": "integer"}`,
		},
		{
			name:     "uint",
			input:    schema.For[uint],
			expected: `{"type": "integer"}`,
		},
		{
			name:     "uint32",
			input:    schema.For[uint32],
			expected: `{"type": "integer"}`,
		},
		{
			name:     "uint64",
			input:    schema.For[uint64],
			expected: `{"type": "integer"}`,
		},
		{
			name:     "float64",
			input:    schema.For[float64],
			expected: `{"type": "number"}`,
		},
		{
			name:     "float32",
			input:    schema.For[float32],
			expected: `{"type": "number"}`,
		},
		{
			name:     "string",
			input:    schema.For[string],
			expected: `{"type": "string"}`,
		},
		{
			name:     "bytes",
			input:    schema.For[[]byte],
			expected: `{"type": "string", "contentEncoding": "base64"}`,
		},
		{
			name:     "array",
			input:    schema.For[[2]int],
			expected: `{"items": {"type": "integer"}, "type": "array"}`,
		},
		{
			name:     "slice",
			input:    schema.For[[]int],
			expected: `{"items": {"type": "integer"}, "type": "array"}`,
		},
		{
			name: "field-int",
			input: schema.For[struct {
				Value int `json:"value"`
			}],
			expected: `{
				"properties": {
					"value": {
						"type": "integer"
					}
				},
				"additionalProperties": false,
				"type": "object",
				"required": ["value"]
			}`,
		},
		{
			name: "field-string",
			input: schema.For[struct {
				Value string `json:"value" encoding:"bar"`
			}],
			expected: `{
				"properties": {
					"value": {
						"type": "string",
						"contentEncoding": "bar"
					}
				},
				"additionalProperties": false,
				"type": "object",
				"required": ["value"]
			}`,
		},
		{
			name: "field-array",
			input: schema.For[struct {
				Value []int `json:"value"`
			}],
			expected: `{
				"properties": {
					"value": {
						"items": {"type": "integer"},
						"type": "array"
					}
				},
				"additionalProperties": false,
				"type": "object",
				"required": ["value"]
			}`,
		},
		{
			name: "field-enum",
			input: schema.For[struct {
				Value string `json:"value" enum:"one,two"`
			}],
			expected: `{
				"properties": {
					"value": {
						"type": "string",
						"enum": ["one", "two"]
					}
				},
				"additionalProperties": false,
				"type": "object",
				"required": ["value"]
			}`,
		},
		{
			name: "field-array-enum",
			input: schema.For[struct {
				Value []int `json:"value" enum:"1,2,3,5,8,11"`
			}],
			expected: `{
				"properties": {
					"value": {
						"items": {
							"type": "integer",
							"enum": [1, 2, 3, 5, 8, 11]
						},
						"type": "array"
					}
				},
				"additionalProperties": false,
				"type": "object",
				"required": ["value"]
			}`,
		},
		{
			name: "field-optional-without-name",
			input: schema.For[struct {
				Value string `json:",omitempty"`
			}],
			expected: `{
				"properties": {
					"Value": {
						"type": ["string","null"]
					}
				},
				"additionalProperties": false,
				"type": "object",
				"required":["Value"]
			}`,
		},
		{
			name: "field-any",
			input: schema.For[struct {
				Value string `json:"value" description:"Some value"`
			}],
			expected: `{
				"properties": {
					"value": {
						"type": "string",
						"description": "Some value"
					}
				},
				"additionalProperties": false,
				"type": "object",
				"required": ["value"]
			}`,
		},
		{
			name:  "field-ref",
			input: schema.For[RecursiveChild],
			expected: `{
				"$defs":{
					"schema_test.RecursiveChildKey":{
						"properties":{
							"key":{
								"type":"string"
							},
							"self":{
								"$ref":"#/$defs/schema_test.RecursiveChildKey"
							}
						},
						"additionalProperties":false,
						"type":"object",
						"required":["key","self"]
					},
					"schema_test.RecursiveChildLoop":{
						"properties":{
							"array":{
								"items":{
									"$ref":"#/$defs/schema_test.RecursiveChildLoop"
								},
								"type":"array"
							},
							"byRef":{
								"$ref":"#/$defs/schema_test.RecursiveChildKey"
							},
							"byValue":{
								"$ref":"#/$defs/schema_test.RecursiveChildKey"
							},
							"slice":{
								"items":{
									"$ref":"#/$defs/schema_test.RecursiveChildLoop"
								},
								"type":"array"
							}
						},
						"additionalProperties":false,
						"type":"object",
						"required":["slice","array","byValue","byRef"]
					}
				},
				"properties":{
					"array":{
						"items":{
							"$ref":"#/$defs/schema_test.RecursiveChildLoop"
						},
						"type":"array"
					},
					"byRef":{
						"$ref":"#/$defs/schema_test.RecursiveChildKey"
					},
					"byValue":{
						"$ref":"#/$defs/schema_test.RecursiveChildKey"
					},
					"slice":{
						"items":{
							"$ref":"#/$defs/schema_test.RecursiveChildLoop"
						},
						"type":"array"
					}
				},
				"additionalProperties":false,
				"type":"object",
				"required":["slice","array","byValue","byRef"]
			}`,
		},
		{
			name:  "field-self",
			input: schema.For[RecursiveChildKey],
			expected: `{
				"properties":{
					"key":{
						"type":"string"
					},
					"self":{"$ref":"#"}
				},
				"additionalProperties":false,
				"type":"object",
				"required":["key","self"]
			}`,
		},
		{
			name: "field-embed",
			input: schema.For[struct {
				// Because this is embedded, the fields should be merged into
				// the parent object.
				*Embedded
				Value2 string `json:"value2"`
			}],
			expected: `{
				"properties": {
					"value": {
						"type": "string",
						"description": "new doc"
					},
					"value2": {
						"type": "string"
					}
				},
				"additionalProperties": false,
				"type": "object",
				"required": ["value2", "value"]
			}`,
		},
		{
			name: "field-embed-override",
			input: schema.For[struct {
				Embedded
				Value string `json:"override" description:"override"`
			}],
			expected: `{
				"properties": {
					"override": {
						"type": "string",
						"description": "override"
					}
				},
				"additionalProperties": false,
				"type": "object",
				"required": ["override"]
			}`,
		},
		{
			name:  "error-unsupported",
			input: schema.For[map[string]string],
			error: "unsupported type: map[string]string",
		},
		{
			name: "error-bool",
			input: schema.For[struct {
				Value string `json:"value" required:"bad"`
			}],
			error: "invalid bool tag 'required' for field 'Value': bad",
		},
		{
			name: "error-json",
			input: schema.For[struct {
				Value int `json:"value" enum:"bad"`
			}],
			error: `invalid integer tag value 'bad' for field 'Value': invalid character 'b' looking for beginning of value`,
		},
		{
			name: "error-json-bool",
			input: schema.For[struct {
				Value bool `json:"value" enum:"123"`
			}],
			error: `invalid boolean tag value '123' for field 'Value'`,
		},
		{
			name: "error-json-int",
			input: schema.For[struct {
				Value int `json:"value" enum:"true"`
			}],
			error: `invalid integer tag value 'true' for field 'Value'`,
		},
		{
			name: "error-json-int2",
			input: schema.For[struct {
				Value int `json:"value" enum:"1.23"`
			}],
			error: `invalid integer tag value '1.23' for field 'Value'`,
		},
		{
			name: "error-json-object",
			input: schema.For[struct {
				Value struct {
					Foo string `json:"foo"`
				} `json:"value" enum:"true"`
			}],
			error: `invalid object tag value 'true' for field 'Value'`,
		},
		{
			name: "error-json-object-field",
			input: schema.For[struct {
				Value struct {
					Foo string `json:"foo"`
				} `json:"value" enum:"{\"foo\": true}"`
			}],
			error: `invalid string tag value 'true' for field 'Value.foo'`,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			actual, err := testcase.input()
			if testcase.error != "" {
				assert.EqualError(t, err, testcase.error)

				return
			}

			assert.NoError(t, err)
			b, _ := json.Marshal(actual)
			var e bytes.Buffer
			_ = json.Compact(&e, []byte(testcase.expected))
			assert.Equal(t, e.String(), string(b))
		})
	}
}
