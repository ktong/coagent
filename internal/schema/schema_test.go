// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package schema_test

import (
	"bytes"
	"encoding/json"
	"math/bits"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"testing"
	"time"

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
	Slice   []*RecursiveChildLoop                    `json:"slice"`
	Array   [1]*RecursiveChildLoop                   `json:"array"`
	Map     map[RecursiveChildKey]RecursiveChildLoop `json:"map"`
	ByValue RecursiveChildKey                        `json:"byValue"`
	ByRef   *RecursiveChildKey                       `json:"byRef"`
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
	bitSize := strconv.Itoa(bits.UintSize)

	testcases := []struct {
		name     string
		input    func() (*schema.Schema, error)
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
			expected: `{"type": "integer", "format": "int` + bitSize + `"}`,
		},
		{
			name:     "int32",
			input:    schema.For[int32],
			expected: `{"type": "integer", "format": "int32"}`,
		},
		{
			name:     "int64",
			input:    schema.For[int64],
			expected: `{"type": "integer", "format": "int64"}`,
		},
		{
			name:     "uint",
			input:    schema.For[uint],
			expected: `{"type": "integer", "minimum": 0, "format": "int` + bitSize + `"}`,
		},
		{
			name:     "uint32",
			input:    schema.For[uint32],
			expected: `{"type": "integer", "minimum": 0, "format": "int32"}`,
		},
		{
			name:     "uint64",
			input:    schema.For[uint64],
			expected: `{"type": "integer", "minimum": 0, "format": "int64"}`,
		},
		{
			name:     "float64",
			input:    schema.For[float64],
			expected: `{"type": "number", "format": "double"}`,
		},
		{
			name:     "float32",
			input:    schema.For[float32],
			expected: `{"type": "number", "format": "float"}`,
		},
		{
			name:     "string",
			input:    schema.For[string],
			expected: `{"type": "string"}`,
		},
		{
			name:     "time",
			input:    schema.For[time.Time],
			expected: `{"type": "string", "format": "date-time"}`,
		},
		{
			name:     "time-pointer",
			input:    schema.For[*time.Time],
			expected: `{"type": "string", "format": "date-time"}`,
		},
		{
			name:     "url",
			input:    schema.For[url.URL],
			expected: `{"type": "string", "format": "uri"}`,
		},
		{
			name:     "ip",
			input:    schema.For[net.IP],
			expected: `{"type": "string", "format": "ipv4"}`,
		},
		{
			name:     "ipAddr",
			input:    schema.For[netip.Addr],
			expected: `{"type": "string", "format": "ipv4"}`,
		},
		{
			name:     "json.RawMessage",
			input:    schema.For[*json.RawMessage],
			expected: `{}`,
		},
		{
			name:     "bytes",
			input:    schema.For[[]byte],
			expected: `{"type": "string", "contentEncoding": "base64"}`,
		},
		{
			name:     "array",
			input:    schema.For[[2]int],
			expected: `{"items": {"type": "integer", "format": "int64"}, "type": "array", "maxItems": 2, "minItems": 2}`,
		},
		{
			name:     "slice",
			input:    schema.For[[]int],
			expected: `{"items": {"type": "integer", "format": "int64"}, "type": "array"}`,
		},
		{
			name:     "map",
			input:    schema.For[map[string]string],
			expected: `{"additionalProperties": {"type": "string"}, "type": "object"}`,
		},
		{
			name: "additionalProps",
			input: schema.For[struct {
				_     struct{} `json:"-" additionalProperties:"true"`
				Value string   `json:"value"`
			}],
			expected: `{
				"properties": {
					"value": {
						"type": "string"
					}
				},
				"additionalProperties": true,
				"type": "object",
				"required": ["value"]
			}`,
		},
		{
			name: "field-int",
			input: schema.For[struct {
				Value int `json:"value" minimum:"1" exclusiveMinimum:"0" maximum:"10" exclusiveMaximum:"11" multipleOf:"2"`
			}],
			expected: `{
				"properties": {
					"value": {
						"type": "integer",
						"multipleOf": 2,
						"maximum": 10,
						"exclusiveMaximum": 11,
						"minimum": 1,
						"exclusiveMinimum": 0,
						"format": "int64"
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
				Value string `json:"value" minLength:"1" maxLength:"10" pattern:"^foo$" format:"foo" encoding:"bar"`
			}],
			expected: `{
				"properties": {
					"value": {
						"type": "string",
						"maxLength": 10,
						"minLength": 1,
						"pattern": "^foo$",
						"format": "foo",
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
				Value []int `json:"value" minItems:"1" maxItems:"10" uniqueItems:"true"`
			}],
			expected: `{
				"properties": {
					"value": {
						"items": {"type": "integer", "format": "int64"},
						"type": "array",
						"maxItems": 10,
						"minItems": 1,
						"uniqueItems": true
					}
				},
				"additionalProperties": false,
				"type": "object",
				"required": ["value"]
			}`,
		},
		{
			name: "field-map",
			input: schema.For[struct {
				Value map[string]string `json:"value" minProperties:"2" maxProperties:"5"`
			}],
			expected: `{
				"properties": {
					"value": {
						"additionalProperties": {
							"type": "string"
						},
						"type": "object",
						"maxProperties": 5,
						"minProperties": 2
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
							"enum": [1, 2, 3, 5, 8, 11],
							"format": "int64"
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
			name: "field-example-string",
			input: schema.For[struct {
				Value string `json:"value" example:"foo"`
			}],
			expected: `{
				"properties": {
					"value": {
						"type": "string",
						"examples": ["foo"]
					}
				},
				"additionalProperties": false,
				"type": "object",
				"required": ["value"]
			}`,
		},
		{
			name: "field-example-string-pointer",
			input: schema.For[struct {
				Value *string `json:"value,omitempty" example:"foo"`
			}],
			expected: `{
				"properties": {
					"value": {
						"type": "string",
						"examples": ["foo"]
					}
				},
				"additionalProperties": false,
				"type": "object"
			}`,
		},
		{
			name: "field-example-array-string",
			input: schema.For[struct {
				Value []string `json:"value" example:"foo,bar"`
			}],
			expected: `{
				"properties": {
					"value": {
						"items": {
							"type": "string"
						},
						"type": "array",
						"examples": [["foo", "bar"]]
					}
				},
				"additionalProperties": false,
				"type": "object",
				"required": ["value"]
			}`,
		},
		{
			name: "field-example-array-int",
			input: schema.For[struct {
				Value []int `json:"value" example:"[1,2]"`
			}],
			expected: `{
				"properties": {
					"value": {
						"items": {
							"type": "integer",
							"format": "int64"
						},
						"type": "array",
						"examples": [[1, 2]]
					}
				},
				"additionalProperties": false,
				"type": "object",
				"required": ["value"]
			}`,
		},
		{
			name: "field-example-duration",
			input: schema.For[struct {
				Value time.Duration `json:"value" example:"5000"`
			}],
			expected: `{
				"properties": {
					"value": {
						"type": "integer",
						"format": "int64",
						"examples": [5000]
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
						"type": "string"
					}
				},
				"additionalProperties": false,
				"type": "object"
			}`,
		},
		{
			name: "field-any",
			input: schema.For[struct {
				Value any `json:"value" description:"Some value"`
			}],
			expected: `{
				"properties": {
					"value": {
						"description": "Some value"
					}
				},
				"additionalProperties": false,
				"type": "object",
				"required": ["value"]
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
			name: "field-pointer-example",
			input: schema.For[struct {
				Int *int64  `json:"int" example:"123"`
				Str *string `json:"str" example:"foo"`
			}],
			expected: `{
				"properties": {
					"int": {
						"type": "integer",
						"format": "int64",
						"examples": [123]
					},
					"str": {
						"type": "string",
						"examples": ["foo"]
					}
				},
				"additionalProperties": false,
				"type": "object",
				"required": ["int", "str"]
			}`,
		},
		{
			name: "error-bool",
			input: schema.For[struct {
				Value string `json:"value" required:"bad"`
			}],
			error: "invalid bool tag 'required' for field 'Value': bad",
		},
		{
			name: "error-int",
			input: schema.For[struct {
				Value string `json:"value" minLength:"bad"`
			}],
			error: "invalid int tag 'minLength' for field 'Value': bad (strconv.Atoi: parsing \"bad\": invalid syntax)",
		},
		{
			name: "error-float",
			input: schema.For[struct {
				Value int `json:"value" minimum:"bad"`
			}],
			error: "invalid float tag 'minimum' for field 'Value': bad (strconv.ParseFloat: parsing \"bad\": invalid syntax)",
		},
		{
			name: "error-json",
			input: schema.For[struct {
				Value int `json:"value" example:"bad"`
			}],
			error: `invalid integer tag value 'bad' for field 'Value': invalid character 'b' looking for beginning of value`,
		},
		{
			name: "error-json-bool",
			input: schema.For[struct {
				Value bool `json:"value" example:"123"`
			}],
			error: `invalid boolean tag value '123' for field 'Value'`,
		},
		{
			name: "error-json-int",
			input: schema.For[struct {
				Value int `json:"value" example:"true"`
			}],
			error: `invalid integer tag value 'true' for field 'Value'`,
		},
		{
			name: "error-json-int2",
			input: schema.For[struct {
				Value int `json:"value" example:"1.23"`
			}],
			error: `invalid integer tag value '1.23' for field 'Value'`,
		},
		{
			name: "error-json-array",
			input: schema.For[struct {
				Value []int `json:"value" example:"true"`
			}],
			error: `invalid array tag value 'true' for field 'Value'`,
		},
		{
			name: "error-json-array-value",
			input: schema.For[struct {
				Value []string `json:"value" example:"[true]"`
			}],
			error: `invalid string tag value 'true' for field 'Value[0]'`,
		},
		{
			name: "error-json-array-value",
			input: schema.For[struct {
				Value []int `json:"value" example:"[true]"`
			}],
			error: `invalid integer tag value 'true' for field 'Value[0]'`,
		},
		{
			name: "error-json-object",
			input: schema.For[struct {
				Value struct {
					Foo string `json:"foo"`
				} `json:"value" example:"true"`
			}],
			error: `invalid object tag value 'true' for field 'Value'`,
		},
		{
			name: "error-json-object-field",
			input: schema.For[struct {
				Value struct {
					Foo string `json:"foo"`
				} `json:"value" example:"{\"foo\": true}"`
			}],
			error: `invalid string tag value 'true' for field 'Value.foo'`,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			schema, err := testcase.input()
			if testcase.error != "" {
				assert.EqualError(t, err, testcase.error)

				return
			}

			assert.NoError(t, err)
			b, _ := json.Marshal(schema)
			var e bytes.Buffer
			_ = json.Compact(&e, []byte(testcase.expected))
			assert.Equal(t, e.String(), string(b))
		})
	}
}
