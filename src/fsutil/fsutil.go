package fsutil

import (
	"io"
	"os"
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
