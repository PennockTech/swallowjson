/*
Package swallowjson provides a utility function for implementing the
encoding/json.Unmarshaler interface's UnmarshalJSON method to decode
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

*/
package swallowjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

var ErrGivenNonStringKey = errors.New("spillover: given object with non-string key")
var ErrMalformedJSON = errors.New("spillover: given malformed JSON")
var ErrMissingSpilloverField = errors.New("spillover: target struct missing specified spillover field")
var ErrNotGivenMutable = errors.New("spillover: not given something which we can assign to")
var ErrNotGivenStruct = errors.New("spillover: not given a struct in the raw stream")
var ErrNotStructHolder = errors.New("spillover: holder is not a struct")
var ErrSpillNotRightMap = errors.New("spillover: target's spillover field is not a map[string]interface{}")
var ErrUnsetableSpilloverField = errors.New("spillover: target struct's spillover field not assignable")

func swallowRuneToken(decoder *json.Decoder, expect rune, failExpect error) error {
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if t != json.Delim(expect) {
		fmt.Printf("Expected %q got %q\n", expect, t)
		return failExpect
	}
	return nil
}

// TODO:
// * write tests
// * write more examples
// * reconsider error types
// * check if we really should panic on the cases where we do
// * handle various linters etc

// UnmarshalWith is used for implementing UnmarshalJSON methods to satisfy
// the encoding/json.Unmarshaler interface.  The method becomes a one-liner
// returning the result of calling this function, supplying only the name of
// the field which should swallow unknown JSON fields.
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
	for i := 0; i < me.Type().NumField(); i++ {
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

		var v interface{}
		err = dec.Decode(&v)
		if err != nil {
			return err
		}
		vv := reflect.ValueOf(v)

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
	if err := swallowRuneToken(dec, '}', ErrMalformedJSON); err != nil {
		return err
	}

	return nil
}
