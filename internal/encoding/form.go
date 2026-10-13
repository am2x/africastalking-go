// Package encoding turns request payloads into the wire formats Africa's
// Talking accepts: form encoding for the v1 endpoints on the api host, and
// JSON for Payments, Mobile Data and parts of Voice.
//
// Every function here takes any rather than a concrete payload type, so this
// package never imports the root package. The dependency runs one way only:
// root imports internal/encoding, never the reverse.
package encoding

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

// formOpts holds the options parsed from a `form` struct tag
type formOpts struct {
	omitEmpty bool
	asJSON    bool
}

// parseFormTag returns the wire name and options for a struct field. A field
// with no `form` tag falls back to its Go name, matching encoding/json.
func parseFormTag(field reflect.StructField) (string, formOpts, error) {
	tag := field.Tag.Get("form")
	if tag == "" {
		return field.Name, formOpts{}, nil
	}

	name, rest, _ := strings.Cut(tag, ",")
	if name == "" {
		name = field.Name // `form:",omitempty"` - options only, keep the Go name
	}

	var opts formOpts
	for rest != "" {
		var opt string
		opt, rest, _ = strings.Cut(rest, ",")
		switch opt {
		case "omitempty":
			opts.omitEmpty = true
		case "json":
			opts.asJSON = true
		default:
			return "", formOpts{}, fmt.Errorf("unknown form tag option %q", opt)
		}
	}
	return name, opts, nil
}

// EncodeForm encodes the exported fields of the struct v as
// application/x-www-form-urlencoded, which is what the v1 endpoints on the
// api host expect. v may be a struct or a pointer to one.
//
// Field names and behaviour come from the `form` struct tag:
//
//	form:"username"            // wire name
//	form:"senderId,omitempty"  // omitted when the value is zero
//	form:"recipients,json"     // marshalled to JSON, sent as a string
//	form:"-"                   // never sent
//
// An untagged field falls back to its Go name. A []string is comma joined,
// matching how Africa's Talking expects recipient lists; any other slice
// needs the json option. Use a pointer field where "unset" and the zero
// value must be distinguished, since omitempty cannot tell them apart.
//
// Floats are rejected rather than encoded: a monetary amount must never go
// over the wire as a binary float. Pass it as a string or minor units.
func EncodeForm(v any) ([]byte, error) {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, errors.New("encoding: form encode of nil pointer")
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("encoding: form encode expects a struct, got %s", rv.Kind())
	}

	values := url.Values{}
	rt := rv.Type()
	for i := range rt.NumField() {
		field, fv := rt.Field(i), rv.Field(i)
		if !field.IsExported() {
			continue
		}
		name, opts, err := parseFormTag(field)
		if err != nil {
			return nil, fmt.Errorf("encoding: field %s: %w", field.Name, err)
		}
		if name == "-" {
			continue
		}
		if opts.omitEmpty && fv.IsZero() {
			continue
		}
		if fv.Kind() == reflect.Pointer { // nil pointer = field absent
			if fv.IsNil() {
				continue
			}
			fv = fv.Elem()
		}
		s, err := encodeFormValue(fv, opts)
		if err != nil {
			return nil, fmt.Errorf("encoding: field %s: %w", field.Name, err)
		}
		values.Set(name, s)
	}
	return []byte(values.Encode()), nil
}

func encodeFormValue(fv reflect.Value, opts formOpts) (string, error) {
	if opts.asJSON {
		b, err := json.Marshal(fv.Interface())
		if err != nil {
			return "", fmt.Errorf("marshal json: %w", err)
		}
		return string(b), nil
	}

	switch fv.Kind() {
	case reflect.String:
		return fv.String(), nil
	case reflect.Bool:
		return strconv.FormatBool(fv.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(fv.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(fv.Uint(), 10), nil
	case reflect.Slice, reflect.Array:
		if fv.Type().Elem().Kind() == reflect.String {
			parts := make([]string, fv.Len())
			for i := range parts {
				parts[i] = fv.Index(i).String()
			}
			return strings.Join(parts, ","), nil
		}
		return "", fmt.Errorf("slice of %s requires the `json` tag option", fv.Type().Elem().Kind())
	default:
		return "", fmt.Errorf("unsupported kind: %s", fv.Kind())
	}
}
