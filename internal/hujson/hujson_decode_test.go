package hujson

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

var hujsonDecodeTests = []unmarshalTest{
	{ptr: new(int), in: "// comment\n7", out: 7},
	{ptr: new(T), in: `// leading comment
		{"X": "xval"}
		// trailing comment`, out: T{X: "xval"}},
	{ptr: new(T), in: `{
		"X": "xval" // trailing-line comment
		}`, out: T{X: "xval"}},
	{ptr: new(T), in: `{
		"Y": 7,
		/* multi-line comment
		"Y": 8,
		*/
		"X": /* comment between field name and value */ "x"
	}`, disallowUnknownFields: false, out: T{Y: 7, X: "x"}},
	{ptr: new(T), in: "{\n\"X\": \"x\",\n}", out: T{X: "x"}}, // trailing comma
	{ptr: new([1]int), in: "[1, \n]", out: [1]int{1}},        // trailing comma
}

func TestHuDecode(t *testing.T) {
	for _, tt := range hujsonDecodeTests {
		t.Run(tt.in, func(t *testing.T) {
			in := []byte(tt.in)
			v := reflect.New(reflect.TypeOf(tt.ptr).Elem())
			dec := NewDecoder(bytes.NewReader(in))
			if tt.disallowUnknownFields {
				dec.DisallowUnknownFields()
			}
			if err := dec.Decode(v.Interface()); !equalError(err, tt.err) {
				t.Errorf("err=%v, want %v", err, tt.err)
				return
			} else if err != nil {
				return
			}
			if !reflect.DeepEqual(v.Elem().Interface(), tt.out) {
				t.Errorf("mismatch\nhave: %#+v\nwant: %#+v", v.Elem().Interface(), tt.out)
			}
		})
	}
}

type ACL struct {
	Action     string
	ByteOffset int64 `hujson:",inputoffset"`
}

type ACLFile struct {
	ACLs []ACL
}

func TestHuInputOffset(t *testing.T) {
	var a ACLFile
	err := NewDecoder(strings.NewReader(`{"ACLs": [
    {"Action": "foo"},
    {"Action": "bar"}, {"Action": "baz"},
]}`)).Decode(&a)
	if err != nil {
		t.Fatal(err)
	}
	want := ACLFile{ACLs: []ACL{
		{Action: "foo", ByteOffset: 15},
		{Action: "bar", ByteOffset: 38},
		{Action: "baz", ByteOffset: 57},
	}}
	if !reflect.DeepEqual(a, want) {
		t.Errorf("mismatch\n got: %+v\nwant: %+v\n", a, want)
	}
}
