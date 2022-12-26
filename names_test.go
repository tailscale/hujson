// Copyright (c) 2021 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hujson

import (
	"fmt"
	"log"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// The "encoding/json" package unfortunately uses case-insensitive matching
// when unmarshaling. For example, the following:
//
//	{"NAME": ...}
//	{"nAmE": ...}
//	{"name": ...}
//	{"Name": ...}
//
// are all equivalent when unmarshaling into a Go struct like:
//
//	struct{ Name string }
//
// In order to conform some HuJSON value to consistently use the same set of
// JSON object names, a Names map can be derived from Go struct type
// and applied upon the HuJSON value using the Value.NormalizeNames method.
func ExampleValue_NormalizeNames() {
	type MyStruct struct {
		Alpha int
		Bravo []struct {
			Foo int
		} `json:"bravo_wavo"`
		Charlie map[string]struct {
			Fizz int `json:"fizzy_wizzy"`
			Buzz int `json:",omitempty"`
		}
		Ignored    int `json:"-"`
		unexported int
	}

	// Derive the set of canonical names from the Go struct type.
	names := NewNames(reflect.TypeOf(MyStruct{}))
	// Verify that the derived names match what we expect.
	gotNames := names
	wantNames := Names{
		"Alpha": nil, // name comes from Go struct field
		"bravo_wavo": { // name comes from `json` tag
			"Foo": nil, // name comes from Go struct field
		},
		"Charlie": { // name comes from Go struct field
			"*": { // implies that all JSON object members use the same set of sub-names
				"fizzy_wizzy": nil, // name comes from `json` tag
				"Buzz":        nil, // name comes from Go struct field
			},
		},
	}
	if diff := cmp.Diff(gotNames, wantNames); diff != "" {
		log.Fatalf("NewNames mismatch (-want +got):\n%s", diff)
	}

	// Parse some HuJSON input with strangely formatted names.
	v, err := Parse([]byte(`{
	"AlPhA": 0,
	"BRAVO_WAVO": [
		{"FOO": 0},
		{"fOo": 1},
		{"Foo": 2},
	],
	"charlie": {
		"kEy": {"FIZZY_WIZZY": 0},
		"KeY": {"bUzZ": 1},
	},
}`))
	if err != nil {
		log.Fatal(err)
	}
	// Conform JSON object names in the HuJSON value to the canonical names.
	v.NormalizeNames(gotNames)
	fmt.Println(v)

	// Output:
	// {
	// 	"Alpha": 0,
	// 	"bravo_wavo": [
	// 		{"Foo": 0},
	// 		{"Foo": 1},
	// 		{"Foo": 2},
	// 	],
	// 	"Charlie": {
	// 		"kEy": {"fizzy_wizzy": 0},
	// 		"KeY": {"Buzz": 1},
	// 	},
	// }
}

func TestNormalizeNames(t *testing.T) {
	type MyStruct struct {
		GoName   int
		JSONName int `json:"json_name"`
	}

	tests := []struct {
		typ       interface{}
		wantNames Names
		in        string
		wantOut   string
	}{{
		typ:       0,
		wantNames: nil,
		in:        `{"hello":"goodbye"}`,
		wantOut:   `{"hello":"goodbye"}`,
	}, {
		typ:       new(int),
		wantNames: nil,
		in:        `{"hello":"goodbye"}`,
		wantOut:   `{"hello":"goodbye"}`,
	}, {
		typ: struct {
			GoName1    int
			GoName2    int `json:",omitempty"`
			JSONName   int `json:"json_name"`
			Ignored    int `json:"-"`
			unexported int `json:"fake_name"`
		}{},
		wantNames: Names{"GoName1": nil, "GoName2": nil, "json_name": nil},
		in:        `{"goname1":0,"goname2":0,"JSON_NAME":0,"JSONNAME":0}`,
		wantOut:   `{"GoName1":0,"GoName2":0,"json_name":0,"JSONNAME":0}`,
	}, {
		typ: struct {
			M *[]map[int][]map[string][]struct {
				F int `json:"field"`
			}
		}{},
		wantNames: Names{"M": {"*": {"*": {"field": nil}}}},
		in:        `{"m":[{"hello":[{"goodbye":[{"FIELD":0}]}]}]}`,
		wantOut:   `{"M":[{"hello":[{"goodbye":[{"field":0}]}]}]}`,
	}, {
		typ: struct {
			M map[string]struct{}
		}{},
		wantNames: Names{"M": nil},
	}, {
		typ: struct {
			MyStruct
			int
		}{},
		wantNames: Names{"GoName": nil, "json_name": nil},
	}, {
		typ: struct {
			*MyStruct
			*int
		}{},
		wantNames: Names{"GoName": nil, "json_name": nil},
	}, {
		typ: struct {
			MyStruct `json:"my_struct"`
		}{},
		wantNames: Names{"my_struct": {"GoName": nil, "json_name": nil}},
	}}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			gotNames := NewNames(reflect.TypeOf(tt.typ))
			if diff := cmp.Diff(tt.wantNames, gotNames); diff != "" {
				t.Errorf("NewNames(%T) mismatch (-want +got):\n%s", tt.typ, diff)
			}

			if tt.in == "" {
				return
			}
			v, err := Parse([]byte(tt.in))
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			v.NormalizeNames(gotNames)
			gotOut := v.String()
			if diff := cmp.Diff(tt.wantOut, gotOut); diff != "" {
				t.Errorf("v.NormalizeNames(%T) mismatch (-want +got):\n%s", tt.typ, diff)
			}
		})
	}
}
