// Copyright (c) 2021 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hujson

import (
	"io"

	json "github.com/tailscale/hujson/internal/hujson"
)

// Deprecated: Do not use. This will be deleted in the near future.
func Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// Deprecated: Do not use. This will be deleted in the near future.
func MarshalIndent(v interface{}, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(v, prefix, indent)
}

// Deprecated: Do not use. This will be deleted in the near future.
// See the "Use with the Standard Library" section for alternatives.
func NewDecoder(r io.Reader) *Decoder {
	return json.NewDecoder(r)
}

// Deprecated: Do not use. This will be deleted in the near future.
func NewEncoder(w io.Writer) *Encoder {
	return json.NewEncoder(w)
}

// Deprecated: Do not use. This will be deleted in the near future.
// See the "Use with the Standard Library" section for alternatives.
func Unmarshal(data []byte, v interface{}) error {
	ast, err := Parse(data)
	if err != nil {
		return err
	}
	ast.Standardize()
	data = ast.Pack()
	return json.Unmarshal(data, v)
}

// Deprecated: Do not use. This will be deleted in the near future.
// See the "Use with the Standard Library" section for alternatives.
type Decoder = json.Decoder

// Deprecated: Do not use. This will be deleted in the near future.
type Encoder = json.Encoder
