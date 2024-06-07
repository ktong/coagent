// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

//nolint:err113
package schema

import (
	"encoding/json"
	"fmt"
	"math/bits"
	"net"
	"net/netip"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type DataType string

const (
	TypeBoolean DataType = "boolean"
	TypeInteger DataType = "integer"
	TypeNumber  DataType = "number"
	TypeString  DataType = "string"
	TypeArray   DataType = "array"
	TypeObject  DataType = "object"
)

type Schema struct {
	// JSON schema core: section 10.3
	Items                *Schema            `json:"items,omitempty"`                // 10.3.1.2
	Properties           map[string]*Schema `json:"properties,omitempty"`           // 10.3.2.1
	AdditionalProperties any                `json:"additionalProperties,omitempty"` // 10.3.2.3

	// JSON schema validation: section 6.1
	Type DataType `json:"type,omitempty"` // 6.1.1
	Enum []any    `json:"enum,omitempty"` // 6.1.2

	// JSON schema validation: section 6.2
	MultipleOf       *float64 `json:"multipleOf,omitempty"`       // 6.2.1
	Maximum          *float64 `json:"maximum,omitempty"`          // 6.2.2
	ExclusiveMaximum *float64 `json:"exclusiveMaximum,omitempty"` // 6.2.3
	Minimum          *float64 `json:"minimum,omitempty"`          // 6.2.4
	ExclusiveMinimum *float64 `json:"exclusiveMinimum,omitempty"` // 6.2.5

	// JSON schema validation: section 6.3
	MaxLength *int   `json:"maxLength,omitempty"` // 6.3.1
	MinLength *int   `json:"minLength,omitempty"` // 6.3.2
	Pattern   string `json:"pattern,omitempty"`   // 6.3.3

	// JSON schema validation: section 6.4
	MaxItems    *int `json:"maxItems,omitempty"`    // 6.4.1
	MinItems    *int `json:"minItems,omitempty"`    // 6.4.2
	UniqueItems bool `json:"uniqueItems,omitempty"` // 6.4.3

	// JSON schema validation: section 6.5
	MaxProperties *int     `json:"maxProperties,omitempty"` // 6.5.1
	MinProperties *int     `json:"minProperties,omitempty"` // 6.5.2
	Required      []string `json:"required,omitempty"`      // 6.5.3

	// JSON schema validation: section 7
	Format string `json:"format,omitempty"`

	// JSON schema validation: section 8
	ContentEncoding string `json:"contentEncoding,omitempty"` // 8.3

	// JSON schema validation: section 9
	Title       string `json:"title,omitempty"`       // 9.1
	Description string `json:"description,omitempty"` // 9.1
	Examples    []any  `json:"examples,omitempty"`    // 9.5
}

func For[T any]() (_ *Schema, err error) { //nolint:nonamedreturns
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			if err, ok = r.(error); !ok {
				err = fmt.Errorf("%v", r)
			}
		}
	}()

	return schemaFor(reflect.TypeOf((*T)(nil)).Elem()), nil
}

func schemaFor(typ reflect.Type) *Schema { //nolint:cyclop,funlen,gocognit,gocyclo
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	// Handle special cases.
	// JSON schema validation: section 7.3
	switch reflect.Zero(typ).Interface().(type) {
	case time.Time:
		return &Schema{Type: TypeString, Format: "date-time"}
	case url.URL:
		return &Schema{Type: TypeString, Format: "uri"}
	case net.IP:
		return &Schema{Type: TypeString, Format: "ipv4"}
	case netip.Addr:
		return &Schema{Type: TypeString, Format: "ipv4"}
	case json.RawMessage:
		return &Schema{}
	}

	minZero := 0.0
	switch typ.Kind() {
	case reflect.Bool:
		return &Schema{Type: TypeBoolean}
	case reflect.Int:
		format := "int64"
		if bits.UintSize == 32 { //nolint:mnd
			format = "int32"
		}

		return &Schema{Type: TypeInteger, Format: format}
	case reflect.Int8, reflect.Int16, reflect.Int32:
		return &Schema{Type: TypeInteger, Format: "int32"}
	case reflect.Int64:
		return &Schema{Type: TypeInteger, Format: "int64"}
	case reflect.Uint:
		format := "int64"
		if bits.UintSize == 32 { //nolint:mnd
			format = "int32"
		}

		return &Schema{Type: TypeInteger, Format: format, Minimum: &minZero}
	case reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return &Schema{Type: TypeInteger, Format: "int32", Minimum: &minZero}
	case reflect.Uint64:
		return &Schema{Type: TypeInteger, Format: "int64", Minimum: &minZero}
	case reflect.Float32:
		return &Schema{Type: TypeNumber, Format: "float"}
	case reflect.Float64:
		return &Schema{Type: TypeNumber, Format: "double"}
	case reflect.String:
		return &Schema{Type: TypeString}
	case reflect.Slice, reflect.Array:
		if typ.Elem().Kind() == reflect.Uint8 {
			// Special case: []byte will be serialized as a base64 string.
			return &Schema{Type: TypeString, ContentEncoding: "base64"}
		}

		schema := &Schema{Type: TypeArray, Items: schemaFor(typ.Elem())}
		if typ.Kind() == reflect.Array {
			l := typ.Len()
			schema.MaxItems = &l
			schema.MinItems = &l
		}

		return schema
	case reflect.Map:
		return &Schema{Type: TypeObject, AdditionalProperties: schemaFor(typ.Elem())}
	case reflect.Struct:
		var required []string
		fieldSet := make(map[string]struct{})
		props := make(map[string]*Schema)

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

			// Controls whether the field is required or not. All fields start as required,
			// then can be made optional with the `omitempty` JSON tag
			// or it can be overridden manually via the `required` tag.
			fieldRequired := !strings.Contains(field.Tag.Get("json"), "omitempty")
			if _, ok := field.Tag.Lookup("required"); ok {
				fieldRequired = boolTag(field, "required")
			}

			if fs := schemaForField(field); fs != nil {
				props[name] = fs
				if fieldRequired {
					required = append(required, name)
				}
			}
		}

		additionalProps := false
		if f, ok := typ.FieldByName("_"); ok {
			if _, ok = f.Tag.Lookup("additionalProperties"); ok {
				additionalProps = boolTag(f, "additionalProperties")
			}
		}

		return &Schema{Type: TypeObject, Properties: props, Required: required, AdditionalProperties: additionalProps}
	default:
		return &Schema{}
	}
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

func schemaForField(field reflect.StructField) *Schema {
	fieldSchema := schemaFor(field.Type)
	if doc := field.Tag.Get("description"); doc != "" {
		fieldSchema.Description = doc
	}
	if format := field.Tag.Get("format"); format != "" {
		fieldSchema.Format = format
	}
	if enc := field.Tag.Get("encoding"); enc != "" {
		fieldSchema.ContentEncoding = enc
	}
	if value := field.Tag.Get("example"); value != "" {
		if e := jsonTagValue(field.Name, fieldSchema, value); e != nil {
			fieldSchema.Examples = []any{e}
		}
	}
	if enum := field.Tag.Get("enum"); enum != "" {
		schema := fieldSchema
		if schema.Type == TypeArray {
			schema = schema.Items
		}

		var enumValues []any
		for _, e := range strings.Split(enum, ",") {
			enumValues = append(enumValues, jsonTagValue(field.Name, schema, e))
		}
		schema.Enum = enumValues
	}

	fieldSchema.MultipleOf = floatTag(field, "multipleOf")
	fieldSchema.Maximum = floatTag(field, "maximum")
	fieldSchema.ExclusiveMaximum = floatTag(field, "exclusiveMaximum")
	fieldSchema.Minimum = floatTag(field, "minimum")
	fieldSchema.ExclusiveMinimum = floatTag(field, "exclusiveMinimum")

	fieldSchema.MinLength = intTag(field, "minLength")
	fieldSchema.MaxLength = intTag(field, "maxLength")
	fieldSchema.Pattern = field.Tag.Get("pattern")

	fieldSchema.MinItems = intTag(field, "minItems")
	fieldSchema.MaxItems = intTag(field, "maxItems")
	fieldSchema.UniqueItems = boolTag(field, "uniqueItems")

	fieldSchema.MinProperties = intTag(field, "minProperties")
	fieldSchema.MaxProperties = intTag(field, "maxProperties")

	return fieldSchema
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

func intTag(field reflect.StructField, tag string) *int {
	value := field.Tag.Get(tag)
	if value == "" {
		return nil
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		panic(fmt.Errorf("invalid int tag '%s' for field '%s': %v (%w)", tag, field.Name, value, err))
	}

	return &intValue
}

func floatTag(field reflect.StructField, tag string) *float64 {
	value := field.Tag.Get(tag)
	if value == "" {
		return nil
	}

	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		panic(fmt.Errorf("invalid float tag '%s' for field '%s': %v (%w)", tag, field.Name, value, err))
	}

	return &floatValue
}

func jsonTagValue(fieldName string, schema *Schema, value string) any {
	// Special case: strings don't need quotes.
	if schema.Type == TypeString {
		return value
	}

	// Special case: array of strings with comma-separated values and no quotes.
	if schema.Type == TypeArray && schema.Items != nil && schema.Items.Type == TypeString && value[0] != '[' {
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
	case TypeBoolean:
		if _, ok := tagValue.(bool); !ok {
			panic(fmt.Errorf("invalid boolean tag value '%s' for field '%s'", value, fieldName))
		}
	case TypeNumber:
		if _, ok := tagValue.(float64); !ok {
			panic(fmt.Errorf("invalid number tag value '%s' for field '%s'", value, fieldName))
		}
	case TypeInteger:
		if f, ok := tagValue.(float64); !ok || f != float64(int(f)) {
			panic(fmt.Errorf("invalid integer tag value '%s' for field '%s'", value, fieldName))
		}
	case TypeString:
		if _, ok := tagValue.(string); !ok {
			panic(fmt.Errorf("invalid string tag value '%s' for field '%s'", value, fieldName))
		}
	case TypeArray:
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
	case TypeObject:
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
