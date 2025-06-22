package main

import (
	"bytes"
	"os"
	"os/exec"

	"github.com/Lexer747/Lexer747.github.io/fsutil"
	"github.com/Lexer747/Lexer747.github.io/types"
)

func runTailwind(css types.CSS) error {
	cmdCssInput, err := addGeneratedCss(css)
	if err != nil {
		return err
	}

	cmd := exec.Command("tailwindcss")
	cmd.Args = append(cmd.Args,
		"--input", cmdCssInput,
		"--output", tailwindOutput,
		"--minify",
	)
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		return wrapf(err, "failed to run tailwind\nstdout: %s\nstderr: %s", stdout.String(), stderr.String())
	}
	return nil
}

func addGeneratedCss(css types.CSS) (string, error) {
	inputCSS, err := os.ReadFile(tailwindInput)
	if err != nil {
		return "", wrap(err, "unable to get input css")
	}
	inputCSS = append(inputCSS, []byte("\n\n/* Auto generated Lexer747 Chroma Styles: */\n")...)
	inputCSS = append(inputCSS, css.Data...)
	inputCSS = append(inputCSS, []byte("\n/* End Lexer747 Chroma Styles: */\n")...)

	const cmdCssInput = outputFiles + "input-generated.css"
	f, err := fsutil.NewOutputFile(cmdCssInput)
	if err != nil {
		return "", wrap(err, "unable to make temp css file")
	}
	_, err = f.Write(inputCSS)
	if err != nil {
		return "", wrap(err, "unable to write css file")
	}
	f.Close()
	return cmdCssInput, nil
}
