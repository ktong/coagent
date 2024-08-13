// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

//nolint:err113
package schema

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

const (
	typeBoolean = "boolean"
	typeInteger = "integer"
	typeNumber  = "number"
	typeString  = "string"
	typeArray   = "array"
	typeObject  = "object"
)

// Schema represents the OpenAI [Supported JSON Schema].
//
// [Supported JSON Schema]: https://platform.openai.com/docs/guides/structured-outputs/supported-schemas
type Schema struct {
	// JSON schema core: section 8.2
	Ref         string             `json:"$ref,omitempty"`  // section 8.2.3.1
	Definitions map[string]*Schema `json:"$defs,omitempty"` // section 8.2.4

	// JSON schema core: section 10.3
	Items                *Schema            `json:"items,omitempty"`                // 10.3.1.2
	Properties           map[string]*Schema `json:"properties,omitempty"`           // 10.3.2.1
	AdditionalProperties any                `json:"additionalProperties,omitempty"` // 10.3.2.3

	// JSON schema validation: section 6.1
	Type any   `json:"type,omitempty"` // 6.1.1
	Enum []any `json:"enum,omitempty"` // 6.1.2

	// JSON schema validation: section 6.5
	Required []string `json:"required,omitempty"` // 6.5.3

	// JSON schema validation: section 7
	Format string `json:"format,omitempty"`

	// JSON schema validation: section 8
	ContentEncoding string `json:"contentEncoding,omitempty"` // 8.3

	// JSON schema validation: section 9
	Title       string `json:"title,omitempty"`       // 9.1
	Description string `json:"description,omitempty"` // 9.1
}

// For generates a JSON schema for the given type using reflection.
//
// It does not support map since additionalProperties must always be set false in objects.
func For[T any]() (_ Schema, err error) { //nolint:nonamedreturns
	// Use panic recovery for simplifying error handling.
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			if err, ok = r.(error); !ok {
				err = fmt.Errorf("%v", r)
			}
		}
	}()

	typ := reflect.TypeOf((*T)(nil)).Elem()
	definitions := make(map[string]*Schema)
	schema := schemaFor(typ, definitions)

	// Delete root schema from definition since it's always referenced using #.
	delete(definitions, typ.String())
	if len(definitions) > 0 {
		schema.Definitions = definitions
	}

	return schema, nil
}

//nolint:cyclop,funlen,gocognit,nonamedreturns
func schemaFor(typ reflect.Type, definitions map[string]*Schema) (def Schema) {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	defer func() {
		if def.Type == typeObject && typ.Name() != "" {
			// Returns a reference to the schema for named struct.
			if schema, ok := definitions[typ.String()]; ok && schema.Ref == "" {
				def = Schema{Ref: ("#/$defs/") + typ.String()}
			}
		}
	}()
	if schema, ok := definitions[typ.String()]; ok {
		return *schema
	}

	var schema Schema
	switch typ.Kind() {
	case reflect.Bool:
		schema.Type = typeBoolean
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema.Type = typeInteger
	case reflect.Float32, reflect.Float64:
		schema.Type = typeNumber
	case reflect.String:
		schema.Type = typeString
	case reflect.Slice, reflect.Array:
		if typ.Elem().Kind() == reflect.Uint8 {
			// Special case: []byte will be serialized as a base64 string.
			schema.Type = typeString
			schema.ContentEncoding = "base64"
		} else {
			schema.Type = typeArray
			elem := schemaFor(typ.Elem(), definitions)
			schema.Items = &elem
		}
	case reflect.Struct:
		schema.Type = typeObject
		schema.AdditionalProperties = false // false must always be set in objects.
		schema.Properties = make(map[string]*Schema)
		if typ.Name() != "" {
			if len(definitions) == 0 {
				// Root schema can be referenced using #.
				definitions[typ.String()] = &Schema{Ref: "#"}
			} else {
				definitions[typ.String()] = &schema
			}
		}

		fieldSet := make(map[string]struct{})
		for _, field := range getFields(typ, make(map[reflect.Type]struct{})) {
			if _, ok := fieldSet[field.Name]; ok {
				// This field was overridden by an ancestor type, so we should ignore it.
				continue
			}
			fieldSet[field.Name] = struct{}{}

			name := field.Name
			if n, _, _ := strings.Cut(field.Tag.Get("json"), ","); n != "" {
				if n == "-" {
					continue // This field is deliberately ignored.
				}
				name = n
			}

			fieldSchema := schemaFor(field.Type, definitions)
			fieldTags(field, &fieldSchema)
			schema.Properties[name] = &fieldSchema
			schema.Required = append(schema.Required, name)
		}
	default:
		panic(fmt.Errorf("unsupported type: %v", typ))
	}

	return schema
}

func getFields(typ reflect.Type, visited map[reflect.Type]struct{}) []reflect.StructField {
	if _, ok := visited[typ]; ok {
		return nil
	}
	visited[typ] = struct{}{}

	fields := make([]reflect.StructField, 0, typ.NumField())
	var embedded []reflect.StructField
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.IsExported() {
			if field.Anonymous {
				embedded = append(embedded, field)
			} else {
				fields = append(fields, field)
			}
		}
	}

	// Put embedded fields at the end to ensure they do not override fields in the current struct.
	for _, field := range embedded {
		switch field.Type.Kind() {
		case reflect.Struct:
			fields = append(fields, getFields(field.Type, visited)...)
		case reflect.Pointer:
			fields = append(fields, getFields(field.Type.Elem(), visited)...)
		default:
		}
	}

	return fields
}

func fieldTags(field reflect.StructField, fieldSchema *Schema) {
	if doc := field.Tag.Get("description"); doc != "" {
		fieldSchema.Description = doc
	}
	if enc := field.Tag.Get("encoding"); enc != "" {
		fieldSchema.ContentEncoding = enc
	}
	if enum := field.Tag.Get("enum"); enum != "" {
		schema := fieldSchema
		if schema.Type == typeArray {
			schema = schema.Items
		}

		var enumValues []any
		for _, e := range strings.Split(enum, ",") {
			enumValues = append(enumValues, jsonTagValue(field.Name, schema, e))
		}
		schema.Enum = enumValues
	}

	if fieldSchema.Ref == "" {
		// Controls whether the field is required or not. All fields start as required,
		// then can be made optional with the `omitempty` JSON tag
		// or it can be overridden manually via the `required` tag.
		fieldRequired := !strings.Contains(field.Tag.Get("json"), "omitempty")
		if _, ok := field.Tag.Lookup("required"); ok {
			fieldRequired = boolTag(field, "required")
		}
		if !fieldRequired {
			fieldSchema.Type = []string{fmt.Sprint(fieldSchema.Type), "null"}
		}
	}
}

func boolTag(field reflect.StructField, tag string) bool {
	value := field.Tag.Get(tag)
	if value == "" {
		return false
	}

	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}

	panic(fmt.Errorf("invalid bool tag '%s' for field '%s': %v", tag, field.Name, value))
}

func jsonTagValue(fieldName string, schema *Schema, value string) any {
	// Special case: strings don't need quotes.
	if schema.Type == typeString {
		return value
	}

	// Special case: array of strings with comma-separated values and no quotes.
	if schema.Type == typeArray && schema.Items != nil && schema.Items.Type == typeString && value[0] != '[' {
		var values []string
		for _, s := range strings.Split(value, ",") {
			values = append(values, strings.TrimSpace(s))
		}

		return values
	}

	var tagValue any
	if err := json.Unmarshal([]byte(value), &tagValue); err != nil {
		panic(fmt.Errorf("invalid %s tag value '%s' for field '%s': %w", schema.Type, value, fieldName, err))
	}
	ensureType(fieldName, schema, value, tagValue)

	return tagValue
}

func ensureType(fieldName string, schema *Schema, value string, tagValue any) { //nolint:cyclop
	switch schema.Type {
	case typeBoolean:
		if _, ok := tagValue.(bool); !ok {
			panic(fmt.Errorf("invalid boolean tag value '%s' for field '%s'", value, fieldName))
		}
	case typeNumber:
		if _, ok := tagValue.(float64); !ok {
			panic(fmt.Errorf("invalid number tag value '%s' for field '%s'", value, fieldName))
		}
	case typeInteger:
		if f, ok := tagValue.(float64); !ok || f != float64(int(f)) {
			panic(fmt.Errorf("invalid integer tag value '%s' for field '%s'", value, fieldName))
		}
	case typeString:
		if _, ok := tagValue.(string); !ok {
			panic(fmt.Errorf("invalid string tag value '%s' for field '%s'", value, fieldName))
		}
	case typeArray:
		items, ok := tagValue.([]any)
		if !ok {
			panic(fmt.Errorf("invalid array tag value '%s' for field '%s'", value, fieldName))
		}

		if schema.Items != nil {
			for i, item := range items {
				b, _ := json.Marshal(item) //nolint:errchkjson
				ensureType(fieldName+"["+strconv.Itoa(i)+"]", schema.Items, string(b), item)
			}
		}
	case typeObject:
		if _, ok := tagValue.(map[string]any); !ok {
			panic(fmt.Errorf("invalid object tag value '%s' for field '%s'", value, fieldName))
		}

		for name, prop := range schema.Properties {
			if val, ok := tagValue.(map[string]any)[name]; ok {
				b, _ := json.Marshal(val) //nolint:errchkjson
				ensureType(fieldName+"."+name, prop, string(b), val)
			}
		}
	}
}
