// Copyright (c) 2021 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hujson

import "testing"

func TestUnmarshal(t *testing.T) {
	var out interface{}
	in := []byte("null\n//\n")
	if err := Unmarshal(in, &out); err != nil {
		t.Error(err)
	}
}
