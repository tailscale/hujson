package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProcessFile(t *testing.T) {
	tests := []struct {
		name  string
		src   string
		want  string
		min   bool
		stand bool
	}{
		{
			name: "formatted",
			src:  "{\n\t\"a\": 1,\n}\n",
			want: "{\n\t\"a\": 1,\n}\n",
		},
		{
			name: "unformatted",
			src:  `{"a":1}`,
			want: "{\"a\": 1}\n",
		},
		{
			name: "minimized",
			src:  `{"a":1}`,
			want: `{"a":1}`,
			min:  true,
		},
		{
			name: "minimize",
			src:  `{"a": 1, /* comment */}`,
			want: `{"a":1}`,
			min:  true,
		},
		{
			name:  "standardized",
			src:   `{"a":1}`,
			want:  `{"a":1}`,
			stand: true,
		},
		{
			name:  "standardize",
			src:   `{"a":1/* comment */}`,
			want:  `{"a":1` + strings.Repeat(" ", len("/* comment */")) + `}`,
			stand: true,
		},
	}
	for _, action := range []string{"list", "write"} {
		for _, tt := range tests {
			t.Run(action+"/"+tt.name, func(t *testing.T) {
				oldMin, oldStand, oldDiff, oldList, oldWrite := *min, *stand, *diff, *list, *write
				t.Cleanup(func() {
					*min, *stand, *diff, *list, *write = oldMin, oldStand, oldDiff, oldList, oldWrite
				})
				*min, *stand, *diff, *list, *write = tt.min, tt.stand, false, action == "list", action == "write"

				dir := t.TempDir()
				filename := filepath.Join(dir, "policy.hujson")
				if err := os.WriteFile(filename, []byte(tt.src), 0644); err != nil {
					t.Fatal(err)
				}
				mtime := time.Unix(1234567890, 0)
				if err := os.Chtimes(filename, mtime, mtime); err != nil {
					t.Fatal(err)
				}
				info, err := os.Stat(filename)
				if err != nil {
					t.Fatal(err)
				}
				stdout, err := os.CreateTemp(t.TempDir(), "stdout")
				if err != nil {
					t.Fatal(err)
				}
				oldStdout := os.Stdout
				os.Stdout = stdout
				t.Cleanup(func() {
					os.Stdout = oldStdout
					stdout.Close()
				})

				if err := processFile(info, filename, nil); err != nil {
					t.Fatal(err)
				}
				gotOutput, err := os.ReadFile(stdout.Name())
				if err != nil {
					t.Fatal(err)
				}
				var wantOutput string
				if action == "list" && tt.src != tt.want {
					wantOutput = filename + "\n"
				}
				if string(gotOutput) != wantOutput {
					t.Errorf("stdout = %q, want %q", gotOutput, wantOutput)
				}
				gotFile, err := os.ReadFile(filename)
				if err != nil {
					t.Fatal(err)
				}
				wantFile := tt.src
				if action == "write" {
					wantFile = tt.want
				}
				if string(gotFile) != wantFile {
					t.Errorf("file = %q, want %q", gotFile, wantFile)
				}
				if action == "list" || tt.src == tt.want {
					after, err := os.Stat(filename)
					if err != nil {
						t.Fatal(err)
					}
					if !after.ModTime().Equal(info.ModTime()) {
						t.Error("unchanged file was rewritten")
					}
				}
			})
		}
	}
}
