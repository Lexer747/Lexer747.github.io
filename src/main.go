package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/Lexer747/Lexer747.github.io/fsutil"
	"github.com/Lexer747/Lexer747.github.io/types"
)

var (
	root           string
	inputFiles     string
	outputFiles    string
	inputPages     string
	outputPages    string
	tailwindInput  string
	tailwindOutput string
)

const (
	siteUrl = "https://lexer747.github.io/"
	homeUrl = siteUrl + "index.html"
)

func setup(newRoot string) {
	root = newRoot

	inputFiles = root + "content/"
	outputFiles = root + "build/"

	inputPages = inputFiles + "pages/"
	outputPages = outputFiles

	tailwindInput = inputPages + "input.css"
	tailwindOutput = outputPages + "output.css"
}

func main() {
	if len(os.Args) < 2 {
		setup("/home/lexer747/repos/Lexer747.github.io/")
	} else {
		newRoot := os.Args[1]
		if !strings.HasSuffix(newRoot, "/") {
			newRoot += "/"
		}
		setup(newRoot)
	}
	slog.Info("Variables",
		"root", root,
		"inputFiles", inputFiles,
		"outputFiles", outputFiles,
		"inputPages", inputPages,
		"outputPages", outputPages,
		"tailwindInput", tailwindInput,
		"tailwindOutput", tailwindOutput,
	)

	err := makeOutputDir()
	if err != nil {
		exit(err)
	}

	blogs, err := getBlogs()
	if err != nil {
		exit(err)
	}

	err = preTemplating()
	if err != nil {
		exit(err)
	}

	err = runTemplating(blogs)
	if err != nil {
		exit(err)
	}

	css, err := runMarkdown(blogs)
	if err != nil {
		exit(err)
	}

	err = runTailwind(css)
	if err != nil {
		exit(err)
	}

	// TODO delete partial generated files
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
	contexts, err := fsutil.Glob(inputFiles, "*.context")
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
	favicons, err := fsutil.Glob(inputFiles, "*.ico")
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

func makeOutputDir() error {
	return os.MkdirAll(outputPages, 0777)
}
