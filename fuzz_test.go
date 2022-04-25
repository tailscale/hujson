// Copyright (c) 2021 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
	for _, tt := range testdataFormat {
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

		// Format should produce parsable HuJSON.
		v3 := v.Clone()
		v3.Format()
		b3 := v3.Pack()
		v4, err := Parse(b3)
		if err != nil {
			t.Fatalf("input %q: Parse after Format error: %v", b, err)
		}

		// Format should be idempotent.
		v4.Format()
		b4 := v4.Pack()
		if !bytes.Equal(b3, b4) {
			t.Fatalf("input %q: Format failed to be idempotent: %s", b, cmp.Diff(b3, b4))
		}

		// Format should preserve standard property.
		if v.IsStandard() && !v3.IsStandard() {
			t.Fatalf("input %q: Format failed to remain standard", b)
		}
	})
}
