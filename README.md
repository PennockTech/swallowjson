swallowjson
===========

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

When invoked on `mt`, which should already exist as a struct,
`swallowjson.UnmarshalWith` will populate `Foo` and `Bar` from JSON fields
`foo` and `bar` respectively, per normal Golang decoding rules.  But if the
JSON also contains fields `baz` and `bat` then those will end up as keys,
holding their child data, in the `Rest` map.

This library was written as a fairly quick proof-of-concept for a friend; I've
no current use for it, so have not spent time on tests.  The library is
released in the hopes that it might prove useful to others.

Behavior notes:

* The Golang [encoding/json][] library has a bunch of legacy
  case-insensitivity handling in the default unmarshaller; while swallowjson
  builds upon that library (it uses `json.NewDecoder()` under the hood) our
  API is a new API, implemented in new code and is not a drop-in replacement.
  Thus there is no need to preserve backwards behavior here, so I didn't
  implement that case-insensitivity.
* The `Rest` map will be created on-demand; if no unexpected keys are seen and
  the map is `nil` going in, then it will still be `nil` afterwards.


--
Copyright © 2016 Pennock Tech, LLC  
Licensed per [LICENSE.txt](./LICENSE.txt)

[encoding/json]: https://golang.org/pkg/encoding/json/
