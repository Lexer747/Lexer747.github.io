package fsutil

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func Copy(input string, output string) error {
	dest, err := NewOutputFile(output)
	if err != nil {
		return err
	}
	defer dest.Close()
	src, err := os.Open(input)
	if err != nil {
		return err
	}
	defer src.Close()
	_, err = io.Copy(dest, src)
	return err
}

func NewOutputFile(dest string) (*os.File, error) {
	return os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0777)
}

func CopyDirRecursively(src, dest string) error {
	files, err := Glob(src, "*")
	if err != nil {
		return err
	}
	for _, file := range files {
		output, err := filepath.Rel(src, file)
		if err != nil {
			return err
		}
		fullOutput := dest + output
		err = os.MkdirAll(filepath.Dir(fullOutput), 0777)
		if err != nil {
			return err
		}
		err = Copy(file, fullOutput)
		if err != nil {
			return err
		}
	}
	return nil
}

func Glob(root, globPattern string) ([]string, error) {
	results := []string{}
	dir := os.DirFS(root)
	if dir == nil {
		return results, fmt.Errorf("bad root %q", root)
	}
	fs.WalkDir(dir, ".", func(path string, d fs.DirEntry, err error) error {
		if d == nil || d.IsDir() {
			return nil
		}
		file := filepath.Base(path)
		matched, globErr := filepath.Match(globPattern, file)
		if globErr != nil {
			panic(globErr.Error())
		}
		if matched {
			results = append(results, root+path)
		}
		return nil
	})
	return results, nil
}
