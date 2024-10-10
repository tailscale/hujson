package hujson

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestAll(t *testing.T) {
	v, err := Parse([]byte(`["fizz", {"key": ["value", {"foo": "bar"}]}, [1,2,3], "buzz"]`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	var got []string
	for v2 := range v.All() {
		got = append(got, v2.String())
	}
	want := []string{
		`["fizz", {"key": ["value", {"foo": "bar"}]}, [1,2,3], "buzz"]`,
		`"fizz"`,
		` {"key": ["value", {"foo": "bar"}]}`,
		`"key"`,
		` ["value", {"foo": "bar"}]`,
		`"value"`,
		` {"foo": "bar"}`,
		`"foo"`,
		` "bar"`,
		` [1,2,3]`,
		`1`,
		`2`,
		`3`,
		` "buzz"`,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
