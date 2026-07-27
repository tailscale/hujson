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
	"github.com/google/go-cmp/cmp/cmpopts"
)

var testdata = []struct {
	in      string
	want    Value
	wantErr error

	wantMin string
	wantStd string
}{{
	in: ` null `,
	want: Value{
		BeforeExtra: Extra(" "),
		StartOffset: 1,
		Value:       Literal("null"),
		EndOffset:   5,
		AfterExtra:  Extra(" "),
	},
	wantMin: `null`,
	wantStd: ` null `,
}, {
	in: ` null,`,
	want: Value{
		BeforeExtra: Extra(" "),
		StartOffset: 1,
		Value:       Literal("null"),
		EndOffset:   5,
	},
	wantErr: fmt.Errorf("hujson: line 1, column 6: %w", errors.New("invalid character ',' after top-level value")),
}, {
	in: "//😊 \r\t\n/*\r\t\n*/null//😊 \r\t\n/*\r\t\n*/",
	want: Value{
		BeforeExtra: Extra("//😊 \r\t\n/*\r\t\n*/"),
		StartOffset: 17,
		Value:       Literal("null"),
		EndOffset:   21,
		AfterExtra:  Extra("//😊 \r\t\n/*\r\t\n*/"),
	},
	wantMin: "null",
	wantStd: "       \r\t\n  \r\t\n  null       \r\t\n  \r\t\n  ",
}, {
	in:      "/?",
	wantErr: fmt.Errorf("hujson: line 1, column 1: %w", errors.New("invalid character '/' at start of value")),
}, {
	in:      "//\xde\xad\xbe\xef\nnull",
	wantErr: fmt.Errorf("hujson: line 1, column 1: %w", errors.New("invalid UTF-8 in comment")),
}, {
	in: "null//",
	want: Value{
		Value:     Literal("null"),
		EndOffset: 4,
	},
	wantErr: fmt.Errorf("hujson: line 1, column 5: %w", fmt.Errorf("parsing comment: %w", io.ErrUnexpectedEOF)),
}, {
	in: "null//\n",
	want: Value{
		Value:      Literal("null"),
		EndOffset:  4,
		AfterExtra: Extra("//\n"),
	},
	wantMin: "null",
	wantStd: "null  \n",
}, {
	in:      `"\"\\\u0022😊`,
	wantErr: fmt.Errorf("hujson: line 1, column 16: %w", fmt.Errorf("parsing string: %w", io.ErrUnexpectedEOF)),
}, {
	in:      `"\xff"`,
	wantErr: fmt.Errorf("hujson: line 1, column 1: %w", errors.New("invalid literal: \"\\xff\"")),
}, {
	in:      `"\"\\\u0022😊"`,
	want:    Value{Value: Literal(`"\"\\\u0022😊"`), EndOffset: 16},
	wantMin: `"\"\\\u0022😊"`,
	wantStd: `"\"\\\u0022😊"`,
}, {
	in:      `3.14159E+435`,
	want:    Value{Value: Literal(`3.14159E+435`), EndOffset: 12},
	wantMin: `3.14159E+435`,
	wantStd: `3.14159E+435`,
}, {
	in:      `+1000`,
	wantErr: fmt.Errorf("hujson: line 1, column 1: %w", errors.New("invalid literal: +1000")),
}, {
	in:      "{",
	want:    Value{Value: &Object{}},
	wantErr: fmt.Errorf("hujson: line 1, column 2: %w", fmt.Errorf("parsing value: %w", io.ErrUnexpectedEOF)),
}, {
	in:      "{,}",
	want:    Value{Value: &Object{}},
	wantErr: fmt.Errorf("hujson: line 1, column 2: %w", errors.New("invalid character ',' at start of value")),
}, {
	in:      `{null:"v"`,
	want:    Value{Value: &Object{}},
	wantErr: fmt.Errorf("hujson: line 1, column 2: %w", errors.New("invalid character 'n' at start of object name")),
}, {
	in:      `{"k"`,
	want:    Value{Value: &Object{}},
	wantErr: fmt.Errorf("hujson: line 1, column 5: %w", fmt.Errorf("parsing object after name: %w", io.ErrUnexpectedEOF)),
}, {
	in:      `{"k";`,
	want:    Value{Value: &Object{}},
	wantErr: fmt.Errorf("hujson: line 1, column 5: %w", errors.New("invalid character ';' after object name")),
}, {
	in:      `{"k":}`,
	want:    Value{Value: &Object{}},
	wantErr: fmt.Errorf("hujson: line 1, column 6: %w", errors.New("invalid character '}' at start of value")),
}, {
	in: `{"k":"v"`,
	want: Value{Value: &Object{
		Members: []ObjectMember{{
			Value{StartOffset: 1, Value: Literal(`"k"`), EndOffset: 4},
			Value{StartOffset: 5, Value: Literal(`"v"`), EndOffset: 8},
		}},
	}},
	wantErr: fmt.Errorf("hujson: line 1, column 9: %w", fmt.Errorf("parsing object after value: %w", io.ErrUnexpectedEOF)),
}, {
	in: `{"k":"v"]`,
	want: Value{Value: &Object{
		Members: []ObjectMember{{
			Value{StartOffset: 1, Value: Literal(`"k"`), EndOffset: 4},
			Value{StartOffset: 5, Value: Literal(`"v"`), EndOffset: 8},
		}},
	}},
	wantErr: fmt.Errorf("hujson: line 1, column 9: %w", errors.New("invalid character ']' after object value (expecting ',' or '}')")),
}, {
	in: ` { "k" : "v" } `,
	want: Value{
		BeforeExtra: Extra(" "),
		StartOffset: 1,
		Value: &Object{
			Members: []ObjectMember{{
				Value{BeforeExtra: Extra(" "), StartOffset: 3, Value: Literal(`"k"`), EndOffset: 6, AfterExtra: Extra(" ")},
				Value{BeforeExtra: Extra(" "), StartOffset: 9, Value: Literal(`"v"`), EndOffset: 12},
			}},
			AfterExtra: Extra(" "),
		},
		EndOffset:  14,
		AfterExtra: Extra(" "),
	},
	wantMin: `{"k":"v"}`,
	wantStd: ` { "k" : "v" } `,
}, {
	in: ` { "k" : "v" , } `,
	want: Value{
		BeforeExtra: Extra(" "),
		StartOffset: 1,
		Value: &Object{
			Members: []ObjectMember{{
				Value{BeforeExtra: Extra(" "), StartOffset: 3, Value: Literal(`"k"`), EndOffset: 6, AfterExtra: Extra(" ")},
				Value{BeforeExtra: Extra(" "), StartOffset: 9, Value: Literal(`"v"`), EndOffset: 12, AfterExtra: Extra(" ")},
			}},
			AfterExtra: Extra(" "),
		},
		EndOffset:  16,
		AfterExtra: Extra(" "),
	},
	wantMin: `{"k":"v"}`,
	wantStd: ` { "k" : "v"   } `,
}, {
	in:      "[",
	want:    Value{Value: &Array{}},
	wantErr: fmt.Errorf("hujson: line 1, column 2: %w", fmt.Errorf("parsing value: %w", io.ErrUnexpectedEOF)),
}, {
	in:      "[,]",
	want:    Value{Value: &Array{}},
	wantErr: fmt.Errorf("hujson: line 1, column 2: %w", errors.New("invalid character ',' at start of value")),
}, {
	in: `["s"`,
	want: Value{Value: &Array{
		Elements: []Value{{StartOffset: 1, Value: Literal(`"s"`), EndOffset: 4}},
	}},
	wantErr: fmt.Errorf("hujson: line 1, column 5: %w", fmt.Errorf("parsing array after value: %w", io.ErrUnexpectedEOF)),
}, {
	in: `["s"}`,
	want: Value{Value: &Array{
		Elements: []Value{{StartOffset: 1, Value: Literal(`"s"`), EndOffset: 4}},
	}},
	wantErr: fmt.Errorf("hujson: line 1, column 5: %w", errors.New("invalid character '}' after array value (expecting ',' or ']')")),
}, {
	in: ` [ "s" ] `,
	want: Value{
		BeforeExtra: Extra(" "),
		StartOffset: 1,
		Value: &Array{
			Elements:   []Value{{BeforeExtra: Extra(" "), StartOffset: 3, Value: Literal(`"s"`), EndOffset: 6}},
			AfterExtra: Extra(" "),
		},
		EndOffset:  8,
		AfterExtra: Extra(" "),
	},
	wantMin: `["s"]`,
	wantStd: ` [ "s" ] `,
}, {
	in: ` [ "s" , ] `,
	want: Value{
		BeforeExtra: Extra(" "),
		StartOffset: 1,
		Value: &Array{
			Elements:   []Value{{BeforeExtra: Extra(" "), StartOffset: 3, Value: Literal(`"s"`), EndOffset: 6, AfterExtra: Extra(" ")}},
			AfterExtra: Extra(" "),
		},
		EndOffset:  10,
		AfterExtra: Extra(" "),
	},
	wantMin: `["s"]`,
	wantStd: ` [ "s"   ] `,
}, {
	in: ` /**/ [ /**/ null /**/ , /**/ false /**/ , /**/ true /**/ , /**/ "string" /**/ , /**/ 0 /**/ , /**/ {} /**/ , /**/ [] /**/ ] /**/ `,
	want: Value{
		BeforeExtra: Extra(" /**/ "),
		StartOffset: 6,
		Value: &Array{
			Elements: []Value{
				{BeforeExtra: Extra(" /**/ "), StartOffset: 13, Value: Literal("null"), EndOffset: 17, AfterExtra: Extra(" /**/ ")},
				{BeforeExtra: Extra(" /**/ "), StartOffset: 30, Value: Literal("false"), EndOffset: 35, AfterExtra: Extra(" /**/ ")},
				{BeforeExtra: Extra(" /**/ "), StartOffset: 48, Value: Literal("true"), EndOffset: 52, AfterExtra: Extra(" /**/ ")},
				{BeforeExtra: Extra(" /**/ "), StartOffset: 65, Value: Literal(`"string"`), EndOffset: 73, AfterExtra: Extra(" /**/ ")},
				{BeforeExtra: Extra(" /**/ "), StartOffset: 86, Value: Literal("0"), EndOffset: 87, AfterExtra: Extra(" /**/ ")},
				{BeforeExtra: Extra(" /**/ "), StartOffset: 100, Value: &Object{}, EndOffset: 102, AfterExtra: Extra(" /**/ ")},
				{BeforeExtra: Extra(" /**/ "), StartOffset: 115, Value: &Array{}, EndOffset: 117},
			},
			AfterExtra: Extra(" /**/ "),
		},
		EndOffset:  124,
		AfterExtra: Extra(" /**/ "),
	},
	wantMin: `[null,false,true,"string",0,{},[]]`,
	wantStd: `      [      null      ,      false      ,      true      ,      "string"      ,      0      ,      {}      ,      []      ]      `,
}, {
	in: " \xff",
	want: Value{
		BeforeExtra: Extra(" "),
		StartOffset: 1,
	},
	wantErr: fmt.Errorf("hujson: line 1, column 2: %w", errors.New("invalid character '\\xff' at start of value")),
}, {
	in: " '",
	want: Value{
		BeforeExtra: Extra(" "),
		StartOffset: 1,
	},
	wantErr: fmt.Errorf("hujson: line 1, column 2: %w", errors.New("invalid character '\\'' at start of value")),
}, {
	in: " 💩",
	want: Value{
		BeforeExtra: Extra(" "),
		StartOffset: 1,
	},
	wantErr: fmt.Errorf("hujson: line 1, column 2: %w", errors.New("invalid character '💩' at start of value")),
}, {
	in: " \uffff",
	want: Value{
		BeforeExtra: Extra(" "),
		StartOffset: 1,
	},
	wantErr: fmt.Errorf("hujson: line 1, column 2: %w", errors.New("invalid character '\\uffff' at start of value")),
}, {
	in: " \U00101234",
	want: Value{
		BeforeExtra: Extra(" "),
		StartOffset: 1,
	},
	wantErr: fmt.Errorf("hujson: line 1, column 2: %w", errors.New("invalid character '\\U00101234' at start of value")),
}}

func Test(t *testing.T) {
	for _, tt := range testdata {
		gotVal, gotErr := Parse([]byte(tt.in))
		if diff := cmp.Diff(tt.want, gotVal, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("Parse mismatch (-want +got):\n%s", diff)
		}
		if !reflect.DeepEqual(gotErr, tt.wantErr) {
			t.Errorf("Parse error mismatch:\ngot  %v\nwant %v", gotErr, tt.wantErr)
		}

		if gotErr == nil {
			gotIsStd := gotVal.IsStandard()
			wantIsStd := tt.in == tt.wantStd
			if gotIsStd != wantIsStd {
				t.Errorf("IsStandard() = %v, want %v", gotIsStd, wantIsStd)
			}

			gotBuf := string(gotVal.Pack())
			if diff := cmp.Diff(gotBuf, tt.in); diff != "" {
				t.Errorf("Pack mismatch (-want +got):\n%s", diff)
			}

			if tt.wantMin != "" {
				gotMinVal := gotVal.Clone()
				gotMinVal.Minimize()
				gotMinBuf := string(gotMinVal.Pack())
				wantMinVal, _ := Parse([]byte(tt.wantMin))
				if diff := cmp.Diff(wantMinVal, gotMinVal, cmpopts.EquateEmpty()); diff != "" {
					t.Errorf("Minimize Value mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(tt.wantMin, gotMinBuf); diff != "" {
					t.Errorf("Minimize buffer mismatch (-want +got):\n%s", diff)
				}
				if !gotMinVal.IsStandard() {
					t.Errorf("IsStandard() = false, want true")
				}
			}

			if tt.wantStd != "" {
				gotStdVal := gotVal.Clone()
				gotStdVal.Standardize()
				gotStdBuf := string(gotStdVal.Pack())
				wantStdVal, _ := Parse([]byte(tt.wantStd))
				if diff := cmp.Diff(wantStdVal, gotStdVal, cmpopts.EquateEmpty()); diff != "" {
					t.Errorf("Standardize Value mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(tt.wantStd, gotStdBuf); diff != "" {
					t.Errorf("Standardize buffer mismatch (-want +got):\n%s", diff)
				}
				if !gotStdVal.IsStandard() {
					t.Errorf("IsStandard() = false, want true")
				}
			}
		}
	}
}

func TestLineSeparators(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantErr string
	}{
		{
			name: "U+2028 in string",
			in:   "\"a\u2028b\"",
		},
		{
			name: "U+2029 in string",
			in:   "\"a\u2029b\"",
		},
		{
			name: "U+2028 in block comment",
			in:   "/* a\u2028b */ null",
		},
		{
			name: "U+2029 in block comment",
			in:   "/* a\u2029b */ null",
		},
		{
			name:    "U+2028 as whitespace",
			in:      "null\u2028",
			wantErr: "hujson: line 1, column 5: invalid character '\\u2028' after top-level value",
		},
		{
			name:    "U+2029 as whitespace",
			in:      "null\u2029",
			wantErr: "hujson: line 1, column 5: invalid character '\\u2029' after top-level value",
		},
		{
			name:    "U+2028 in line comment",
			in:      "// hidden\u2028null\ntrue",
			wantErr: "hujson: line 1, column 10: invalid character '\\u2028' in line comment",
		},
		{
			name:    "U+2029 in line comment",
			in:      "// hidden\u2029null\ntrue",
			wantErr: "hujson: line 1, column 10: invalid character '\\u2029' in line comment",
		},
		{
			name:    "U+2028 in line comment at EOF",
			in:      "// hidden\u2028null",
			wantErr: "hujson: line 1, column 10: invalid character '\\u2028' in line comment",
		},
		{
			name:    "U+2029 in line comment at EOF",
			in:      "// hidden\u2029null",
			wantErr: "hujson: line 1, column 10: invalid character '\\u2029' in line comment",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.in))
			switch {
			case err == nil && tt.wantErr != "":
				t.Fatalf("Parse() error = nil, want %q", tt.wantErr)
			case err != nil && err.Error() != tt.wantErr:
				t.Fatalf("Parse() error = %q, want %q", err, tt.wantErr)
			}
		})
	}
}
