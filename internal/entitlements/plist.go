package entitlements

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

// ValueKind is the typed plist value variant used by entitlement entries.
type ValueKind uint8

const (
	ValueKindUnknown ValueKind = iota
	ValueKindString
	ValueKindBool
	ValueKindStringArray
)

// Value stores a typed entitlement plist value.
type Value struct {
	Kind        ValueKind
	StringValue string
	BoolValue   bool
	ArrayValue  []string
}

// Document is an in-memory representation of a plist dict.
type Document struct {
	entries map[string]Value
	order   []string
}

// NewDocument creates an empty entitlement plist document.
func NewDocument() *Document {
	return &Document{
		entries: make(map[string]Value),
		order:   make([]string, 0),
	}
}

// LoadPlistFile reads and parses an XML plist file from disk.
func LoadPlistFile(path string) (*Document, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read plist %q: %w", path, err)
	}

	doc, err := ParsePlistXML(payload)
	if err != nil {
		return nil, fmt.Errorf("parse plist %q: %w", path, err)
	}

	return doc, nil
}

// WritePlistFile serializes and writes an XML plist file to disk.
func WritePlistFile(path string, doc *Document) error {
	payload, err := MarshalPlistXML(doc)
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("write plist %q: %w", path, err)
	}

	return nil
}

// ParsePlistXML parses a plist XML payload into an in-memory document.
func ParsePlistXML(payload []byte) (*Document, error) {
	decoder := xml.NewDecoder(bytes.NewReader(payload))

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("decode plist XML: %w", err)
		}

		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "plist" {
			continue
		}

		return parsePlistDict(decoder)
	}

	return nil, fmt.Errorf("plist root element not found")
}

// MarshalPlistXML serializes a document into a plist XML payload.
func MarshalPlistXML(doc *Document) ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("plist document is required")
	}

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	b.WriteString(`<plist version="1.0">` + "\n")
	b.WriteString(`<dict>` + "\n")

	for _, key := range doc.order {
		value, ok := doc.entries[key]
		if !ok {
			continue
		}

		b.WriteString("\t<key>" + escapeXML(key) + "</key>\n")
		if err := writeValue(&b, value); err != nil {
			return nil, fmt.Errorf("marshal plist key %q: %w", key, err)
		}
	}

	b.WriteString(`</dict>` + "\n")
	b.WriteString(`</plist>` + "\n")

	return []byte(b.String()), nil
}

// Set inserts or updates a key in the document.
func (d *Document) Set(key string, value Value) {
	if d == nil {
		return
	}

	normalizedKey := strings.TrimSpace(key)
	if normalizedKey == "" {
		return
	}

	normalizedValue := cloneValue(value)

	if d.entries == nil {
		d.entries = make(map[string]Value)
	}

	if _, exists := d.entries[normalizedKey]; !exists {
		d.order = append(d.order, normalizedKey)
	}

	d.entries[normalizedKey] = normalizedValue
}

// Remove deletes a key from the document. Returns true when a key was removed.
func (d *Document) Remove(key string) bool {
	if d == nil {
		return false
	}

	normalizedKey := strings.TrimSpace(key)
	if normalizedKey == "" {
		return false
	}

	if _, exists := d.entries[normalizedKey]; !exists {
		return false
	}

	delete(d.entries, normalizedKey)
	for index, existingKey := range d.order {
		if existingKey != normalizedKey {
			continue
		}
		d.order = append(d.order[:index], d.order[index+1:]...)
		break
	}

	return true
}

// Get returns a key value when present.
func (d *Document) Get(key string) (Value, bool) {
	if d == nil {
		return Value{}, false
	}

	normalizedKey := strings.TrimSpace(key)
	value, ok := d.entries[normalizedKey]
	if !ok {
		return Value{}, false
	}

	return cloneValue(value), true
}

// Keys returns document keys in file order.
func (d *Document) Keys() []string {
	if d == nil {
		return nil
	}

	keys := make([]string, len(d.order))
	copy(keys, d.order)
	return keys
}

func parsePlistDict(decoder *xml.Decoder) (*Document, error) {
	for {
		token, err := nextSignificantToken(decoder)
		if err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("plist dict element not found")
			}
			return nil, err
		}

		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		if start.Name.Local != "dict" {
			return nil, fmt.Errorf("unexpected element <%s>, expected <dict>", start.Name.Local)
		}

		return parseDictContents(decoder)
	}
}

func parseDictContents(decoder *xml.Decoder) (*Document, error) {
	doc := NewDocument()

	for {
		token, err := nextSignificantToken(decoder)
		if err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("unexpected EOF while parsing plist dict")
			}
			return nil, err
		}

		switch typed := token.(type) {
		case xml.EndElement:
			if typed.Name.Local == "dict" {
				return doc, nil
			}
			return nil, fmt.Errorf("unexpected closing element </%s> in plist dict", typed.Name.Local)
		case xml.StartElement:
			if typed.Name.Local != "key" {
				return nil, fmt.Errorf("unexpected element <%s> in plist dict, expected <key>", typed.Name.Local)
			}

			var key string
			if err := decoder.DecodeElement(&key, &typed); err != nil {
				return nil, fmt.Errorf("decode plist key: %w", err)
			}

			valueToken, err := nextSignificantToken(decoder)
			if err != nil {
				if err == io.EOF {
					return nil, fmt.Errorf("missing value for plist key %q", key)
				}
				return nil, err
			}

			valueStart, ok := valueToken.(xml.StartElement)
			if !ok {
				return nil, fmt.Errorf("missing value element for plist key %q", key)
			}

			value, err := parseValue(decoder, valueStart)
			if err != nil {
				return nil, fmt.Errorf("parse value for plist key %q: %w", key, err)
			}

			doc.Set(key, value)
		default:
			return nil, fmt.Errorf("unexpected token %T in plist dict", token)
		}
	}
}

func parseValue(decoder *xml.Decoder, start xml.StartElement) (Value, error) {
	switch start.Name.Local {
	case "string":
		var content string
		if err := decoder.DecodeElement(&content, &start); err != nil {
			return Value{}, fmt.Errorf("decode string value: %w", err)
		}
		return Value{Kind: ValueKindString, StringValue: content}, nil
	case "true":
		if err := decoder.Skip(); err != nil {
			return Value{}, fmt.Errorf("decode true value: %w", err)
		}
		return Value{Kind: ValueKindBool, BoolValue: true}, nil
	case "false":
		if err := decoder.Skip(); err != nil {
			return Value{}, fmt.Errorf("decode false value: %w", err)
		}
		return Value{Kind: ValueKindBool, BoolValue: false}, nil
	case "array":
		items, err := parseStringArray(decoder)
		if err != nil {
			return Value{}, err
		}
		return Value{Kind: ValueKindStringArray, ArrayValue: items}, nil
	default:
		return Value{}, fmt.Errorf("unsupported plist value type <%s>", start.Name.Local)
	}
}

func parseStringArray(decoder *xml.Decoder) ([]string, error) {
	items := make([]string, 0)

	for {
		token, err := nextSignificantToken(decoder)
		if err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("unexpected EOF in array value")
			}
			return nil, err
		}

		switch typed := token.(type) {
		case xml.EndElement:
			if typed.Name.Local == "array" {
				return items, nil
			}
			return nil, fmt.Errorf("unexpected closing element </%s> in array", typed.Name.Local)
		case xml.StartElement:
			if typed.Name.Local != "string" {
				return nil, fmt.Errorf("unsupported array item type <%s>", typed.Name.Local)
			}

			var item string
			if err := decoder.DecodeElement(&item, &typed); err != nil {
				return nil, fmt.Errorf("decode array item: %w", err)
			}
			items = append(items, item)
		default:
			return nil, fmt.Errorf("unexpected token %T in array", token)
		}
	}
}

func nextSignificantToken(decoder *xml.Decoder) (xml.Token, error) {
	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}

		switch typed := token.(type) {
		case xml.CharData:
			if strings.TrimSpace(string(typed)) == "" {
				continue
			}
			return token, nil
		case xml.Comment:
			continue
		default:
			return token, nil
		}
	}
}

func writeValue(builder *strings.Builder, value Value) error {
	switch value.Kind {
	case ValueKindString:
		builder.WriteString("\t<string>" + escapeXML(value.StringValue) + "</string>\n")
	case ValueKindBool:
		if value.BoolValue {
			builder.WriteString("\t<true/>\n")
		} else {
			builder.WriteString("\t<false/>\n")
		}
	case ValueKindStringArray:
		builder.WriteString("\t<array>\n")
		for _, item := range value.ArrayValue {
			builder.WriteString("\t\t<string>" + escapeXML(item) + "</string>\n")
		}
		builder.WriteString("\t</array>\n")
	default:
		return fmt.Errorf("unsupported value kind %d", value.Kind)
	}

	return nil
}

func cloneValue(value Value) Value {
	cloned := value
	if value.ArrayValue != nil {
		cloned.ArrayValue = make([]string, len(value.ArrayValue))
		copy(cloned.ArrayValue, value.ArrayValue)
	}
	return cloned
}

func escapeXML(value string) string {
	var b bytes.Buffer
	if err := xml.EscapeText(&b, []byte(value)); err != nil {
		return value
	}
	return b.String()
}
