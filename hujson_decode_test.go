package hujson

import (
	"bytes"
	"reflect"
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
