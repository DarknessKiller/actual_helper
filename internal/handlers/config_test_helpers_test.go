package handlers_test

import (
	"errors"
	"io"
	"strings"
)

func stringReader(s string) io.Reader { return strings.NewReader(s) }

// errReader always fails on Read, used to exercise the unreadable-body path.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read boom") }
