/*
Basic json-schema generator based on Go types, for easy interchange of Go
structures between diferent languages.
*/
package jsonschema

import (
	"encoding/json"
	"reflect"
	"strings"
)

const DEFAULT_SCHEMA = "http://json-schema.org/schema#"

type Document struct {
	Schema string `json:"$schema,omitempty"`
	Property
}

// Reads the variable structure into the JSON-Schema Document
func (d *Document) Read(variable interface{}) {
	d.setDefaultSchema()

	value := reflect.ValueOf(variable)
	d.read(value.Type(), "")
}

// ReadDeep reads the variable structure into the JSON-Schema Document
func (d *Document) ReadDeep(variable interface{}) {
	d.setDefaultSchema()

	value := reflect.ValueOf(variable)
	d.readDeep(value, "")
}

func (d *Document) setDefaultSchema() {
	if d.Schema == "" {
		d.Schema = DEFAULT_SCHEMA
	}
}

// Marshal returns the JSON encoding of the Document
func (d *Document) Marshal() ([]byte, error) {
	return json.MarshalIndent(d, "", "    ")
}

// String return the JSON encoding of the Document as a string
func (d *Document) String() string {
	jsonBytes, _ := d.Marshal()
	return string(jsonBytes)
}

type Property struct {
	Type                 string               `json:"type,omitempty"`
	Format               string               `json:"format,omitempty"`
	Items                *Property            `json:"items,omitempty"`
	Properties           map[string]*Property `json:"properties,omitempty"`
	Required             []string             `json:"required,omitempty"`
	AdditionalProperties bool                 `json:"additionalProperties,omitempty"`
	Description          string               `json:"description,omitempty"`
	Default              string               `json:"default,omitempty"`
	Deprecated           string               `json:"deprecated,omitempty"`
}

func (p *Property) read(t reflect.Type, opts tagOptions) {
	jsType, format, kind := getTypeFromMapping(t)
	if jsType != "" {
		p.Type = jsType
	}
	if format != "" {
		p.Format = format
	}

	switch kind {
	case reflect.Slice:
		p.readFromSlice(t)
	case reflect.Map:
		p.readFromMap(t)
	case reflect.Struct:
		p.readFromStruct(t)
	case reflect.Ptr:
		p.read(t.Elem(), opts)
	}
}

func (p *Property) readDeep(v reflect.Value, opts tagOptions) {
	if !v.IsValid() {
		p.Type = "null"
		return
	}
	jsType, format, kind := getTypeFromMapping(v.Type())
	if jsType != "" {
		p.Type = jsType
	}
	if format != "" {
		p.Format = format
	}

	switch kind {
	case reflect.Slice:
		p.readFromSliceDeep(v)
	case reflect.Map:
		p.readFromMapDeep(v)
	case reflect.Struct:
		p.readFromStructDeep(v)
	case reflect.Ptr, reflect.Interface:
		p.readDeep(v.Elem(), opts)
	}
}

func (p *Property) readFromSlice(t reflect.Type) {
	jsType, _, kind := getTypeFromMapping(t.Elem())
	if kind == reflect.Uint8 {
		p.Type = "string"
	} else if jsType != "" {
		p.Items = &Property{}
		p.Items.read(t.Elem(), "")
	}
}

func (p *Property) readFromSliceDeep(v reflect.Value) {
	if v.Len() == 0 {
		t := v.Type()
		jsType, _, kind := getTypeFromMapping(t.Elem())
		if kind == reflect.Uint8 {
			p.Type = "string"
		} else if jsType != "" {
			p.Items = &Property{}
			if v.Len() == 0 {
				p.Items.read(t.Elem(), "")
				return
			}
			p.Items.readDeep(v.Index(0), "")
		}
		return
	}

	_, _, kind := getTypeFromMapping(v.Index(0).Type())
	if kind == reflect.Uint8 {
		p.Type = "string"
	} else {
		p.Items = &Property{}
		p.Items.readDeep(v.Index(0), "")
	}
}

func (p *Property) readFromMap(t reflect.Type) {
	jsType, format, _ := getTypeFromMapping(t.Elem())

	if jsType != "" {
		p.Properties = make(map[string]*Property, 0)
		p.Properties[".*"] = &Property{Type: jsType, Format: format}
	} else {
		p.AdditionalProperties = true
	}
}

func (p *Property) readFromMapDeep(v reflect.Value) {
	properties := make(map[string]*Property)
	iter := v.MapRange()
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()
		keyName := mapKeyToString(key)
		properties[keyName] = &Property{}
		properties[keyName].readDeep(value, "")
	}

	if len(properties) > 0 {
		p.Properties = properties
	}
}

func mapKeyToString(key reflect.Value) string {
	keyKind := key.Kind()

	if keyKind == reflect.Interface {
		return mapKeyToString(key.Elem())
	}

	return key.String()
}

func (p *Property) readFromStruct(t reflect.Type) {
	p.Type = "object"
	p.Properties = make(map[string]*Property, 0)
	p.AdditionalProperties = false

	count := t.NumField()
	for i := 0; i < count; i++ {
		field := t.Field(i)

		tag := field.Tag.Get("json")
		name, opts := parseTag(tag)
		if name == "" {
			name = field.Name
		}
		if name == "-" {
			continue
		}

		if field.Anonymous {
			embeddedProperty := &Property{}
			embeddedProperty.read(field.Type, opts)

			for name, property := range embeddedProperty.Properties {
				p.Properties[name] = property
			}
			p.Required = append(p.Required, embeddedProperty.Required...)

			continue
		}

		p.Properties[name] = &Property{}
		p.Properties[name].read(field.Type, opts)

		if !opts.Contains("omitempty") {
			p.Required = append(p.Required, name)
		}
	}
}

func (p *Property) readFromStructDeep(v reflect.Value) {
	t := v.Type()
	p.Type = "object"
	p.Properties = make(map[string]*Property, 0)
	p.AdditionalProperties = false

	count := t.NumField()
	for i := 0; i < count; i++ {
		field := t.Field(i)

		tag := field.Tag.Get("json")
		name, opts := parseTag(tag)
		if name == "" {
			name = field.Name
		}
		if name == "-" {
			continue
		}

		if field.Anonymous {
			embeddedProperty := &Property{}
			embeddedProperty.readDeep(v.Field(i), opts)

			for name, Property := range embeddedProperty.Properties {
				p.Properties[name] = Property
			}
			p.Required = append(p.Required, embeddedProperty.Required...)

			continue
		}

		p.Properties[name] = &Property{}
		p.Properties[name].readDeep(v.Field(i), opts)

		if !opts.Contains("omitempty") {
			p.Required = append(p.Required, name)
		}
	}
}

var formatMapping = map[string][]string{
	"time.Time": {"string", "date-time"},
}

var kindMapping = map[reflect.Kind]string{
	reflect.Bool:    "boolean",
	reflect.Int:     "integer",
	reflect.Int8:    "integer",
	reflect.Int16:   "integer",
	reflect.Int32:   "integer",
	reflect.Int64:   "integer",
	reflect.Uint:    "integer",
	reflect.Uint8:   "integer",
	reflect.Uint16:  "integer",
	reflect.Uint32:  "integer",
	reflect.Uint64:  "integer",
	reflect.Float32: "number",
	reflect.Float64: "number",
	reflect.String:  "string",
	reflect.Slice:   "array",
	reflect.Struct:  "object",
	reflect.Map:     "object",
}

func getTypeFromMapping(t reflect.Type) (string, string, reflect.Kind) {
	if v, ok := formatMapping[t.String()]; ok {
		return v[0], v[1], reflect.String
	}

	kind := t.Kind()
	if v, ok := kindMapping[kind]; ok {
		return v, "", kind
	}

	return "", "", kind
}

type tagOptions string

func parseTag(tag string) (string, tagOptions) {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx], tagOptions(tag[idx+1:])
	}
	return tag, ""
}

func (o tagOptions) Contains(optionName string) bool {
	if len(o) == 0 {
		return false
	}

	s := string(o)
	for s != "" {
		var next string
		i := strings.Index(s, ",")
		if i >= 0 {
			s, next = s[:i], s[i+1:]
		}
		if s == optionName {
			return true
		}
		s = next
	}
	return false
}
