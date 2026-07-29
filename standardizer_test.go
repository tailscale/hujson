// Copyright (c) 2023 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hujson

import (
	"bytes"
	"io"
	"math/rand"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

const jwccTestdata = "/**/ [ /**/ { /**/ \"k\" /**/ : /**/ \"v\" /**/ , /*x*/ } /**/ , /**/ 0 /**/ , /*x*/ ] /**/"

var standardizerTestdata = []struct {
	in      string
	want    string
	wantErr error
}{
	{in: "", want: ""},
	{in: "/", want: "", wantErr: io.ErrUnexpectedEOF},
	{in: "/ ", want: "/ "},
	{in: "//", want: "  ", wantErr: io.ErrUnexpectedEOF},
	{in: "//\n", want: "  \n"},
	{in: "//*\n", want: "   \n"},
	{in: "/ ", want: "/ "},
	{in: " \n\r\t", want: " \n\r\t"},
	{in: "//x\xff\xffx\n", want: "   \xff\xff \n"},
	{in: "//💩\n"[:3], want: "  ", wantErr: io.ErrUnexpectedEOF},
	{in: "//💩\n"[:4], want: "  ", wantErr: io.ErrUnexpectedEOF},
	{in: "//💩\n"[:5], want: "  ", wantErr: io.ErrUnexpectedEOF},
	{in: "//💩\n"[:6], want: "      ", wantErr: io.ErrUnexpectedEOF},
	{in: "//💩\n"[:7], want: "      \n"},
	{in: "/", want: "", wantErr: io.ErrUnexpectedEOF},
	{in: "/*", want: "  ", wantErr: io.ErrUnexpectedEOF},
	{in: "/**", want: "  ", wantErr: io.ErrUnexpectedEOF},
	{in: "/**/", want: "    "},
	{in: "/***/", want: "     "},
	{in: "/****/", want: "      "},
	{in: "/**?", want: "    ", wantErr: io.ErrUnexpectedEOF},
	{in: "/*\n*/", want: "  \n  "},
	{in: "/*x\xff\xffx*/", want: "   \xff\xff   "},
	{in: "/*💩*/"[:3], want: "  ", wantErr: io.ErrUnexpectedEOF},
	{in: "/*💩*/"[:4], want: "  ", wantErr: io.ErrUnexpectedEOF},
	{in: "/*💩*/"[:5], want: "  ", wantErr: io.ErrUnexpectedEOF},
	{in: "/*💩*/"[:6], want: "      ", wantErr: io.ErrUnexpectedEOF},
	{in: "/*💩*/"[:7], want: "      ", wantErr: io.ErrUnexpectedEOF},
	{in: "/*💩*/"[:8], want: "        "},
	{in: `"`, want: `"`},
	{in: `""`, want: `""`},
	{in: `"\""`, want: `"\""`},
	{in: `"\""//` + "\n", want: `"\""  ` + "\n"},
	{in: `"\"/**/"`, want: `"\"/**/"`},
	{in: ",", want: ","},
	{in: ",]", want: ",]"},
	{in: "[,", want: "[,"},
	{in: "[,]", want: "[,]"},
	{in: "[a,", want: "[a", wantErr: io.ErrUnexpectedEOF},
	{in: "[a,]", want: "[a ]"},
	{in: "[{},", want: "[{}", wantErr: io.ErrUnexpectedEOF},
	{in: "[{},]", want: "[{} ]"},
	{in: `[["",],{},]`, want: `[["" ],{} ]`},
	{in: "{\"hello\":\"goodbye\", /*\nfizz\n*/ // buzz\n }", want: "{\"hello\":\"goodbye\"    \n    \n          \n }"},
	{in: jwccTestdata, want: strings.ReplaceAll(strings.ReplaceAll(jwccTestdata, "/**/", "    "), ", /*x*/", "       ")},
}

func TestStandardizer(t *testing.T) {
	for _, tt := range standardizerTestdata {
		switch got, gotErr := io.ReadAll(NewStandardizer(strings.NewReader(tt.in))); {
		case string(got) != tt.want:
			t.Errorf("Standardize mismatch (-got +want):\n%s", cmp.Diff(string(got), tt.want))
		case gotErr != tt.wantErr:
			t.Errorf("Standardize error = %v, want %v", gotErr, tt.wantErr)
		}
	}
}

func BenchmarkStandardize(b *testing.B) {
	in := []byte(jwccTestdata)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := Standardize(in); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStandardizer(b *testing.B) {
	in := []byte(jwccTestdata)
	out := make([]byte, len(in))
	b.ReportAllocs()
	var br bytes.Reader
	var sb Standardizer
	for i := 0; i < b.N; i++ {
		br.Reset(in)
		sb.Reset(&br)
		if _, err := io.ReadFull(&sb, out); err != nil {
			b.Fatal(err)
		}
	}
}

func FuzzStandardizer(f *testing.F) {
	for _, tt := range standardizerTestdata {
		f.Add(int64(0), tt.in)
	}
	f.Fuzz(func(t *testing.T, seed int64, in string) {
		rn := rand.New(rand.NewSource(seed))
		inner := &randomReader{R: strings.NewReader(in), RN: rn}
		outer := &randomReader{R: NewStandardizer(inner), RN: rn}
		got, gotErr := io.ReadAll(outer)
		if gotErr == nil {
			_, gotErr = Parse(got)
		}
		want, wantErr := Standardize([]byte(in))
		if (gotErr == nil) != (wantErr == nil) {
			t.Errorf("error mismatch: got %v, want %v", gotErr, wantErr)
		}
		if gotErr == nil && string(got) != string(want) {
			t.Errorf("standardize mismatch (-got +want):\n%s", cmp.Diff(string(got), string(want)))
		}
	})
}

type randomReader struct {
	R  io.Reader
	RN *rand.Rand
}

func (r randomReader) Read(b []byte) (int, error) {
	n, err := r.R.Read(b[:r.RN.Intn(len(b)+1)])
	if err == io.EOF && r.RN.Intn(2) == 0 {
		err = nil
	}
	return n, err
}
