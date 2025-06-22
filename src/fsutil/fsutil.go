package fsutil

import (
	"io"
	"os"
)

func Copy(input string, output string) error {
	dest, err := NewOutputFile(output)
	defer dest.Close()
	if err != nil {
		return err
	}
	src, err := os.Open(input)
	defer src.Close()
	if err != nil {
		return err
	}
	_, err = io.Copy(dest, src)
	return err
}

func NewOutputFile(dest string) (*os.File, error) {
	return os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0777)
}
