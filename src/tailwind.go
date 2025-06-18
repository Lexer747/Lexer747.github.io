package main

import (
	"bytes"
	"os/exec"
)

func runTailwind() error {
	cmd := exec.Command("tailwindcss")
	cmd.Args = append(cmd.Args,
		"--input", tailwindInput,
		"--output", tailwindOutput,
		"--minify",
	)
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Dir = root
	cmd.CombinedOutput()
	if err := cmd.Run(); err != nil {
		return wrapf(err, "failed to run tailwind\nstdout: %s\nstderr: %s", stdout.String(), stderr.String())
	}
	return nil
}
