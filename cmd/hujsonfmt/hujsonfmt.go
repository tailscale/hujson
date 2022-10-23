package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/tailscale/hujson"
)

var (
	min   = flag.Bool("m", false, "minify results")
	stand = flag.Bool("s", false, "standardize results to plain JSON")
	diff  = flag.Bool("d", false, "display diffs instead of rewriting files")
	list  = flag.Bool("l", false,
		"list files whose formatting differs from hujsonfmt's",
	)
	write = flag.Bool("w", false,
		"write result to (source) file instead of stdout",
	)

	chmodSupported = runtime.GOOS != "windows"
	huJSONExt      = ".hujson"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: hujsonfmt [flags] [path ...]\n")
	flag.PrintDefaults()
}

func main() {
	err := mainE()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		usage()
		os.Exit(1)
	}
}

func mainE() error {
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()

	if len(args) == 0 || (len(args) == 1 && args[0] == "-") {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			return fmt.Errorf("no files paths or stdin provided")
		}
		if *write {
			return fmt.Errorf("cannot use -w with standard input")
		}

		return processFile(nil, "<standard input>", os.Stdin)
	}

	for _, arg := range args {
		info, err := os.Stat(arg)
		switch {
		case err != nil:
			return err
		case !info.IsDir():
			err := processFile(info, arg, nil)
			if err != nil {
				return err
			}
		default:
			err := filepath.WalkDir(
				arg,
				func(path string, f fs.DirEntry, err error) error {
					if err != nil || !isHuJSONFile(f) {
						return err
					}

					return processFile(info, path, nil)
				},
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func isHuJSONFile(f fs.DirEntry) bool {
	return strings.HasSuffix(f.Name(), huJSONExt) && !f.IsDir()
}

func processFile(info fs.FileInfo, filename string, in io.Reader) error {
	src, err := readFile(filename, in)
	if err != nil {
		return err
	}

	// The main hujson functions will sometimes modify the original input byte
	// slice. Hence we create a copy of the src byte slice to avoid modifying
	// src, enabling us to reliably print diffs.
	input := make([]byte, len(src))
	_ = copy(input, src)

	output, err := processSrc(input)
	if err != nil {
		return err
	}

	switch {
	case *diff:
		printDiff(filename, src, output)
	case *list:
		fmt.Println(filename)
	case *write:
		err = writeFile(info, filename, src, output)
		if err != nil {
			return err
		}
	default:
		fmt.Print(string(output))
	}

	return nil
}

func readFile(path string, in io.Reader) ([]byte, error) {
	if in == nil {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		in = f
	}

	src, err := io.ReadAll(in)
	if err != nil {
		return nil, err
	}

	return src, nil
}

func processSrc(src []byte) ([]byte, error) {
	var r []byte
	var err error
	switch {
	case *min:
		r, err = hujson.Minimize(src)
	case *stand:
		r, err = hujson.Standardize(src)
	default:
		r, err = hujson.Format(src)
	}
	if err != nil {
		return nil, err
	}

	return r, nil
}

func printDiff(filename string, src, modified []byte) {
	origFile := filename + ".orig"
	old := string(src)
	new := string(modified)
	edits := myers.ComputeEdits(
		span.URIFromPath(origFile), old, new,
	)
	diff := fmt.Sprint(
		gotextdiff.ToUnified(origFile, filename, old, edits),
	)

	if diff == "" {
		return
	}

	fmt.Printf("diff %s %s\n", origFile, filename)
	fmt.Println(diff)
}

func writeFile(info fs.FileInfo, filename string, src, data []byte) error {
	if info == nil {
		panic("-w should not have been allowed with standard input")
	}

	perms := info.Mode().Perm()

	var bak string
	bak, err := backupFile(filename, src, perms)
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, data, perms)
	if err != nil {
		_ = os.Rename(bak, filename)

		return err
	}

	err = os.Remove(bak)
	if err != nil {
		return err
	}

	return nil
}

func backupFile(
	filename string,
	data []byte,
	perms fs.FileMode,
) (backupFile string, err error) {
	var f *os.File
	f, err = os.CreateTemp(filepath.Dir(filename), filepath.Base(filename))
	if err != nil {
		return "", err
	}
	defer f.Close()

	backupFile = f.Name()

	if chmodSupported {
		err = f.Chmod(perms)
		if err != nil {
			_ = os.Remove(backupFile)

			return "", err
		}
	}

	_, err = f.Write(data)
	if err != nil {
		_ = os.Remove(backupFile)

		return "", err
	}

	return backupFile, nil
}
