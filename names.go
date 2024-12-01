// Copyright (c) 2021 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hujson

import (
	"reflect"
	"strings"
)

// Names is a map of canonical JSON object names.
// The value for each entry is another map of JSON object names to use
// for any JSON sub-objects.
//
// As a special case, a map with only a single entry where the key is "*"
// indicates that the sub-map of names is to be applied to all sub-objects.
//
// See the example for Value.NormalizeNames for more information.
type Names map[string]Names

// NewNames constructs a Names map for the provided type
// as typically understood by the "encoding/json" package.
//
// See the example for Value.NormalizeNames for more information.
func NewNames(t reflect.Type) Names {
	// TODO(dsnet): Handle cycles in the type graph.
	// TODO(dsnet): What happens when t implements json.Unmarshaler?
	switch t.Kind() {
	case reflect.Array, reflect.Slice, reflect.Ptr:
		return NewNames(t.Elem())
	case reflect.Map:
		names := NewNames(t.Elem())
		if len(names) == 0 {
			return nil
		}
		return Names{"*": names}
	case reflect.Struct:
		names := make(Names)
		for i := 0; i < t.NumField(); i++ {
			sf := t.Field(i)
			if sf.PkgPath != "" {
				// TODO(dsnet): Technically, an embedded, unexported type with
				// exported fields can have serializable fields.
				// This almost never occurs in practice.
				continue // unexported fields are ignored
			}

			// Derive JSON name from either the Go field name or `json` tag.
			name := sf.Name
			inlined := sf.Anonymous && mayIndirect(sf.Type).Kind() == reflect.Struct
			switch tag := sf.Tag.Get("json"); tag {
			case "":
				break // do nothing
			case "-":
				continue // explicitly ignored field
			default:
				if i := strings.IndexByte(tag, ','); i >= 0 {
					tag = tag[:i]
				}
				if tag != "" {
					name = tag
					inlined = false // explicitly named fields are never inlined
				}
			}

			// If inlined, hoist all child names up to the parent.
			// Otherwise, just insert the current name.
			if inlined {
				// TODO(dsnet): This does not properly handle name conflicts.
				// However, conflicts rarely occur in practice.
				// See https://github.com/golang/go/blob/aa4e0f528e1e018e2847decb549cfc5ac07ecf20/src/encoding/json/encode.go#L1352-L1378
				for name, subNames := range NewNames(sf.Type) {
					names[name] = subNames
				}
			} else {
				names[name] = NewNames(sf.Type)
			}
		}
		if len(names) == 0 {
			return nil
		}
		return names
	default:
		return nil
	}
}

func mayIndirect(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// NormalizeNames recursively iterates through v and replaces any JSON object
// names that is a case-insensitive match with a name found in names,
// with the canonical name found in names.
//
// See the example for Value.NormalizeNames for more information.
func (v *Value) NormalizeNames(names Names) {
	v.normalizeNames(names)
	v.UpdateOffsets()
}
func (v *Value) normalizeNames(names Names) {
	if len(names) == 0 {
		return
	}
	switch v2 := v.Value.(type) {
	case *Object:
		// If names is a map with only a "*" key,
		// then apply the same subNames map to all map values.
		if subNames, ok := names["*"]; ok && len(names) == 1 {
			for i := range v2.Members {
				v2.Members[i].Value.normalizeNames(subNames)
			}
			break
		}

		for i := range v2.Members {
			name := v2.Members[i].Name.Value.(Literal).String()

			// Fast-path: Exact match with names map.
			subNames, ok := names[name]
			if !ok {
				// Slow-path: Case-insensitive match with names map.
				var match string
				for name2 := range names {
					if (match == "" || match < name2) && strings.EqualFold(name, name2) {
						match = name2
					}
				}
				// If a case-insensitive match was found, update the name.
				if match != "" {
					v2.Members[i].Name.Value = String(match)
					subNames = names[match]
				}
			}
			v2.Members[i].Value.normalizeNames(subNames)
		}
	case *Array:
		for i := range v2.Elements {
			v2.Elements[i].normalizeNames(names)
		}
	}
}
