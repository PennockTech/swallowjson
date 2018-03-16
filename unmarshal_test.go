package swallowjson

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"
)

type foo1 struct {
	Foo  string                 `json:"foo"`
	Bar  int                    `json:"bar"`
	Rest map[string]interface{} `json:"-"`
}

type foo2 struct {
	Foo string    `json:"foo"`
	Bar int       `json:"bar"`
	Baz time.Time `json:"baz"`
}

type foo3 struct {
	Foo  string                     `json:"foo"`
	Bar  int                        `json:"bar"`
	Rest map[string]json.RawMessage `json:"-"`
}

type foo4 struct {
	Foo  string         `json:"foo"`
	Bar  int            `json:"bar"`
	Rest map[string]int `json:"-"`
}

type foo5 struct {
	Foo    string `json:"foo"`
	Direct bool
	Rest   map[string]interface{} `json:"-"`
}

func (f *foo1) UnmarshalJSON(raw []byte) error { return UnmarshalWith(f, "Rest", raw) }
func (f *foo3) UnmarshalJSON(raw []byte) error { return UnmarshalWith(f, "Rest", raw) }
func (f *foo4) UnmarshalJSON(raw []byte) error { return UnmarshalWith(f, "Rest", raw) }

const rawA = `{
	"foo": "alpha", "bar": 42, "baz": "2009-11-10T23:00:00Z", "more": "wibble", "num": 3.14159
}
`

const rawB = `{ "foo": "alpha", "bar": 42 }`

const rawC = `{
	"foo": "alpha", "bar": 42, "depth": { "a": 1, "b": 2 }, "arr": [10,20,30], "scalar": "x"
}
`

const rawD = `{
	"foo": "alpha", "bar": 42, "depth": { "a": 1, "b": 2 }, "arr": [10,20,30]
}
`

const rawE = `{
    "foo": "alpha", "bar": 42, "Direct": true, "more": "wibble", "num": 3.14159
}`

func TestDecode(t *testing.T) {
	var (
		f1a     foo1
		f1b     foo1
		f1c     foo1
		f2a     foo2
		f3a     foo3
		f3b     foo3
		f3c     foo3
		f3d     foo3
		f3aTime time.Time
		f4a     foo4
		f5a     foo5
	)
	if err := json.Unmarshal([]byte(rawA), &f1a); err != nil {
		t.Error("foo1/a decode failed", err)
	}
	if err := json.Unmarshal([]byte(rawB), &f1b); err != nil {
		t.Error("foo1/b decode failed", err)
	}
	if err := json.Unmarshal([]byte(rawC), &f1c); err != nil {
		t.Error("foo1/c decode failed", err)
	}
	if err := json.Unmarshal([]byte(rawA), &f2a); err != nil {
		t.Error("foo2/a decode failed", err)
	}
	if err := json.Unmarshal([]byte(rawA), &f3a); err != nil {
		t.Error("foo3/a decode failed", err)
	}
	if err := json.Unmarshal([]byte(rawB), &f3b); err != nil {
		t.Error("foo3/b decode failed", err)
	}
	if err := json.Unmarshal([]byte(rawC), &f3c); err != nil {
		t.Error("foo3/c decode failed", err)
	}
	if err := json.Unmarshal([]byte(rawD), &f3d); err != nil {
		t.Error("foo3/d decode failed", err)
	}
	if err := json.Unmarshal([]byte(rawE), &f5a); err != nil {
		t.Error("foo5/e decode failed", err)
	}

	err := json.Unmarshal([]byte(rawA), &f4a)
	if err == nil {
		t.Error("foo4/a should have failed to decode but didn't")
	} else {
		t.Logf("foo4/a errored as it should have: %s", err)
	}

	if t.Failed() {
		return
	}

	if f1a.Foo != f2a.Foo {
		t.Errorf("field Foo mismatch: %q vs %q", f1a.Foo, f2a.Foo)
	}
	if f1a.Bar != f2a.Bar {
		t.Errorf("field Bar mismatch: %q vs %q", f1a.Bar, f2a.Bar)
	}
	if len(f1a.Rest) != 3 {
		t.Error("foo1/a decode did not pick up three entries into Rest")
	}
	if f1b.Rest != nil {
		t.Errorf("foo1/b has a non-nil Rest field?? Len %d", len(f1b.Rest))
	}

	/*
		t.Logf("foo1/a: %+v", f1a)
		t.Logf("foo1/b: %+v", f1b)
		t.Logf("foo1/c: %#+v", f1c)
		t.Logf("foo2/a: %+v", f2a)
		t.Logf("foo3/a: %+v", f3a)
		t.Logf("foo3/b: %+v", f3b)
		t.Logf("foo3/c: %#+v", f3c)
		t.Logf("foo3/d: %#+v", f3d)
	*/

	var (
		n  float64
		ok bool
	)
	if err := json.Unmarshal(f3a.Rest["baz"], &f3aTime); err != nil {
		t.Errorf("foo3/a baz Time decode of %q failed: %s", f3a.Rest["baz"], err)
	} else {
		t.Logf("foo3/a baz Time: %+v", f3aTime)
	}
	if n, ok = f1a.Rest["num"].(float64); !ok {
		t.Errorf("foo1/a num not stored as a float")
	}
	t.Logf("foo1/a num decode: %#v", n)

	if !f3aTime.Equal(f2a.Baz) {
		t.Errorf("time mismatch direct %v vs RawMessage %v", f2a.Baz, f3aTime)
	}
}

func TestUnmarshalFailsCorrectlyBadJSON(t *testing.T) {
	var f foo1

	for i, entry := range []struct {
		bad  string
		jmsg string // via JSON library
		wmsg string // ours via direct call to UnmarshalWith
	}{
		{"alfa", "invalid character 'a' looking for beginning of value", ""},
		{"foxtrot", "invalid character 'o' in literal false (expecting 'a')", ""},
		{"[]", `swallowjson: not given a struct in the raw stream: expected '{' got "["`, ""},
		{`{"foo": 42}`, "json: cannot unmarshal number into Go value of type string", ""},
		{`{ 42: "foo"}`, "invalid character '4' looking for beginning of object key string", "invalid character '4' "},
		{`{"foo", 42}`, "invalid character ',' after object key", "expected colon after object key"},
		{`{`, "unexpected end of JSON input", "EOF"},
	} {
		err := json.Unmarshal([]byte(entry.bad), &f)
		if err == nil {
			t.Errorf("✗ [%d.J] decode did not fail but should have", i)
		} else {
			got := err.Error()
			if got != entry.jmsg {
				t.Errorf("✗ [%d.J] decode error was %q but expected %q", i, got, entry.jmsg)
			} else {
				t.Logf("✓ [%d.J] decode yielded expected error %q", i, entry.jmsg)
			}
		}
		// Go directly without the protection of the JSON layer, to hit more of our edge-case handling
		err = UnmarshalWith(&f, "Rest", []byte(entry.bad))
		if err == nil {
			t.Errorf("✗ [%d.W] decode did not fail but should have", i)
		} else {
			got := err.Error()
			want := entry.wmsg
			if want == "" {
				want = entry.jmsg
			}
			if got != want {
				t.Errorf("✗ [%d.W] decode error was %q but expected %q", i, got, want)
			} else {
				t.Logf("✓ [%d.W] decode yielded expected error %q", i, want)
			}
		}

	}
}

func expectError(t *testing.T, label string, err error, emsg interface{}) bool {
	if err == nil {
		t.Errorf("✗ %s decode did not error when we expected it to", label)
		return false
	}
	got := err.Error()
	var match bool
	var show string
	switch want := emsg.(type) {
	case string:
		match = got == want
		show = strconv.Quote(want)
	default:
		match = err == want
		show = reflect.TypeOf(want).String()
	}
	if match {
		t.Logf("✓ %s decode got expected error %s", label, show)
		return true
	}
	t.Errorf("✗ %s decode error was %q but expected %s", label, got, show)
	return false
}

func labelOf(item interface{}, which rune) string {
	switch t := item.(type) {
	case int:
		return fmt.Sprintf("[%d.%c]", t, which)
	case string:
		return fmt.Sprintf("%q.%c", t, which)
	default:
		panic("bad type")
	}
}

func failEachWay(t *testing.T, index interface{}, obj interface{}, data string, jmsg string, wmsg interface{}) {
	if wmsg == nil {
		wmsg = jmsg
	}
	err := json.Unmarshal([]byte(data), obj)
	if jmsg != "" || err != nil {
		expectError(t, labelOf(index, 'J'), err, jmsg)
	} else if err == nil {
		t.Logf("✓ %s decode did not error", labelOf(index, 'J'))
	}
	err = UnmarshalWith(obj, "Rest", []byte(data))
	expectError(t, labelOf(index, 'W'), err, wmsg)
}

func TestFailsCorrectlyBadStruct(t *testing.T) {
	var (
		oneItem       foo1
		sliceItem     []foo1
		noSpill       struct{}
		badSpill      struct{ Rest string }
		badSpillKey   struct{ Rest map[int]json.RawMessage }
		intSpillValue struct{ Rest map[string]int }

		// For these, if you can see a way to trigger these, please file a bug
		// with details, because I do want to test them.

		// don't see a way to trigger that the .Elem() [value] is not .CanSet()
		// so can't trigger ErrUnsetableSpilloverField ?

		// looks like the stateful parsing in json decoder.Token() will never
		// successfully return a non-string token in the key position of an
		// object, so can't trigger ErrGivenNonStringKey ?
	)

	for i, entry := range []struct {
		obj  interface{}
		jmsg string      // via JSON library
		wmsg interface{} // ours via direct call to UnmarshalWith
	}{
		{oneItem, "json: Unmarshal(non-pointer swallowjson.foo1)", ErrNotGivenMutable},
		{sliceItem, "json: Unmarshal(non-pointer []swallowjson.foo1)", ErrNotGivenMutable},
		{&sliceItem, "json: cannot unmarshal object into Go value of type []swallowjson.foo1", ErrNotStructHolder},
		{&noSpill, "", ErrMissingSpilloverField},
		{&badSpill, "", ErrSpillNotRightMap},
		{&badSpillKey, "", ErrSpillNotRightMap},
		{&intSpillValue, "", "json: cannot unmarshal string into Go value of type int"},
	} {
		failEachWay(t, i, entry.obj, rawE, entry.jmsg, entry.wmsg)
	}
}
