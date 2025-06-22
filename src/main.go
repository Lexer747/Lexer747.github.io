package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/Lexer747/Lexer747.github.io/fsutil"
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
	err := preTemplating()
	if err != nil {
		exit(err)
	}

	err = runTemplating()
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
var eval = types.Evaluator{Context: map[types.Contexts]any{}}

func preTemplating() error {
	contexts, err := glob(inputFiles, "*.context")
	if err != nil {
		return wrap(err, "failed to get contexts files")
	}
	for _, context := range contexts {
		name := strings.Split(filepath.Base(context), ".context")[0]
		if name == string(types.MarkdownContext) {
			file, err := os.ReadFile(context)
			if err != nil {
				return wrap(err, "failed to get markdown.context files")
			}
			fixture := &Fixture{
				SrcPath: context,
				File:    file,
			}
			fixture.Parse()
			eval.Context[types.MarkdownContext] = fixture
		}
	}
	favicons, err := glob(inputFiles, "*.ico")
	if err != nil {
		return wrap(err, "failed to get contexts files")
	}
	if len(favicons) == 1 {
		srcPath := favicons[0]
		file := filepath.Base(srcPath)
		destFile := outputPages + file
		err := fsutil.Copy(srcPath, destFile)
		if err != nil {
			return wrap(err, "failed to get favicon")
		}
		eval.Context[types.FaviconContext] = destFile
	} else {
		slog.Warn("unexpected number of favicons, not choosing any", "favicons", favicons)
	}
	return nil
}
