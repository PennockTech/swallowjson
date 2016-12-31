package swallowjson

import (
	"encoding/json"
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

	err := json.Unmarshal([]byte(rawA), &f4a)
	if err == nil {
		t.Error("foo4/a should have failed to decode but didn't")
	} else {
		t.Logf("foo4/a errored as it should have: %s", err)
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
