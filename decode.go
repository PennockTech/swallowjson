/*
Package swallowjson provides a utility function for implementing the
encoding/json.Unmarshaler interface's UnmarshalJSON method to decode,
without discarding unknown keys.  The "swallow" concept says that any
keys in a JSON object which do not correspond to a known field in a
(Golang) struct should not be discarded but instead put into a map.

	type MyType struct {
		Foo  string                 `json:"foo"`
		Bar  int                    `json:"bar"`
		Rest map[string]interface{} `json:"-"`
	}

	func (mt *MyType) UnmarshalJSON(raw []byte) error {
		return swallowjson.UnmarshalWith(mt, "Rest", raw)
	}

The struct field to swallow fields not explicitly named must be a
map keyed by string.  The type of map values is handled reliably, returning a
JSON error if unsuitable.
Common types to use might be `interface{}` or `json.RawMessage`.

Errors are either of type swallowjson.SwallowError or are bubbled through from
the json or reflect packages.
*/
package swallowjson // import "go.pennock.tech/swallowjson"

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// SwallowError is the type of all errors generated within the swallowjson package.
// Errors will either be of this type, or bubbled through.
// The string representation of the error is guaranteed to start "swallowjson:".
type SwallowError struct {
	s   string
	aux string
}

func (se SwallowError) Error() string {
	s := "swallowjson: " + se.s
	if se.aux != "" {
		s += ": " + se.aux
	}
	return s
}

// These errors may be returned by UnmarshalWith.
var (
	ErrGivenNonStringKey       = SwallowError{s: "given object with non-string key"}
	ErrMalformedJSON           = SwallowError{s: "given malformed JSON"}
	ErrMissingSpilloverField   = SwallowError{s: "target struct missing specified spillover field"}
	ErrNotGivenMutable         = SwallowError{s: "not given something which we can assign to"}
	ErrNotGivenStruct          = SwallowError{s: "not given a struct in the raw stream"}
	ErrNotStructHolder         = SwallowError{s: "holder is not a struct"}
	ErrSpillNotRightMap        = SwallowError{s: "target's spillover field is not a map[string]interface{}"}
	ErrUnsetableSpilloverField = SwallowError{s: "target struct's spillover field not assignable"}
)

func swallowRuneToken(decoder *json.Decoder, expect rune, failExpect SwallowError) error {
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if t != json.Delim(expect) {
		return SwallowError{
			s:   failExpect.s,
			aux: fmt.Sprintf("expected %q got %q", expect, t),
		}
	}
	return nil
}

// TODO:
// * write more examples
// * check if we really should panic on the cases where we do
// * handle various linters etc

// UnmarshalWith is used for implementing UnmarshalJSON methods to satisfy
// the encoding/json.Unmarshaler interface.  The method becomes a one-liner
// returning the result of calling this function, supplying in addition to the
// object and input only the name of the field which should swallow unknown
// JSON fields.
func UnmarshalWith(target interface{}, spilloverName string, raw []byte) error {
	me := reflect.ValueOf(target)
	if me.Kind() != reflect.Ptr {
		return ErrNotGivenMutable
	}
	me = me.Elem()
	if me.Kind() != reflect.Struct {
		return ErrNotStructHolder
	}
	spillInto := me.FieldByName(spilloverName)
	if spillInto.Kind() == 0 {
		return ErrMissingSpilloverField
	}
	if spillInto.Kind() != reflect.Map {
		return ErrSpillNotRightMap
	}
	if spillInto.Type().Key().Kind() != reflect.String {
		return ErrSpillNotRightMap
	}
	// if the caller specifies a map value type other than interface{}, that's
	// on them; things might work, or they might panic on mismatch.  Panic is
	// the right failure mode, so we just try to Convert and let that panic.
	spillValueType := spillInto.Type().Elem()
	if !spillInto.CanSet() {
		return ErrUnsetableSpilloverField
	}

	met := me.Type()
	fieldsLookup := make(map[string]int, met.NumField()-1)
	// encoding/json has various case-insensitive fallbacks
	// skip that; we don't need to be compatible, this is a _new_ API
	// file a feature request with use-case if want that too
	var (
		sf       reflect.StructField
		tag      string
		sections []string
		jsonName string
	)
	for i := 0; i < met.NumField(); i++ {
		sf = met.Field(i)
		if tag = sf.Tag.Get("json"); tag != "" {
			sections = strings.Split(tag, ",")
			jsonName = sections[0]
			if jsonName != "-" {
				fieldsLookup[jsonName] = i
			}
		} else {
			fieldsLookup[sf.Name] = i
		}
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := swallowRuneToken(dec, '{', ErrNotGivenStruct); err != nil {
		return err
	}
	for dec.More() {
		keyToken, err := dec.Token()
		if err != nil {
			return err
		}
		key, ok := keyToken.(string)
		if !ok {
			return ErrGivenNonStringKey
		}

		// dec.Token() skips over colons!

		var wantType reflect.Type
		if fieldIndex, ok := fieldsLookup[key]; ok {
			wantType = met.Field(fieldIndex).Type
		} else {
			wantType = spillValueType
		}

		vvl := reflect.MakeSlice(reflect.SliceOf(wantType), 1, 1)
		vv := vvl.Index(0)
		err = dec.Decode(vv.Addr().Interface())
		if err != nil {
			return err
		}

		if fieldIndex, ok := fieldsLookup[key]; ok {
			me.Field(fieldIndex).Set(vv.Convert(met.Field(fieldIndex).Type))
		} else {
			kv := reflect.ValueOf(key)
			if spillInto.IsNil() {
				spillInto.Set(reflect.MakeMap(spillInto.Type()))
			}
			spillInto.SetMapIndex(kv, vv.Convert(spillValueType))
		}
	}

	return swallowRuneToken(dec, '}', ErrMalformedJSON)
}
