package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Lexer747/Lexer747.github.io/types"
)

const (
	root = "/home/lexer747/repos/Lexer747.github.io/"

	inputFiles  = root + "content/"
	outputFiles = root + "build/"

	inputPages  = inputFiles + "pages/"
	outputPages = outputFiles + "pages/"

	tailwindInput  = inputPages + "input.css"
	tailwindOutput = outputPages + "output.css"
)

func main() {
	err := runTemplating()
	if err != nil {
		exit(err)
	}

	err = runMarkdown()
	if err != nil {
		exit(err)
	}

	err = runTailwind()
	if err != nil {
		exit(err)
	}
}

func glob(root, glob string) ([]string, error) {
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
		matched, globErr := filepath.Match(glob, file)
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

func exit(err error) {
	fmt.Fprint(os.Stderr, err.Error()+"\n")
	os.Exit(1)
}

func wrap(err error, msg string) error {
	addedErr := fmt.Sprintf(msg+": %s", err.Error())
	return errors.New(addedErr)
}

func wrapf(err error, msg string, args ...any) error {
	explain := fmt.Sprintf(msg, args...)
	addedErr := fmt.Sprintf(explain+": %s", err.Error())
	return errors.New(addedErr)
}

// Maybe don't make global 🤷‍♂️
var eval = types.Evaluator{Context: map[types.Contexts][]byte{}}
