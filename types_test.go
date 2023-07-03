package hujson_test

import (
	"encoding/json"
	"fmt"

	"github.com/tailscale/hujson"
)

func Example() {
	humanBytes := []byte(`{
		"author": "Strugatsky, Arkady and Boris",
		"title":  "The Time Wanderers",
		// my comment
}`)

	jsonBytes, err := hujson.Standardize(humanBytes)
	if err != nil {
		panic(err)
	}

	result := map[string]any{}

	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		panic(err)
	}

	fmt.Println(result["author"])

	// Output:
	// Strugatsky, Arkady and Boris
}
