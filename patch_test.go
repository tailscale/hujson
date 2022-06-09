// Copyright (c) 2021 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hujson

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var testdataPatch = []struct {
	in      string
	patch   string
	want    string
	wantErr error
}{{
	// RFC 6902, appendix A.1.
	in:    `{ "foo": "bar"}`,
	patch: `[{ "op": "add", "path": "/baz", "value": "qux" }]`,
	want:  `{ "foo": "bar","baz":"qux"}`,
}, {
	// RFC 6902, appendix A.2.
	in:    `{ "foo": [ "bar", "baz" ] }`,
	patch: `[{ "op": "add", "path": "/foo/1", "value": "qux" }]`,
	want:  `{ "foo": [ "bar","qux", "baz" ] }`,
}, {
	// RFC 6902, appendix A.3.
	in: `{
	"baz": "qux",
	"foo": "bar"
}`,
	patch: `[{ "op": "remove", "path": "/baz" }]`,
	want: `{
	"foo": "bar"
}`,
}, {
	// RFC 6902, appendix A.4.
	in:    `{ "foo": [ "bar", "qux", "baz" ] }`,
	patch: `[{ "op": "remove", "path": "/foo/1" }]`,
	want:  `{ "foo": [ "bar", "baz" ] }`,
}, {
	// RFC 6902, appendix A.5.
	in: `{
	"baz": "qux",
	"foo": "bar"
}`,
	patch: `[{ "op": "replace", "path": "/baz", "value": "boo" }]`,
	want: `{
	"baz": "boo",
	"foo": "bar"
}`,
}, {
	// RFC 6902, appendix A.6.
	in: `{
	"foo": {
		"bar": "baz",
		"waldo": "fred"
	},
	"qux": {
		"corge": "grault"
	}
}`,
	patch: `[{ "op": "move", "from": "/foo/waldo", "path": "/qux/thud" }]`,
	want: `{
	"foo": {
		"bar": "baz"
	},
	"qux": {
		"corge": "grault","thud":"fred"
	}
}`,
}, {
	// RFC 6902, appendix A.7.
	in:    `{ "foo": [ "all", "grass", "cows", "eat" ] }`,
	patch: `[{ "op": "move", "from": "/foo/1", "path": "/foo/3" }]`,
	want:  `{ "foo": [ "all", "cows", "eat","grass" ] }`,
}, {
	// RFC 6902, appendix A.8.
	in: `{ "baz": "qux", "foo": [ "a", 2, "c" ] }`,
	patch: `[
		{ "op": "test", "path": "/baz", "value": "qux" },
		{ "op": "test", "path": "/foo/1", "value": 2 }
	]`,
}, {
	// RFC 6902, appendix A.9.
	in:      `{ "baz": "qux" }`,
	patch:   `[{ "op": "test", "path": "/baz", "value": "bar" }]`,
	wantErr: errors.New(`hujson: patch operation 0: values differ at "/baz"`),
}, {
	// RFC 6902, appendix A.10.
	in:    `{ "foo": "bar" }`,
	patch: `[{ "op": "add", "path": "/child", "value": { "grandchild": { } } }]`,
	want:  `{ "foo": "bar","child":{ "grandchild": { } } }`,
}, {
	// RFC 6902, appendix A.11.
	in:    `{ "foo": "bar" }`,
	patch: `[{ "op": "add", "path": "/baz", "value": "qux", "xyz": 123 }]`,
	want:  `{ "foo": "bar","baz":"qux" }`,
}, {
	// RFC 6902, appendix A.12.
	in:      `{ "foo": "bar" }`,
	patch:   `[{ "op": "add", "path": "/baz/bat", "value": "qux" }]`,
	want:    `{ "foo": "bar" }`,
	wantErr: errors.New("hujson: patch operation 0: value not found"),
}, {
	// RFC 6902, appendix A.13.
	in:      `null`,
	patch:   `[{ "op": "add", "path": "/baz", "value": "qux", "op": "remove" }]`,
	want:    `null`,
	wantErr: errors.New(`hujson: patch operation 0: duplicate name "op"`),
}, {
	// RFC 6902, appendix A.14.
	in:    `{ "/": 9, "~1": 10 }`,
	patch: `[{"op": "test", "path": "/~01", "value": 10}]`,
}, {
	// RFC 6902, appendix A.15.
	in:      `{ "/": 9, "~1": 10 }`,
	patch:   `[{ "op": "test", "path": "/~01", "value": "10" }]`,
	wantErr: errors.New(`hujson: patch operation 0: values differ at "/~01"`),
}, {
	// RFC 6902, appendix A.16.
	in:    `{ "foo": ["bar"] }`,
	patch: `[{ "op": "add", "path": "/foo/-", "value": ["abc", "def"] }]`,
	want:  `{ "foo": ["bar",["abc", "def"]] }`,
}, {
	// Test operation should be agnostic of object ordering and string escaping.
	in: `{
	"fizz": "buzz",
	"foo": "bar"
}`,
	patch: `[{ "op": "test", "path": "", "value": {"foo":"bar","\u0066izz":"buzz"} }]`,
}, {
	in:    `"hello"`,
	patch: `[{ "op": "add", "path": "", "value": "goodbye" }]`,
	want:  `"goodbye"`,
}, {
	in:      `"hello"`,
	patch:   `[{ "op": "remove", "path": "" }]`,
	wantErr: errors.New(`hujson: patch operation 0: cannot remove root value`),
}, {
	in:      `{}`,
	patch:   `[{ "op": "remove", "path": "/noexist" }]`,
	wantErr: errors.New(`hujson: patch operation 0: value not found`),
}, {
	in:    `{"hello":"goodbye","fizz":"buzz"}`,
	patch: `[{ "op": "add", "path": "/hello", "value": "bonjour" }]`,
	want:  `{"hello":"bonjour","fizz":"buzz"}`,
}, {
	in:    `{"hello":"goodbye","fizz":"buzz"}`,
	patch: `[{ "op": "move", "from": "/fizz", "path": "" }]`,
	want:  `"buzz"`,
}, {
	in:      `{"hello":"goodbye","fizz":"buzz"}`,
	patch:   `[{ "op": "move", "from": "", "path": "/fizz" }]`,
	wantErr: errors.New(`hujson: patch operation 0: cannot move "" into "/fizz"`),
}, {
	in:      `{"fizz":["buzz","wuzz"],"fizzy":"wizzy"}`,
	patch:   `[{ "op": "move", "from": "/fizz", "path": "/fizz" }]`,
	wantErr: errors.New(`hujson: patch operation 0: cannot move "/fizz" into "/fizz"`),
}, {
	in:      `{"fizz":["buzz","wuzz"],"fizzy":"wizzy"}`,
	patch:   `[{ "op": "move", "from": "/fizz", "path": "/fizz/wuzz" }]`,
	wantErr: errors.New(`hujson: patch operation 0: cannot move "/fizz" into "/fizz/wuzz"`),
}, {
	in:    `{"fizz":["buzz","wuzz"],"fizzy":"wizzy"}`,
	patch: `[{ "op": "move", "from": "/fizz", "path": "/fizzy" }]`,
	want:  `{"fizzy":["buzz","wuzz"]}`,
}, {
	in:      `{"fizz":["buzz","wuzz"],"fizzy":"wizzy"}`,
	patch:   `[{ "op": "move", "from": "/noexist", "path": "/fizzy" }]`,
	wantErr: errors.New(`hujson: patch operation 0: value not found`),
}, {
	in:      `{"fizz":["buzz","wuzz"],"fizzy":"wizzy"}`,
	patch:   `[{ "op": "test", "path": "/noexist", "value": null }]`,
	wantErr: errors.New(`hujson: patch operation 0: value not found`),
}, {
	in:      `{}`,
	patch:   `[{`,
	wantErr: fmt.Errorf(`hujson: line 1, column 3: %w`, fmt.Errorf("parsing value: %w", io.ErrUnexpectedEOF)),
}, {
	in:      `{}`,
	patch:   `{}`,
	wantErr: errors.New(`hujson: patch must be a JSON array`),
}, {
	in:      `{}`,
	patch:   `[[]]`,
	wantErr: errors.New(`hujson: patch operation 0: must be a JSON object`),
}, {
	in:      `{}`,
	patch:   `[{"op":null}]`,
	wantErr: errors.New(`hujson: patch operation 0: member "op" must be a JSON string`),
}, {
	in:      `{}`,
	patch:   `[{"op":"Move"}]`,
	wantErr: errors.New(`hujson: patch operation 0: unknown operation "Move"`),
}, {
	in:      `{}`,
	patch:   `[{"op":"move","path":null}]`,
	wantErr: errors.New(`hujson: patch operation 0: member "path" must be a JSON string`),
}, {
	in:      `{}`,
	patch:   `[{"op":"move","from":null}]`,
	wantErr: errors.New(`hujson: patch operation 0: member "from" must be a JSON string`),
}, {
	in:      `{}`,
	patch:   `[{}]`,
	wantErr: errors.New(`hujson: patch operation 0: missing required member "op"`),
}, {
	in:      `{}`,
	patch:   `[{"op":"move"}]`,
	wantErr: errors.New(`hujson: patch operation 0: missing required member "path"`),
}, {
	in:      `{}`,
	patch:   `[{"op":"move","path":""}]`,
	wantErr: errors.New(`hujson: patch operation 0: missing required member "from"`),
}, {
	in:      `{}`,
	patch:   `[{"op":"add","path":""}]`,
	wantErr: errors.New(`hujson: patch operation 0: missing required member "value"`),
}, {
	in:      `{"~1":0}`,
	patch:   `[{"op":"test","path":""}]`,
	wantErr: errors.New(`hujson: patch operation 0: missing required member "value"`),
}, {
	in:      "{}",
	patch:   `[{"op":"move","from":"","path":"z"}]`,
	wantErr: errors.New(`hujson: patch operation 0: cannot move "" into "z"`),
}, {
	in:      "{}",
	patch:   `[{"op":"copy","from":"","path":"/noexist"}]`,
	wantErr: errors.New(`hujson: patch operation 0: cannot copy "" into "/noexist"`),
}, {
	in:      `"` + "\xff" + `"`,
	patch:   `[{ "op": "test", "path": "", "value": "` + "\ufffd" + `" }]`,
	wantErr: nil, // TODO(dsnet): Should fail due to invalid UTF-8.
}, {
	in:      `9223372036854775800`,
	patch:   `[{ "op": "test", "path": "", "value": 9223372036854775801 }]`,
	wantErr: nil, // TODO(dsnet): Should fail under precise integer equality.
}, {
	in:    `1e1000`,
	patch: `[{ "op": "test", "path": "", "value": 1e1000 }]`,
	// TODO(dsnet): Should pass under comparison of closest floating-point values.
	wantErr: errors.New(`hujson: patch operation 0: values differ at ""`),
}, {
	in:      `{ "dupe": "foo", "dupe": "bar" }`,
	patch:   `[{ "op": "test", "path": "", "value": { "dupe": "bar" } }]`,
	wantErr: nil, // TODO(dsnet): Should fail because of duplicate members.
}, {
	in: `{
	"name1": "value",
	// Comment1
	
	// Comment2

	// Comment3
	"name2": "value", // Comment4
	// Comment5
	
	// Comment6
	
	// Comment7
	"name3": "value",
}`,
	patch: `[{ "op": "remove", "path": "/name2" }]`,
	want: `{
	"name1": "value",
	// Comment1
	
	// Comment2

	// Comment6
	
	// Comment7
	"name3": "value",
}`,
}, {
	in: `[
	"value1",
	// Comment1
	
	// Comment2

	// Comment3
	"value2", // Comment4
	// Comment5
	
	// Comment6
	
	// Comment7
	"value3",
]`,
	patch: `[{ "op": "remove", "path": "/1" }]`,
	want: `[
	"value1",
	// Comment1
	
	// Comment2

	// Comment6
	
	// Comment7
	"value3",
]`,
}, {
	in: `{}`,
	patch: `[
	{ "op": "add", "path": "/name1",
	// Comment1

	// Comment2

	// Comment3
	"value": "value", // Comment4
},
	{ "op": "copy", "from": "/name1", "path": "/name2" },
	{ "op": "copy", "from": "/name2", "path": "/name3" },
]`,
	want: `{
// Comment3
	"name1":"value", // Comment4
// Comment3
	"name2":"value", // Comment4
// Comment3
	"name3":"value" // Comment4
}`,
}, {
	in: `[]`,
	patch: `[
	{ "op": "add", "path": "/0",
	// Comment1

	// Comment2

	// Comment3
	"value": "value", // Comment4
},
	{ "op": "copy", "from": "/0", "path": "/1" },
	{ "op": "copy", "from": "/1", "path": "/2" },
]`,
	want: `[
// Comment3
	"value", // Comment4
// Comment3
	"value", // Comment4
// Comment3
	"value" // Comment4
]`,
}, {
	in: `{
	// Comment3
	"name1":"value", // Comment4
	// Comment3
	"name2":"value", // Comment4
	// Comment3
	"name3":"value" // Comment4
}`,
	patch: `[
	{ "op": "remove", "path": "/name2" },
]`,
	want: `{
	// Comment3
	"name1":"value", // Comment4
	// Comment3
	"name3":"value" // Comment4
}`,
}, {
	in: `[
	// Comment3
	"value1", // Comment4
	// Comment3
	"value2", // Comment4
	// Comment3
	"value3" // Comment4
]`,
	patch: `[
	{ "op": "remove", "path": "/1" },
]`,
	want: `[
	// Comment3
	"value1", // Comment4
	// Comment3
	"value3" // Comment4
]`,
}, {
	in: `{
	// Comment3
	"name1":"value", // Comment4

	// Comment3
	"name2":"value", // Comment4

	// Comment3
	"name3":"value" // Comment4
}`,
	patch: `[
	{ "op": "remove", "path": "/name2" },
]`,
	want: `{
	// Comment3
	"name1":"value", // Comment4

	// Comment3
	"name3":"value" // Comment4
}`,
}, {
	in: `{
	// Comment3
	"name1":"value", // Comment4

	// Comment3
	"name2":"value", // Comment4

	// Comment3
	"name3":"value" // Comment4
}`,
	patch: `[
	{ "op": "remove", "path": "/name2" },
]`,
	want: `{
	// Comment3
	"name1":"value", // Comment4

	// Comment3
	"name3":"value" // Comment4
}`,
}, {
	in: `[
	// Comment3
	"value1", // Comment4

	// Comment3
	"value2", // Comment4

	// Comment3
	"value3" // Comment4
]`,
	patch: `[
	{ "op": "remove", "path": "/1" },
]`,
	want: `[
	// Comment3
	"value1", // Comment4

	// Comment3
	"value3" // Comment4
]`,
}, {
	in: `{
	// Comment3
	"name1":"value", // Comment4

	// Comment3
	"name2":"value", // Comment4

	// Comment3
	"name3":"value" // Comment4
}`,
	patch: `[
	{ "op": "replace", "path": "/name2","value":"VALUE"},
]`,
	want: `{
	// Comment3
	"name1":"value", // Comment4

	// Comment3
	"name2":"VALUE", // Comment4

	// Comment3
	"name3":"value" // Comment4
}`,
}, {
	in: `[
	// Comment3
	"value1", // Comment4

	// Comment3
	"value2", // Comment4

	// Comment3
	"value3" // Comment4
]`,
	patch: `[
	{ "op": "replace", "path": "/1","value":"VALUE"},
]`,
	want: `[
	// Comment3
	"value1", // Comment4

	// Comment3
	"VALUE", // Comment4

	// Comment3
	"value3" // Comment4
]`,
}}

func TestPatch(t *testing.T) {
	for _, tt := range testdataPatch {
		t.Run("", func(t *testing.T) {
			v, err := Parse([]byte(tt.in))
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			gotErr := v.Patch([]byte(tt.patch))
			if !reflect.DeepEqual(gotErr, tt.wantErr) {
				t.Errorf("Patch error mismatch:\ngot  %v\nwant %v", gotErr, tt.wantErr)
			}
			got := v.String()
			if diff := cmp.Diff(tt.want, got); diff != "" && tt.want != "" {
				t.Errorf("Patch mismatch (-want +got):\n%s\n\ngot:\n%s\n\nwant:\n%s", diff, got, tt.want)
			}
		})
	}
}
