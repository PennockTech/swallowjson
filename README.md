swallowjson
===========

[![Continuous Integration](https://secure.travis-ci.org/PennockTech/swallowjson.svg?branch=main)](http://travis-ci.org/PennockTech/swallowjson)
[![Documentation](https://godoc.org/go.pennock.tech/swallowjson?status.svg)](https://godoc.org/go.pennock.tech/swallowjson)
[![Coverage Status](https://coveralls.io/repos/github/PennockTech/swallowjson/badge.svg)](https://coveralls.io/github/PennockTech/swallowjson)

The `swallowjson` Golang library provides a simple-to-use support function to
use to implement an interface method in your type, to aid in JSON decoding.

When decoding JSON into a struct, the default Golang handling is to discard
the JSON fields encountered which have not been declared as struct fields.
The `UnmarshalWith` function lets you implement the `UnmarshalJSON` method as
a one-liner, declaring a map within the struct which should swallow all JSON
fields not otherwise accepted by the struct.

A simple example:

```go
type MyType struct {
	Foo  string                 `json:"foo"`
	Bar  int                    `json:"bar"`
	Rest map[string]interface{} `json:"-"`
}

func (mt *MyType) UnmarshalJSON(raw []byte) error {
	return swallowjson.UnmarshalWith(mt, "Rest", raw)
}
```

You can then decode as normal for Golang, letting the JSON decoder dispatch to
your overridden `UnmarshalJSON` method whenever it expects to decode a
`MyType` (whether at the top-level, or nested inside other types, etc):

```go
var myData MyType
if err := json.Unmarshal(rawBytes, &myData); err != nil {
	processError(err)
}
```

When invoked on `mt`, which should already exist as a struct,
`swallowjson.UnmarshalWith` will populate `Foo` and `Bar` from JSON fields
`foo` and `bar` respectively, per normal Golang decoding rules.  But if the
JSON also contains fields `baz` and `bat` then those will end up as keys,
holding their child data, in the `Rest` map.

This library was written as a fairly quick proof-of-concept for a friend; I've
no current use for it, so this has only rudimentary tests and has not seen
heavy production usage to battle-test it.
The library is released in the hopes that it might prove useful to others.

Behavior notes:

* The Golang [encoding/json][] library has a bunch of legacy
  case-insensitivity handling in the default unmarshaller; while swallowjson
  builds upon that library (it uses `json.NewDecoder()` under the hood) our
  API is a new API, implemented in new code and is not a drop-in replacement.
  Thus there is no need to preserve backwards behavior here, so I didn't
  implement that case-insensitivity.
* The `Rest` map will be created on-demand; if no unexpected keys are seen and
  the map is `nil` going in, then it will still be `nil` afterwards.
* The `Rest` map can have arbitrary value types, but if the content won't
  parse then you'll get an error.  Sensible choices for generic usage include
  `interface{}` and `json.RawMessage`.

Canonical import path is: `go.pennock.tech/swallowjson`

---
Copyright © 2016 Pennock Tech, LLC  
Licensed per [LICENSE.txt](./LICENSE.txt)

[encoding/json]: https://golang.org/pkg/encoding/json/
