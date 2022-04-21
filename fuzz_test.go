// Copyright (c) 2021 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dev.fuzz
// +build dev.fuzz

package hujson

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Fuzz(f *testing.F) {
	for _, tt := range testdata {
		f.Add([]byte(tt.in))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) > 1<<12 {
			t.Skip("input too large")
		}

		// Parse for valid HuJSON input.
		v, err := Parse(b)
		if err != nil {
			t.Skipf("input %q: Parse error: %v", b, err)
		}

		// Pack should preserve the original input exactly.
		if b1 := v.Pack(); !bytes.Equal(b, b1) {
			t.Fatalf("input %q: Pack mismatch: %s", b, cmp.Diff(b, b1))
		}

		// Standardize should produce valid JSON.
		v2 := v.Clone()
		v2.Standardize()
		b2 := v2.Pack()
		if !json.Valid(b2) {
			t.Fatalf("input %q: Standardize failure", b)
		}
	})
}
